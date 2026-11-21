import { create } from 'zustand'
import { commitEdit, getProject, type ProjectState } from '../api'
import { emptyProject, MAX_FRAMES, trackKind, type Asset, type Media } from './model'
import type { Edit } from './operations'

type Editor = ProjectState & {
  ready: boolean
  selected: string | null
  error: string | null
  initialize: () => Promise<void>
  dispatch: (edit: Edit) => Promise<boolean>
  select: (id: string | null) => void
  putAsset: (asset: Asset) => void
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
    set({ ...state, ready: true, selected: state.project.clips.some(clip => clip.id === selection) ? selection : null })
  }

  return {
    id: '',
    revision: 0,
    project: emptyProject(),
    canUndo: false,
    canRedo: false,
    ready: false,
    selected: null,
    error: null,
    initialize: () => {
      if (get().ready) return Promise.resolve()
      if (opening) return opening
      set({ error: null })
      opening = getProject()
        .then(state => adopt(state))
        .catch((error: unknown) => set({ error: errorMessage(error) }))
        .finally(() => {
          opening = undefined
        })
      return opening
    },
    dispatch: edit =>
      serialize(async () => {
        const id = crypto.randomUUID()
        try {
          if (!get().ready) return false
          if (needsRefresh) {
            adopt(await getProject())
            needsRefresh = false
          }
          const state = await commitEdit(id, get().revision, edit)
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
            adopt(await getProject())
            needsRefresh = false
          } catch {
            /* Resync before the next edit. */
          }
          set({ error: errorMessage(error) })
          return false
        }
      }),
    select: selected => set({ selected, error: null }),
    putAsset: asset =>
      set(({ project }) => ({
        project: { ...project, assets: { ...project.assets, [asset.id]: asset } },
      })),
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
      const { selected, dispatch } = get()
      if (selected) void dispatch({ type: 'split', clipId: selected, frame, rightId: crypto.randomUUID() })
    },
    remove: () => {
      const { selected, dispatch } = get()
      if (selected) void dispatch({ type: 'delete', clipId: selected })
    },
    undo: () => {
      if (get().canUndo) void get().dispatch({ type: 'undo' })
    },
    redo: () => {
      if (get().canRedo) void get().dispatch({ type: 'redo' })
    },
  }
})
