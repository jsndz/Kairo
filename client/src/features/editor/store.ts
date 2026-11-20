import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { demoProject, MAX_FRAMES, type Clip, type Media, type Project } from './model'

type Editor = {
  project: Project
  selected: string | null
  past: Project[]
  future: Project[]
  commit: (project: Project) => void
  select: (id: string | null) => void
  add: (media: Media, at: number) => void
  update: (id: string, patch: Pick<Partial<Clip>, 'at' | 'offset' | 'length' | 'volume'>, media: Media) => void
  split: (frame: number) => void
  remove: () => void
  undo: () => void
  redo: () => void
}

export const useEditor = create<Editor>()(persist((set, get) => ({
  project: demoProject,
  selected: 'opening',
  past: [],
  future: [],
  commit: project => set(state => ({ project, past: [...state.past.slice(-49), state.project], future: [] })),
  select: selected => set({ selected }),
  add: (media, at) => {
    const { project, commit } = get()
    const id = crypto.randomUUID()
    const length = Math.min(media.frames, MAX_FRAMES - at)
    if (length < 1) return
    commit({ ...project, clips: [...project.clips, { id, mediaId: media.id, at, offset: 0, length, volume: 1 }] })
    set({ selected: id })
  },
  update: (id, patch, media) => {
    const { project, commit } = get()
    const clip = project.clips.find(item => item.id === id)
    if (!clip || Object.values(patch).some(value => !Number.isFinite(value))) return
    const next = { ...clip, ...patch }
    next.at = Math.max(0, Math.min(MAX_FRAMES - 1, Math.round(next.at)))
    next.offset = Math.max(0, Math.min(media.frames - 1, Math.round(next.offset)))
    next.length = Math.max(1, Math.min(media.frames - next.offset, MAX_FRAMES - next.at, Math.round(next.length)))
    next.volume = Math.max(0, Math.min(1, next.volume))
    if (next.at === clip.at && next.offset === clip.offset && next.length === clip.length && next.volume === clip.volume) return
    commit({ ...project, clips: project.clips.map(item => item.id === id ? next : item) })
  },
  split: frame => {
    const { project, selected, commit } = get()
    const clip = project.clips.find(item => item.id === selected)
    if (!clip || frame <= clip.at || frame >= clip.at + clip.length) return
    const length = frame - clip.at
    const right = { ...clip, id: crypto.randomUUID(), at: frame, offset: clip.offset + length, length: clip.length - length }
    commit({ ...project, clips: project.clips.flatMap(item => item.id === clip.id ? [{ ...clip, length }, right] : [item]) })
    set({ selected: right.id })
  },
  remove: () => {
    const { project, selected, commit } = get()
    if (!project.clips.some(clip => clip.id === selected)) return
    commit({ ...project, clips: project.clips.filter(clip => clip.id !== selected) })
    set({ selected: null })
  },
  undo: () => set(({ past, future, project }) => past.length ? {
    project: past[past.length - 1], past: past.slice(0, -1), future: [project, ...future], selected: null,
  } : {}),
  redo: () => set(({ past, future, project }) => future.length ? {
    project: future[0], past: [...past, project], future: future.slice(1), selected: null,
  } : {}),
}), {
  name: 'kairo-project-v1',
  storage: createJSONStorage(() => localStorage),
  partialize: ({ project }) => ({ project }),
}))
