import type { Asset } from '../state/model'
import type { Edit } from '../state/operations'
import type { ProjectState } from './client'

export type CursorPos = { x: number; y: number }

export type PeerPresence = {
  clientId: string
  cursor?: CursorPos | null
  playhead?: number
  selected?: string | null
}

export type WSInMessage =
  | { type: 'init'; client_id: string; current_seq: number; state?: ProjectState }
  | { type: 'peer_joined'; client_id: string }
  | { type: 'peer_left'; client_id: string }
  | { type: 'op_ack'; req_id: string; seq: number; status: 'ok'; state?: ProjectState }
  | { type: 'op_broadcast'; seq: number; op?: Edit; author?: string; state?: ProjectState }
  | { type: 'op_error'; req_id: string; code: 'conflict' | 'invalid'; current_seq?: number; message?: string }
  | { type: 'presence_broadcast'; client_id: string; cursor?: CursorPos | null; playhead?: number; selected?: string | null }
  | { type: 'asset_updated'; asset: Asset }
  | { type: 'asset_deleted'; asset_id: string }
  | { type: 'sync_resp'; current_seq: number; ops?: unknown[]; state?: ProjectState }

export type WSOutMessage =
  | { type: 'op_submit'; req_id: string; op: Edit; expected_revision: number }
  | { type: 'presence'; cursor?: CursorPos | null; playhead?: number; selected?: string | null }
  | { type: 'sync_req'; last_seen_seq: number }

type PendingOp = {
  resolve: (state: ProjectState) => void
  reject: (error: Error) => void
  timer: ReturnType<typeof setTimeout>
}

export type SocketHandlers = {
  onState?: (state: ProjectState) => void
  onPeerPresence?: (presence: PeerPresence | null) => void
  onAssetUpdated?: (asset: Asset) => void
  onAssetDeleted?: (assetId: string) => void
  onConnectionChange?: (connected: boolean) => void
  getRevision?: () => number
}

class ProjectSocket {
  private ws: WebSocket | null = null
  private pendingOps = new Map<string, PendingOp>()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private presenceTimer: ReturnType<typeof setTimeout> | null = null
  private queuedPresence: { cursor?: CursorPos | null; playhead?: number; selected?: string | null } | null = null
  private isExplicitClose = false
  private handlers: SocketHandlers = {}
  public clientId: string | null = null
  public isConnected = false

  setHandlers(handlers: SocketHandlers) {
    this.handlers = { ...this.handlers, ...handlers }
  }

  connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    this.isExplicitClose = false
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/api/project/ws`

    try {
      this.ws = new WebSocket(url)
    } catch {
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.isConnected = true
      this.handlers.onConnectionChange?.(true)
      const currentRev = this.handlers.getRevision?.() ?? 0
      if (currentRev > 0) {
        this.send({ type: 'sync_req', last_seen_seq: currentRev })
      }
      this.send({ type: 'presence', playhead: 0 })
    }

    this.ws.onmessage = event => {
      const text = typeof event.data === 'string' ? event.data : ''
      const lines = text.split('\n').filter(line => line.trim().length > 0)
      for (const line of lines) {
        try {
          const msg: WSInMessage = JSON.parse(line)
          this.handleMessage(msg)
        } catch {
          // Ignore unparseable frames
        }
      }
    }

    this.ws.onclose = event => {
      this.isConnected = false
      this.handlers.onConnectionChange?.(false)
      this.cleanupPendingOps(new Error('Connection lost.'))
      if (!this.isExplicitClose && event.code !== 1008) {
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = () => {
      this.isConnected = false
      this.handlers.onConnectionChange?.(false)
    }
  }

  disconnect() {
    this.isExplicitClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.isConnected = false
    this.handlers.onConnectionChange?.(false)
  }

  sendOp(id: string, expectedRevision: number, edit: Edit): Promise<ProjectState> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        return reject(new Error('WebSocket is not connected.'))
      }

      const timer = setTimeout(() => {
        this.pendingOps.delete(id)
        reject(new Error('Operation confirmation timed out.'))
      }, 10000)

      this.pendingOps.set(id, { resolve, reject, timer })
      const { type, ...payload } = edit
      this.send({
        type: 'op_submit',
        req_id: id,
        op: {
          id,
          type,
          payload,
          expectedRevision,
        } as unknown as Edit,
        expected_revision: expectedRevision,
      })
    })
  }

  sendPresence(presence: { cursor?: CursorPos | null; playhead?: number; selected?: string | null }) {
    this.queuedPresence = { ...this.queuedPresence, ...presence }
    if (this.presenceTimer) return

    this.presenceTimer = setTimeout(() => {
      this.presenceTimer = null
      if (this.queuedPresence && this.ws?.readyState === WebSocket.OPEN) {
        this.send({ type: 'presence', ...this.queuedPresence })
        this.queuedPresence = null
      }
    }, 50) // Throttled to ~20Hz
  }

  private send(msg: WSOutMessage) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
    }
  }

  private handleMessage(msg: WSInMessage) {
    switch (msg.type) {
      case 'init':
        this.clientId = msg.client_id
        if (msg.state) {
          this.handlers.onState?.(msg.state)
        }
        break

      case 'peer_joined':
        this.handlers.onPeerPresence?.({
          clientId: msg.client_id,
        })
        break

      case 'op_ack': {
        const pending = this.pendingOps.get(msg.req_id)
        if (pending) {
          clearTimeout(pending.timer)
          this.pendingOps.delete(msg.req_id)
          if (msg.state) {
            this.handlers.onState?.(msg.state)
            pending.resolve(msg.state)
          } else {
            pending.resolve(null as unknown as ProjectState)
          }
        } else if (msg.state) {
          this.handlers.onState?.(msg.state)
        }
        break
      }

      case 'op_error': {
        const pending = this.pendingOps.get(msg.req_id)
        if (pending) {
          clearTimeout(pending.timer)
          this.pendingOps.delete(msg.req_id)
          pending.reject(new Error(msg.message || 'Operation failed.'))
        }
        break
      }

      case 'op_broadcast':
        if (msg.state) {
          this.handlers.onState?.(msg.state)
        }
        break

      case 'presence_broadcast':
        this.handlers.onPeerPresence?.({
          clientId: msg.client_id,
          cursor: msg.cursor,
          playhead: msg.playhead,
          selected: msg.selected ?? null,
        })
        break

      case 'peer_left':
        this.handlers.onPeerPresence?.(null)
        break

      case 'asset_updated':
        this.handlers.onAssetUpdated?.(msg.asset)
        break

      case 'asset_deleted':
        this.handlers.onAssetDeleted?.(msg.asset_id)
        break

      case 'sync_resp':
        if (msg.state) {
          this.handlers.onState?.(msg.state)
        }
        break
    }
  }

  private cleanupPendingOps(error: Error) {
    for (const [, pending] of this.pendingOps) {
      clearTimeout(pending.timer)
      pending.reject(error)
    }
    this.pendingOps.clear()
  }

  private scheduleReconnect() {
    if (this.reconnectTimer || this.isExplicitClose) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, 2000)
  }
}

export const projectSocket = new ProjectSocket()
