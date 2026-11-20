import { describeAsset, emptyProject, type Asset, type Clip, type Project } from './model.ts'
import { overlaps } from './operations.ts'

export type LegacyProject = { name: string; clips: Omit<Clip, 'trackId'>[] }
export const PROJECT_KEY = 'kairo-project-v2'

export function migrateProject(legacy: LegacyProject, media: Asset[]): Project {
  const project = emptyProject(legacy.name)
  const byId = new Map(media.map(asset => [asset.id, asset]))
  const sourceEnd = new Map<string, number>()
  for (const clip of legacy.clips) {
    sourceEnd.set(clip.mediaId, Math.max(sourceEnd.get(clip.mediaId) ?? 0, clip.offset + clip.length))
  }
  // Old projects used insertion order for layers. Keep earlier overlapping clips below later ones.
  for (const clip of legacy.clips) {
    const known = byId.get(clip.mediaId)
    const asset = known ? describeAsset(known) : {
      id: clip.mediaId, name: 'Missing media', kind: 'video' as const, frames: sourceEnd.get(clip.mediaId)!,
    }
    project.assets[asset.id] = asset
    const tracks = project.tracks.filter(track => track.kind === asset.kind)
    const blocked = new Set(project.clips.filter(item => overlaps(item, clip)).map(item => item.trackId))
    const lastOverlap = tracks.findLastIndex(track => blocked.has(track.id))
    const track = tracks[lastOverlap + 1] ?? { id: asset.kind + '-' + (tracks.length + 1), kind: asset.kind }
    if (!project.tracks.includes(track)) project.tracks.push(track)
    project.clips.push({ ...clip, trackId: track.id })
  }
  return project
}

export function loadProject(fallback: Project, media: Asset[]): Project {
  try {
    const current = localStorage.getItem(PROJECT_KEY)
    if (current) return JSON.parse(current) as Project
    const legacy = localStorage.getItem('kairo-project-v1')
    if (legacy) return migrateProject(JSON.parse(legacy).state.project, media)
  } catch {
    // A blocked store or unreadable document should not prevent the editor from opening.
  }
  return fallback
}
