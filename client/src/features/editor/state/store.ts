import { create } from 'zustand'
import { commitEdit, getProject, projectSocket, setActiveProjectId, getActiveProjectId, type PeerPresence, type ProjectState } from '../api/index.ts'
import { emptyProject, MAX_FRAMES, trackKind, type Asset, type Media } from './model'
import type { Edit } from './operations'
import type { OwnershipSlice } from './ownership'

type Editor = ProjectState &
  OwnershipSlice & {
    ready: boolean
    connected: boolean
    selected: string | null
    error: string | null
    notice: string | null
    peerPresence: PeerPresence | null
    initialize: (projectId?: string) => Promise<void>
    reset: () => void
    dispatch: (edit: Edit) => Promise<boolean>
    select: (id: string | null) => void
    putAsset: (asset: Asset) => void
    removeAsset: (assetId: string) => void
    setConnected: (connected: boolean) => void
    setNotice: (notice: string | null) => void
    setPeerPresence: (presence: PeerPresence | null) => void
    adopt: (state: ProjectState, selection?: string | null) => void
    add: (media: Media) => void
    split: (frame: number) => void
    remove: () => void
    undo: () => void
    redo: () => void
  }

let queue: Promise<unknown> = Promise.resolve()
let opening: Promise<void> | undefined
let needsRefresh = false

export function serialize<T>(action: () => Promise<T>): Promise<T> {
  const next = queue.then(action)
  queue = next.catch(() => {})
  return next
}

export const errorMessage = (error: unknown) =>
  error instanceof Error ? error.message : 'The request could not be completed.'

export const useEditor = create<Editor>((set, get) => {
  function adopt(state: ProjectState, selection = get().selected) {
    if (get().ready && state.revision < get().revision) {
      return
    }
    set({
      ...state,
      ready: true,
      selected: state.project.clips.some(clip => clip.id === selection) ? selection : null,
    })
  }

  // Register WebSocket handlers
  projectSocket.setHandlers({
    onState: state => adopt(state),
    onPeerPresence: presence => set({ peerPresence: presence }),
    onAssetUpdated: asset => get().putAsset(asset),
    onAssetDeleted: assetId => get().removeAsset(assetId),
    onConnectionChange: connected => set({ connected }),
    getRevision: () => get().revision,
    onInitClientId: (clientId, initialOwners) =>
      set(s => ({
        myClientId: clientId,
        owners: initialOwners ? { ...s.owners, ...initialOwners } : s.owners,
      })),
    onClaimBroadcast: (clientId, clipIds) =>
      set(s => {
        const next = { ...s.owners }
        for (const id of clipIds) {
          next[id] = clientId
        }
        return { owners: next }
      }),
    onReleaseBroadcast: clipIds =>
      set(s => {
        const next = { ...s.owners }
        for (const id of clipIds) {
          delete next[id]
        }
        return { owners: next }
      }),
    onPermissionPrompt: prompt => set({ incomingPermission: prompt }),
    onPermissionResolved: res =>
      set(s => {
        const wasMyRequest = s.outgoingPermission?.reqId === res.reqId
        const targetClip = s.project.clips.find(c => c.id === res.clipId)
        const clipName = targetClip ? s.project.assets[targetClip.mediaId]?.name || 'Clip' : 'Clip'

        return {
          outgoingPermission: wasMyRequest ? null : s.outgoingPermission,
          incomingPermission: s.incomingPermission?.reqId === res.reqId ? null : s.incomingPermission,
          notice:
            wasMyRequest && !res.approved
              ? `Permission denied: Peer declined your request to edit "${clipName}".`
              : s.notice,
        }
      }),
    onConsensusPrompt: proposal => set({ activeConsensus: proposal }),
    onConsensusResolved: res =>
      set(s => {
        const wasMyProposal =
          s.activeConsensus?.proposalId === res.proposalId && s.activeConsensus?.proposerId === s.myClientId
        const desc = s.activeConsensus?.description || 'make structural changes'
        let notice = s.notice
        if (wasMyProposal && !res.approved) {
          if (res.message) {
            notice = `Consensus failed for "${desc}": ${res.message}`
          } else {
            notice = `Consensus rejected: Peer declined the proposal to ${desc}.`
          }
        }

        return {
          activeConsensus: s.activeConsensus?.proposalId === res.proposalId ? null : s.activeConsensus,
          notice,
        }
      }),
  })

  return {
    id: '',
    revision: 0,
    project: emptyProject(),
    canUndo: false,
    canRedo: false,
    ready: false,
    connected: false,
    selected: null,
    error: null,
    notice: null,
    peerPresence: null,

    // Ownership state
    owners: {},
    myClientId: null,
    incomingPermission: null,
    outgoingPermission: null,
    activeConsensus: null,

    adopt,
    setConnected: connected => set({ connected }),
    setNotice: notice => set({ notice }),
    setPeerPresence: peerPresence =>
      set(state => ({
        peerPresence:
          peerPresence === null
            ? null
            : {
                ...state.peerPresence,
                ...peerPresence,
              },
      })),
    initialize: (projectId?: string) => {
      if (projectId) {
        setActiveProjectId(projectId)
      }
      const activeId = projectId || getActiveProjectId()
      projectSocket.connect(activeId || undefined)
      if (get().ready && get().id === activeId) return Promise.resolve()
      if (opening) return opening
      set({ error: null, ready: false })
      opening = getProject(activeId || undefined)
        .then(state => adopt(state))
        .catch((error: unknown) => set({ error: errorMessage(error) }))
        .finally(() => {
          opening = undefined
        })
      return opening
    },
    reset: () => {
      projectSocket.close()
      setActiveProjectId('')
      set({
        id: '',
        revision: 0,
        project: emptyProject(),
        canUndo: false,
        canRedo: false,
        ready: false,
        connected: false,
        selected: null,
        error: null,
        notice: null,
        peerPresence: null,
        owners: {},
        myClientId: null,
        incomingPermission: null,
        outgoingPermission: null,
        activeConsensus: null,
      })
    },
    dispatch: edit =>
      serialize(async () => {
        if (get().peerPresence) {
          if (edit.type === 'add-track') {
            const kindLabel = edit.kind === 'audio' ? 'Audio' : 'Video'
            return get().proposeConsensus(edit, `Add ${kindLabel} Track`)
          }
          if (edit.type === 'delete-track') {
            return get().proposeConsensus(edit, 'Delete Track')
          }
          if (edit.type === 'reset') {
            return get().proposeConsensus(edit, 'Reset Timeline')
          }
        }
        const id = crypto.randomUUID()
        try {
          if (!get().ready) return false
          if (!get().connected) {
            set({ error: 'Disconnected from server. Reconnecting…' })
            return false
          }
          if (needsRefresh) {
            adopt(await getProject(get().id))
            needsRefresh = false
          }

          let state: ProjectState
          try {
            state = await projectSocket.sendOp(id, get().revision, edit)
          } catch {
            // Fallback to HTTP if WS sendOp encounters an error
            state = await commitEdit(id, get().revision, edit, get().id)
          }

          const selected =
            edit.type === 'add'
              ? edit.clip.id
              : edit.type === 'split'
                ? edit.rightId
                : edit.type === 'undo' || edit.type === 'redo' || edit.type === 'reset'
                  ? null
                  : get().selected
          adopt(state, selected)
          set({ error: null })
          return true
        } catch (error) {
          needsRefresh = true
          try {
            adopt(await getProject(get().id))
            needsRefresh = false
          } catch {
            /* Resync before the next edit. */
          }
          set({ error: errorMessage(error) })
          return false
        }
      }),
    select: id => {
      set({ selected: id, error: null })
      if (id) {
        const { owners, myClientId, claimClips } = get()
        if (!owners[id] && myClientId) {
          void claimClips([id])
        }
      }
    },
    putAsset: asset =>
      set(({ project }) => ({
        project: { ...project, assets: { ...project.assets, [asset.id]: asset } },
      })),
    removeAsset: assetId =>
      set(({ project }) => {
        const nextAssets = { ...project.assets }
        delete nextAssets[assetId]
        return { project: { ...project, assets: nextAssets } }
      }),
    add: media => {
      const { project, selected, dispatch } = get()
      const kind = trackKind(media.kind)
      const selectedTrack = project.clips.find(clip => clip.id === selected)?.trackId
      const track =
        project.tracks.find(item => item.id === selectedTrack && item.kind === kind) ??
        project.tracks.find(item => item.kind === kind)
      if (!track) return
      const at = project.clips.reduce(
        (end, clip) => (clip.trackId === track.id ? Math.max(end, clip.at + clip.length) : end),
        0,
      )
      void dispatch({
        type: 'add',
        clip: {
          id: crypto.randomUUID(),
          trackId: track.id,
          mediaId: media.id,
          at,
          offset: 0,
          length: Math.min(media.frames, MAX_FRAMES - at),
          volume: 1,
        },
      })
    },
    split: frame => {
      const { selected, owners, myClientId, dispatch } = get()
      if (selected) {
        const owner = owners[selected]
        if (owner && myClientId && owner !== myClientId) {
          set({ error: 'Cannot split clip owned by peer. Request permission first.' })
          return
        }
        void dispatch({ type: 'split', clipId: selected, frame, rightId: crypto.randomUUID() })
      }
    },
    remove: () => {
      const { selected, owners, myClientId, dispatch } = get()
      if (selected) {
        const owner = owners[selected]
        if (owner && myClientId && owner !== myClientId) {
          set({ error: 'Cannot delete clip owned by peer. Request permission first.' })
          return
        }
        void dispatch({ type: 'delete', clipId: selected })
      }
    },
    undo: () => {
      if (get().canUndo) void get().dispatch({ type: 'undo' })
    },
    redo: () => {
      if (get().canRedo) void get().dispatch({ type: 'redo' })
    },

    // Ownership actions
    claimClips: async (clipIds: string[]) => {
      if (!clipIds.length) return true
      const reqId = crypto.randomUUID()
      return projectSocket.claimClips(reqId, clipIds)
    },
    releaseClips: (clipIds: string[]) => {
      if (!clipIds.length) return
      projectSocket.releaseClips(clipIds)
      set(s => {
        const next = { ...s.owners }
        for (const id of clipIds) {
          delete next[id]
        }
        return { owners: next }
      })
    },
    requestPermission: async (clipId: string) => {
      const reqId = crypto.randomUUID()
      set({ outgoingPermission: { reqId, clipId } })
      const approved = await projectSocket.requestPermission(reqId, clipId)
      set(s => {
        const targetClip = s.project.clips.find(c => c.id === clipId)
        const clipName = targetClip ? s.project.assets[targetClip.mediaId]?.name || 'Clip' : 'Clip'
        return {
          outgoingPermission: s.outgoingPermission?.reqId === reqId ? null : s.outgoingPermission,
          notice: !approved ? `Permission denied: Peer declined your request to edit "${clipName}".` : s.notice,
        }
      })
      return approved
    },
    respondPermission: (reqId: string, approve: boolean) => {
      projectSocket.respondPermission(reqId, approve)
      set(s => ({
        incomingPermission: s.incomingPermission?.reqId === reqId ? null : s.incomingPermission,
      }))
    },
    proposeConsensus: async (op: Edit, description: string) => {
      const proposalId = crypto.randomUUID()
      set({
        activeConsensus: {
          proposalId,
          proposerId: get().myClientId || '',
          op,
          description,
          timeoutMs: 30000,
          expiresAt: Date.now() + 30000,
        },
      })
      const approved = await projectSocket.proposeConsensus(proposalId, op, description)
      set(s => ({
        activeConsensus: s.activeConsensus?.proposalId === proposalId ? null : s.activeConsensus,
        notice: !approved ? `Consensus rejected: Peer declined the proposal to ${description}.` : s.notice,
      }))
      return approved
    },
    voteConsensus: (proposalId: string, approve: boolean) => {
      projectSocket.voteConsensus(proposalId, approve)
      set(s => ({
        activeConsensus: s.activeConsensus?.proposalId === proposalId ? null : s.activeConsensus,
      }))
    },
  }
})
