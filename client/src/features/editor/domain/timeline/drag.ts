import { MAX_FRAMES } from '../../state/model.ts'

/**
 * Calculates the target start frame (`at`) for a clip drag, clamped within timeline boundaries.
 */
export function calculateDragTarget(
  initialAt: number,
  deltaX: number,
  scale: number,
  clipLength: number,
  maxFrames: number = MAX_FRAMES,
): number {
  return Math.max(0, Math.min(maxFrames - clipLength, Math.round(initialAt + deltaX / scale)))
}

/**
 * Computes constrained horizontal transform offset preventing a clip from moving left of frame 0.
 */
export function calculateDragTransformX(initialAt: number, scale: number, transformX: number): number {
  return Math.max(-initialAt * scale, transformX)
}
