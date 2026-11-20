export const FPS = 30
export const MAX_FRAMES = FPS * 60 * 60

export type Media = {
  id: string
  name: string
  kind: 'video' | 'audio'
  frames: number
  src: string
  poster?: string
}

export type Clip = {
  id: string
  mediaId: string
  at: number
  offset: number
  length: number
  volume: number
}

export type Project = { name: string; clips: Clip[] }
export const duration = (clips: Clip[]) => Math.max(1, ...clips.map(c => c.at + c.length))

export function timecode(frame: number, frames = true) {
  const seconds = Math.floor(frame / FPS)
  const values = [Math.floor(seconds / 60), seconds % 60]
  if (frames) values.push(frame % FPS)
  return values.map(value => String(value).padStart(2, '0')).join(':')
}

const mediaRoot = import.meta.env.BASE_URL + 'media/'
export const demoMedia: Media[] = [
  { id: 'coast', name: 'Coastal drift.mp4', kind: 'video', frames: 270, src: mediaRoot + 'coast.mp4', poster: mediaRoot + 'coast.jpg' },
  { id: 'tide', name: 'Breaking tide.mp4', kind: 'video', frames: 240, src: mediaRoot + 'tide.mp4', poster: mediaRoot + 'tide.jpg' },
  { id: 'ambient', name: 'Ocean ambience.wav', kind: 'audio', frames: 450, src: mediaRoot + 'ambient.wav' },
]

export const demoProject: Project = {
  name: 'The Quiet Coast',
  clips: [
    { id: 'opening', mediaId: 'coast', at: 0, offset: 0, length: 240, volume: 1 },
    { id: 'closing', mediaId: 'tide', at: 240, offset: 0, length: 210, volume: 1 },
    { id: 'sound', mediaId: 'ambient', at: 0, offset: 0, length: 450, volume: 0.65 },
  ],
}
