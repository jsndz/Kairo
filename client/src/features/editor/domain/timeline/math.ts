import { duration, FPS, type Clip } from '../../state/model.ts'

/**
 * Calculates pixels per frame based on zoom level.
 * 100% zoom = 45px per second (1.5px/frame at 30fps).
 */
export function calculateScale(zoom: number, fps: number = FPS): number {
  return (45 / fps) * (zoom / 100)
}

/**
 * Calculates the total timeline width in frames.
 * Minimum 20 seconds, or longest clip end + 2 seconds buffer.
 */
export function calculateExtent(clips: Clip[], fps: number = FPS): number {
  return Math.max(20 * fps, duration(clips) + fps * 2)
}

/**
 * Calculates adaptive tick interval in frames so tick marks never crowd.
 */
export function calculateTickStep(extent: number, scale: number, fps: number = FPS): number {
  return fps * Math.max(1, Math.ceil(66 / (scale * fps)), Math.ceil(extent / fps / 240))
}

/**
 * Generates frame positions for all ruler tick marks.
 */
export function generateRulerTicks(extent: number, scale: number, fps: number = FPS): number[] {
  const step = calculateTickStep(extent, scale, fps)
  const count = Math.floor(extent / step) + 1
  return Array.from({ length: count }, (_, i) => i * step)
}

/**
 * Converts a frame number to a pixel offset.
 */
export function frameToPixel(frame: number, scale: number): number {
  return frame * scale
}

/**
 * Converts a pixel offset to a frame number.
 */
export function pixelToFrame(pixel: number, scale: number): number {
  return pixel / scale
}

/**
 * Clamps a frame value within [0, maxFrames].
 */
export function clampFrame(frame: number, maxFrames: number): number {
  return Math.max(0, Math.min(maxFrames, Math.round(frame)))
}
