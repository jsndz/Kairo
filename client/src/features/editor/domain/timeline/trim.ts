import { FPS, type Clip } from '../../state/model.ts'
import { edgeTrim } from '../../state/operations.ts'

export type TrimEdge = 'left' | 'right'

/**
 * Calculates updated clip properties during a trim operation.
 */
export function calculateTrimPreview(
  clip: Clip,
  assetFrames: number,
  edge: TrimEdge,
  delta: number,
): Clip {
  const trimResult = edgeTrim(clip, assetFrames, edge, delta)
  return { ...clip, ...trimResult }
}

/**
 * Calculates frame delta for keyboard arrow trimming.
 */
export function calculateKeyboardTrimDelta(
  key: 'ArrowLeft' | 'ArrowRight',
  shiftKey: boolean,
  fps: number = FPS,
): number {
  const direction = key === 'ArrowLeft' ? -1 : 1
  const step = shiftKey ? fps : 1
  return direction * step
}
