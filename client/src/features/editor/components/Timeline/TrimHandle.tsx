import type { Clip } from '../../state/model.ts'
import { useTrimHandle } from '../../hooks/useTrimHandle.ts'
import type { TrimEdge } from '../../domain/timeline/trim.ts'

export function TrimHandle({
  edge,
  clip,
  assetFrames,
  scale,
  onPreview,
  onCommit,
}: {
  edge: TrimEdge
  clip: Clip
  assetFrames: number
  scale: number
  onPreview: (clip: Clip | null) => void
  onCommit: (trimUpdate: { at: number; offset: number; length: number }) => void
}) {
  const { onPointerDown, onPointerMove, onPointerUp, onLostPointerCapture, onKeyDown } = useTrimHandle({
    clip,
    assetFrames,
    edge,
    scale,
    onPreview,
    onCommit,
  })

  return (
    <button
      type="button"
      className={`trim-handle ${edge}`}
      aria-label={edge === 'left' ? 'Trim clip start' : 'Trim clip end'}
      title={edge === 'left' ? 'Drag to trim start' : 'Drag to trim end'}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onLostPointerCapture={onLostPointerCapture}
      onKeyDown={onKeyDown}
    />
  )
}
