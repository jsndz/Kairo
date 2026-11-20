import { create } from 'zustand'
import { demoProject } from './demo'
import { describeAsset, MAX_FRAMES, type Media, type Project } from './model'
import { applyOperation, type Operation } from './operations'
import { loadProject, PROJECT_KEY } from './projectStorage'

type HistoryEntry = { operation: Operation; before: Project; after: Project }
type Editor = {
  project: Project
  ready: boolean
  selected: string | null
  past: HistoryEntry[]
  future: HistoryEntry[]
  error: string | null
  storageError: boolean
  initialize: (media: Media[]) => void
  dispatch: (edit: OmitOperationId) => void
  select: (id: string | null) => void
  add: (media: Media) => void
  split: (frame: number) => void
  remove: () => void
  undo: () => void
  redo: () => void
}

type WithoutId<T> = T extends { id: string } ? Omit<T, 'id'> : never
type OmitOperationId = WithoutId<Operation>

export const useEditor = create<Editor>((set, get) => ({
  project: demoProject,
  ready: false,
  selected: 'opening',
  past: [],
  future: [],
  error: null,
  storageError: false,
  initialize: media => {
    if (get().ready) return
    const project = loadProject(demoProject, media)
    set({ project, ready: true, selected: project.clips[0]?.id ?? null, past: [], future: [], error: null })
  },
  dispatch: edit => {
    const state = get()
    const operation = { ...edit, id: crypto.randomUUID() }
    const result = applyOperation(state.project, operation)
    if (result.error || result.project === state.project) {
      set({ error: result.error ?? null })
      return
    }
    const selected = operation.type === 'add' ? operation.clip.id : operation.type === 'split' ? operation.rightId
      : result.project.clips.some(clip => clip.id === state.selected) ? state.selected : null
    set({
      project: result.project, selected, error: null, future: [],
      past: [...state.past.slice(-49), { operation, before: state.project, after: result.project }],
    })
  },
  select: selected => set({ selected, error: null }),
  add: media => {
    const { project, selected, dispatch } = get()
    const selectedTrack = project.clips.find(clip => clip.id === selected)?.trackId
    const track = project.tracks.find(item => item.id === selectedTrack && item.kind === media.kind)
      ?? project.tracks.find(item => item.kind === media.kind)
    if (!track) return
    const at = project.clips.reduce((end, clip) => clip.trackId === track.id ? Math.max(end, clip.at + clip.length) : end, 0)
    dispatch({ type: 'add', asset: describeAsset(media), clip: {
      id: crypto.randomUUID(), trackId: track.id, mediaId: media.id, at, offset: 0,
      length: Math.min(media.frames, MAX_FRAMES - at), volume: 1,
    } })
  },
  split: frame => {
    const { selected, dispatch } = get()
    if (selected) dispatch({ type: 'split', clipId: selected, frame, rightId: crypto.randomUUID() })
  },
  remove: () => {
    const { selected, dispatch } = get()
    if (selected) dispatch({ type: 'delete', clipId: selected })
  },
  undo: () => set(({ past, future }) => {
    const entry = past.at(-1)
    return entry ? { project: entry.before, past: past.slice(0, -1), future: [entry, ...future], selected: null, error: null } : {}
  }),
  redo: () => set(({ past, future }) => {
    const entry = future[0]
    return entry ? { project: entry.after, past: [...past, entry], future: future.slice(1), selected: null, error: null } : {}
  }),
}))

useEditor.subscribe((state, previous) => {
  if (state.project === previous.project || (!state.past.length && !state.future.length)) return
  try {
    localStorage.setItem(PROJECT_KEY, JSON.stringify(state.project))
    if (state.storageError) useEditor.setState({ storageError: false })
  } catch {
    if (!state.storageError) useEditor.setState({ storageError: true })
  }
})
