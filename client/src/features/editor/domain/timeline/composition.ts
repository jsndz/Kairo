import type { Clip, Media, Track } from '../../state/model.ts'

export type OrderedCompositionClip = {
  clip: Clip
  asset: Media
}

/**
 * Orders timeline clips by track stacking order and filters out clips with missing media.
 */
export function orderCompositionClips(
  clips: Clip[],
  tracks: Track[],
  media: Media[],
): OrderedCompositionClip[] {
  const assets = new Map(media.map(asset => [asset.id, asset]))
  const trackOrder = new Map(tracks.map((track, index) => [track.id, index]))

  return [...clips]
    .sort((a, b) => (trackOrder.get(a.trackId) ?? 0) - (trackOrder.get(b.trackId) ?? 0))
    .flatMap(clip => {
      const asset = assets.get(clip.mediaId)
      return asset ? [{ clip, asset }] : []
    })
}
