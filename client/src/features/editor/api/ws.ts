import type { Asset } from '../state/model'
import type { Edit } from '../state/operations'
import type { ConsensusProposal, PermissionRequest } from '../state/ownership'
import type { ProjectState } from './client'

export type CursorPos = { x: number; y: number }

export type PeerPresence = {
  clientId: string
  cursor?: CursorPos | null
  playhead?: number
  selected?: string | null
}

export type WSInMessage =
  | { type: 'init'; client_id: string; current_seq: number; state?: ProjectState; owners?: Record<string, string> }
  | { type: 'peer_joined'; client_id: string }
  | { type: 'peer_left'; client_id: string }
  | { type: 'op_ack'; req_id: string; seq: number; status: 'ok'; state?: ProjectState }
  | { type: 'op_broadcast'; seq: number; op?: Edit; author?: string; state?: ProjectState }
  | { type: 'op_error'; req_id: string; code: 'conflict' | 'invalid'; current_seq?: number; message?: string }
  | { type: 'presence_broadcast'; client_id: string; cursor?: CursorPos | null; playhead?: number; selected?: string | null }
  | { type: 'asset_updated'; asset: Asset }
  | { type: 'asset_deleted'; asset_id: string }
  | { type: 'sync_resp'; current_seq: number; ops?: unknown[]; state?: ProjectState }
  | { type: 'claim_ack'; req_id: string; granted?: string[]; rejected?: string[] }
  | { type: 'claim_broadcast'; client_id: string; clip_ids?: string[] }
  | { type: 'release_broadcast'; clip_ids?: string[] }
  | { type: 'permission_prompt'; req_id: string; from_client_id: string; clip_id: string; timeout_ms: number }
  | { type: 'permission_resolved'; req_id: string; clip_id: string; approved: boolean; new_owner?: string | null }
  | { type: 'consensus_prompt'; proposal_id: string; proposer_id: string; op: Edit; description: string; timeout_ms: number }
  | { type: 'consensus_resolved'; proposal_id: string; approved: boolean; op?: Edit; message?: string }

export type WSOutMessage =
  | { type: 'op_submit'; req_id: string; op: Edit; expected_revision: number }
  | { type: 'presence'; cursor?: CursorPos | null; playhead?: number; selected?: string | null }
  | { type: 'sync_req'; last_seen_seq: number }
  | { type: 'claim_request'; req_id: string; clip_ids: string[] }
  | { type: 'release_request'; clip_ids: string[] }
  | { type: 'permission_request'; req_id: string; clip_id: string }
  | { type: 'permission_response'; req_id: string; approved: boolean }
  | { type: 'consensus_propose'; proposal_id: string; op: Edit; description: string }
  | { type: 'consensus_vote'; proposal_id: string; approved: boolean }

type PendingOp = {
  resolve: (state: ProjectState) => void
  reject: (error: Error) => void
  timer: ReturnType<typeof setTimeout>
}

type PendingPromise<T> = {
  resolve: (val: T) => void
  timer: ReturnType<typeof setTimeout>
  meta?: unknown
}

export type SocketHandlers = {
  onState?: (state: ProjectState) => void
  onPeerPresence?: (presence: PeerPresence | null) => void
  onAssetUpdated?: (asset: Asset) => void
  onAssetDeleted?: (assetId: string) => void
  onConnectionChange?: (connected: boolean) => void
  getRevision?: () => number
  onInitClientId?: (clientId: string, owners?: Record<string, string>) => void
  onClaimBroadcast?: (clientId: string, clipIds: string[]) => void
  onReleaseBroadcast?: (clipIds: string[]) => void
  onPermissionPrompt?: (prompt: PermissionRequest) => void
  onPermissionResolved?: (res: { reqId: string; clipId: string; approved: boolean; newOwner?: string | null }) => void
  onConsensusPrompt?: (proposal: ConsensusProposal) => void
  onConsensusResolved?: (res: { proposalId: string; approved: boolean; op?: Edit; message?: string }) => void
}

class ProjectSocket {
  private ws: WebSocket | null = null
  private pendingOps = new Map<string, PendingOp>()
  private pendingClaims = new Map<string, PendingPromise<boolean>>()
  private pendingPermissions = new Map<string, PendingPromise<boolean>>()
  private pendingConsensus = new Map<string, PendingPromise<boolean>>()
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
    const query = this.clientId ? `?clientId=${encodeURIComponent(this.clientId)}` : ''
    const url = `${protocol}//${window.location.host}/api/project/ws${query}`

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

  claimClips(reqId: string, clipIds: string[]): Promise<boolean> {
    return new Promise(resolve => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        return resolve(false)
      }
      const timer = setTimeout(() => {
        this.pendingClaims.delete(reqId)
        resolve(false)
      }, 5000)
      this.pendingClaims.set(reqId, { resolve, timer, meta: clipIds })
      this.send({
        type: 'claim_request',
        req_id: reqId,
        clip_ids: clipIds,
      })
    })
  }

  releaseClips(clipIds: string[]) {
    this.send({
      type: 'release_request',
      clip_ids: clipIds,
    })
  }

  requestPermission(reqId: string, clipId: string): Promise<boolean> {
    return new Promise(resolve => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        return resolve(false)
      }
      const timer = setTimeout(() => {
        this.pendingPermissions.delete(reqId)
        resolve(false)
      }, 35000)
      this.pendingPermissions.set(reqId, { resolve, timer })
      this.send({
        type: 'permission_request',
        req_id: reqId,
        clip_id: clipId,
      })
    })
  }

  respondPermission(reqId: string, approved: boolean) {
    this.send({
      type: 'permission_response',
      req_id: reqId,
      approved,
    })
  }

  proposeConsensus(proposalId: string, op: Edit, description: string): Promise<boolean> {
    return new Promise(resolve => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        return resolve(false)
      }
      const timer = setTimeout(() => {
        this.pendingConsensus.delete(proposalId)
        resolve(false)
      }, 35000)
      this.pendingConsensus.set(proposalId, { resolve, timer })
      const { type, ...payload } = op as Record<string, unknown>
      this.send({
        type: 'consensus_propose',
        proposal_id: proposalId,
        op: {
          id: proposalId,
          type,
          payload,
        } as unknown as Edit,
        description,
      })
    })
  }

  voteConsensus(proposalId: string, approved: boolean) {
    this.send({
      type: 'consensus_vote',
      proposal_id: proposalId,
      approved,
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
    }, 50)
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
        this.handlers.onInitClientId?.(msg.client_id, msg.owners)
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

      case 'claim_ack': {
        const pending = this.pendingClaims.get(msg.req_id)
        if (pending) {
          clearTimeout(pending.timer)
          this.pendingClaims.delete(msg.req_id)
          const requested = (pending.meta as string[]) ?? []
          const allGranted = requested.every(id => msg.granted?.includes(id))
          pending.resolve(allGranted)
        }
        break
      }

      case 'claim_broadcast':
        if (msg.client_id && msg.clip_ids) {
          this.handlers.onClaimBroadcast?.(msg.client_id, msg.clip_ids)
        }
        break

      case 'release_broadcast':
        if (msg.clip_ids) {
          this.handlers.onReleaseBroadcast?.(msg.clip_ids)
        }
        break

      case 'permission_prompt':
        this.handlers.onPermissionPrompt?.({
          reqId: msg.req_id,
          clipId: msg.clip_id,
          fromClientId: msg.from_client_id,
          timeoutMs: msg.timeout_ms || 30000,
          expiresAt: Date.now() + (msg.timeout_ms || 30000),
        })
        break

      case 'permission_resolved': {
        const pending = this.pendingPermissions.get(msg.req_id)
        if (pending) {
          clearTimeout(pending.timer)
          this.pendingPermissions.delete(msg.req_id)
          pending.resolve(msg.approved)
        }
        this.handlers.onPermissionResolved?.({
          reqId: msg.req_id,
          clipId: msg.clip_id,
          approved: msg.approved,
          newOwner: msg.new_owner,
        })
        break
      }

      case 'consensus_prompt':
        this.handlers.onConsensusPrompt?.({
          proposalId: msg.proposal_id,
          proposerId: msg.proposer_id,
          op: msg.op,
          description: msg.description,
          timeoutMs: msg.timeout_ms || 30000,
          expiresAt: Date.now() + (msg.timeout_ms || 30000),
        })
        break

      case 'consensus_resolved': {
        const pending = this.pendingConsensus.get(msg.proposal_id)
        if (pending) {
          clearTimeout(pending.timer)
          this.pendingConsensus.delete(msg.proposal_id)
          pending.resolve(msg.approved)
        }
        this.handlers.onConsensusResolved?.({
          proposalId: msg.proposal_id,
          approved: msg.approved,
          op: msg.op,
          message: msg.message,
        })
        break
      }
    }
  }

  private cleanupPendingOps(error: Error) {
    for (const [, pending] of this.pendingOps) {
      clearTimeout(pending.timer)
      pending.reject(error)
    }
    this.pendingOps.clear()
    for (const [, pending] of this.pendingClaims) {
      clearTimeout(pending.timer)
      pending.resolve(false)
    }
    this.pendingClaims.clear()
    for (const [, pending] of this.pendingPermissions) {
      clearTimeout(pending.timer)
      pending.resolve(false)
    }
    this.pendingPermissions.clear()
    for (const [, pending] of this.pendingConsensus) {
      clearTimeout(pending.timer)
      pending.resolve(false)
    }
    this.pendingConsensus.clear()
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
