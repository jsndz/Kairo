import { memo } from 'react'
import { AudioLines, Film, Image } from 'lucide-react'
import { FPS, type Asset, type Clip, type Media } from '../../state/model.ts'
import { Waveform } from '../Waveform.tsx'

export const ClipContent = memo(function ClipContent({
  asset,
  source,
  clip,
}: {
  asset: Asset
  source?: Media
  clip: Clip
}) {
  return (
    <>
      <span className="clip-name">
        {asset.kind === 'video' ? (
          <Film size={12} />
        ) : asset.kind === 'audio' ? (
          <AudioLines size={12} />
        ) : (
          <Image size={12} />
        )}
        {asset.name}
        {!source && ' · Missing media'}
      </span>
      {asset.kind === 'audio' && source && asset.frames <= FPS * 600 && (
        <Waveform src={source.src} offset={clip.offset} length={clip.length} frames={asset.frames} />
      )}
    </>
  )
})
