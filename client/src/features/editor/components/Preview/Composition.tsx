import { useMemo } from 'react'
import { AbsoluteFill, Html5Audio, Html5Video, Sequence, Img } from 'remotion'
import { FPS, type Clip, type Media, type Track } from '../../state/model.ts'
import { orderCompositionClips } from '../../domain/timeline/composition.ts'

export type PreviewInput = {
  clips: Clip[]
  tracks: Track[]
  media: Media[]
  onMediaError: (src: string) => void
}

export function Composition({ clips, tracks, media, onMediaError }: PreviewInput) {
  const ordered = useMemo(
    () => orderCompositionClips(clips, tracks, media),
    [clips, tracks, media],
  )

  return (
    <AbsoluteFill style={{ backgroundColor: '#050607' }}>
      {ordered.map(({ clip, asset }) => (
        <Sequence key={clip.id} from={clip.at} durationInFrames={clip.length} premountFor={FPS}>
          {asset.kind === 'video' ? (
            <Html5Video
              key={asset.src}
              src={asset.src}
              trimBefore={clip.offset}
              volume={clip.volume}
              pauseWhenBuffering
              onError={() => onMediaError(asset.src)}
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          ) : asset.kind === 'audio' ? (
            <Html5Audio
              key={asset.src}
              src={asset.src}
              trimBefore={clip.offset}
              volume={clip.volume}
              pauseWhenBuffering
              onError={() => onMediaError(asset.src)}
            />
          ) : (
            <Img
              key={asset.src}
              src={asset.src}
              onError={() => onMediaError(asset.src)}
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          )}
        </Sequence>
      ))}
    </AbsoluteFill>
  )
}
