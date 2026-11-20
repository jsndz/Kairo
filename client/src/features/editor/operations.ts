import { describeAsset, emptyProject, MAX_FRAMES, type Asset, type Clip, type Project } from './model.ts'

type Edit =
  | { type: 'add'; clip: Clip; asset: Asset }
  | { type: 'move'; clipId: string; trackId: string; at: number }
  | { type: 'trim'; clipId: string; at: number; offset: number; length: number }
  | { type: 'split'; clipId: string; frame: number; rightId: string }
  | { type: 'delete'; clipId: string }
  | { type: 'volume'; clipId: string; volume: number }
  | { type: 'add-track'; trackId: string }
  | { type: 'rename'; name: string }
  | { type: 'reset' }

export type Operation = Edit & { id: string }
export type EditResult = { project: Project; error?: string }

export function overlaps(a: Pick<Clip, 'at' | 'length'>, b: Pick<Clip, 'at' | 'length'>) {
  return a.at < b.at + b.length && b.at < a.at + a.length
}

const integer = (n: number, min: number, max: number) => Number.isSafeInteger(n) && n >= min && n <= max

export function applyOperation(project: Project, operation: Operation): EditResult {
  const reject = (error: string): EditResult => ({ project, error })
  const accept = (next: Project): EditResult => ({ project: next })
  if (!operation.id) return reject('The edit needs an operation ID.')

  if (operation.type === 'reset') return accept(emptyProject())
  if (operation.type === 'rename') {
    const name = operation.name.trim()
    return !name || name.length > 80 ? reject('Enter a project name of up to 80 characters.')
      : accept(name === project.name ? project : { ...project, name })
  }
  if (operation.type === 'add-track') {
    if (!operation.trackId || project.tracks.some(track => track.id === operation.trackId)) return reject('That track already exists.')
    return accept({ ...project, tracks: [...project.tracks, { id: operation.trackId, kind: 'video' }] })
  }

  const clip = operation.type === 'add' ? operation.clip : project.clips.find(item => item.id === operation.clipId)
  if (!clip) return reject('That clip is no longer on the timeline.')
  if (operation.type === 'delete') return accept({ ...project, clips: project.clips.filter(item => item !== clip) })
  if (operation.type === 'split') {
    if (!integer(operation.frame, clip.at + 1, clip.at + clip.length - 1)) return reject('Move the playhead inside the clip to split it.')
    if (!operation.rightId || project.clips.some(item => item.id === operation.rightId)) return reject('That clip ID already exists.')
    const length = operation.frame - clip.at
    const right = { ...clip, id: operation.rightId, at: operation.frame, offset: clip.offset + length, length: clip.length - length }
    return accept({ ...project, clips: project.clips.flatMap(item => item === clip ? [{ ...clip, length }, right] : [item]) })
  }
  if (operation.type === 'volume') {
    if (!Number.isFinite(operation.volume) || operation.volume < 0 || operation.volume > 1) return reject('Volume must be between 0% and 100%.')
    if (operation.volume === clip.volume) return accept(project)
    return accept({ ...project, clips: project.clips.map(item => item === clip ? { ...clip, volume: operation.volume } : item) })
  }

  const asset = project.assets[clip.mediaId] ?? (operation.type === 'add' ? operation.asset : undefined)
  if (!asset || asset.id !== clip.mediaId || !integer(asset.frames, 1, Number.MAX_SAFE_INTEGER)) return reject('Media metadata is unavailable.')
  if (operation.type === 'add' && (!clip.id || project.clips.some(item => item.id === clip.id))) return reject('That clip ID already exists.')
  const next = operation.type === 'move' ? { ...clip, at: operation.at, trackId: operation.trackId }
    : operation.type === 'trim' ? { ...clip, at: operation.at, offset: operation.offset, length: operation.length } : clip

  if (!project.tracks.some(track => track.id === next.trackId && track.kind === asset.kind)) return reject('Choose a track that matches the media type.')
  if (!integer(next.at, 0, MAX_FRAMES - 1) || !integer(next.offset, 0, asset.frames - 1) ||
    !integer(next.length, 1, Math.min(asset.frames - next.offset, MAX_FRAMES - next.at))) return reject('The edit exceeds the source or timeline limits.')
  if (!Number.isFinite(next.volume) || next.volume < 0 || next.volume > 1) return reject('Volume must be between 0% and 100%.')
  if (project.clips.some(item => item !== clip && item.trackId === next.trackId && overlaps(item, next))) {
    return reject('Clips on the same track cannot overlap.')
  }
  if (operation.type === 'add') return accept({
    ...project, clips: [...project.clips, next],
    assets: project.assets[asset.id] ? project.assets : { ...project.assets, [asset.id]: describeAsset(asset) },
  })
  if (next.at === clip.at && next.trackId === clip.trackId && next.offset === clip.offset && next.length === clip.length) return accept(project)
  return accept({ ...project, clips: project.clips.map(item => item === clip ? next : item) })
}

export function edgeTrim(clip: Clip, sourceFrames: number, edge: 'left' | 'right', delta: number) {
  if (edge === 'left') {
    const shift = Math.max(-Math.min(clip.at, clip.offset), Math.min(clip.length - 1, delta))
    return { at: clip.at + shift, offset: clip.offset + shift, length: clip.length - shift }
  }
  return { at: clip.at, offset: clip.offset, length: Math.max(1, Math.min(sourceFrames - clip.offset, MAX_FRAMES - clip.at, clip.length + delta)) }
}
