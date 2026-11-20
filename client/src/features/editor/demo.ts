import { AUDIO_TRACK, VIDEO_TRACK, describeAsset, emptyProject, type Media, type Project } from './model'

const mediaRoot = import.meta.env.BASE_URL + 'media/'
export const demoMedia: Media[] = [
  { id: 'coast', name: 'Coastal drift.mp4', kind: 'video', frames: 270, src: mediaRoot + 'coast.mp4', poster: mediaRoot + 'coast.jpg' },
  { id: 'tide', name: 'Breaking tide.mp4', kind: 'video', frames: 240, src: mediaRoot + 'tide.mp4', poster: mediaRoot + 'tide.jpg' },
  { id: 'ambient', name: 'Ocean ambience.wav', kind: 'audio', frames: 450, src: mediaRoot + 'ambient.wav' },
]

export const demoProject: Project = {
  ...emptyProject('The Quiet Coast'),
  assets: Object.fromEntries(demoMedia.map(asset => [asset.id, describeAsset(asset)])),
  clips: [
    { id: 'opening', trackId: VIDEO_TRACK, mediaId: 'coast', at: 0, offset: 0, length: 240, volume: 1 },
    { id: 'closing', trackId: VIDEO_TRACK, mediaId: 'tide', at: 240, offset: 0, length: 210, volume: 1 },
    { id: 'sound', trackId: AUDIO_TRACK, mediaId: 'ambient', at: 0, offset: 0, length: 450, volume: 0.65 },
  ],
}
