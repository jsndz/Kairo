export const FPS = 30
export const MAX_FRAMES = FPS * 60 * 60

export type Source = {
  name: string
  kind: 'video' | 'audio' | 'image'
  mime: string
  bytes: number
  frames: number
  duration: number
  width?: number
  height?: number
}
export type Asset = Source & {
  id: string
  status: 'pending' | 'uploading' | 'completed' | 'failed'
  attempt?: string
}

export type Media = Asset & { src: string; poster?: string }
export type Track = { id: string; kind: 'video' | 'audio' }

export type Clip = {
  trackId: string
  id: string
  mediaId: string
  at: number
  offset: number
  length: number
  volume: number
}

export type Project = { name: string; tracks: Track[]; clips: Clip[]; assets: Record<string, Asset> }
export const VIDEO_TRACK = 'video-1'
export const AUDIO_TRACK = 'audio-1'
export const emptyProject = (name = 'Untitled project'): Project => ({
  name,
  tracks: [
    { id: VIDEO_TRACK, kind: 'video' },
    { id: AUDIO_TRACK, kind: 'audio' },
  ],
  clips: [],
  assets: {},
})

export const duration = (clips: Clip[]) => clips.reduce((end, clip) => Math.max(end, clip.at + clip.length), 1)

export function timecode(frame: number, frames = true) {
  const seconds = Math.floor(frame / FPS)
  const values = [Math.floor(seconds / 60), seconds % 60]
  if (frames) values.push(frame % FPS)
  return values.map(value => String(value).padStart(2, '0')).join(':')
}

export const trackKind = (kind: Source['kind']): Track['kind'] => (kind === 'audio' ? 'audio' : 'video')
