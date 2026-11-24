import type { FrameRate, Quality, Resolution } from '../domain/types.ts'

export const RESOLUTION_DIMENSIONS: Record<Resolution, { width: number; height: number }> = {
  '720p': { width: 1280, height: 720 },
  '1080p': { width: 1920, height: 1080 },
  '4k': { width: 3840, height: 2160 },
}

export const BITRATE_PRESETS: Record<Resolution, Record<Quality, number>> = {
  '720p': { standard: 4_000_000, high: 6_000_000 },
  '1080p': { standard: 8_000_000, high: 14_000_000 },
  '4k': { standard: 20_000_000, high: 35_000_000 },
}

export type LetterboxRect = {
  dx: number
  dy: number
  dWidth: number
  dHeight: number
}

/**
 * Calculates the centered letterbox / pillarbox destination rectangle.
 * Preserves source aspect ratio inside target canvas bounds.
 * Matches FFmpeg: scale=W:H:force_original_aspect_ratio=decrease,pad=W:H:(ow-iw)/2:(oh-ih)/2
 */
export function calculateLetterbox(
  canvasWidth: number,
  canvasHeight: number,
  srcWidth: number,
  srcHeight: number,
): LetterboxRect {
  if (srcWidth <= 0 || srcHeight <= 0 || canvasWidth <= 0 || canvasHeight <= 0) {
    return { dx: 0, dy: 0, dWidth: Math.max(0, canvasWidth), dHeight: Math.max(0, canvasHeight) }
  }

  const scale = Math.min(canvasWidth / srcWidth, canvasHeight / srcHeight)
  const dWidth = Math.round(srcWidth * scale)
  const dHeight = Math.round(srcHeight * scale)
  const dx = Math.round((canvasWidth - dWidth) / 2)
  const dy = Math.round((canvasHeight - dHeight) / 2)

  return { dx, dy, dWidth, dHeight }
}

/**
 * Checks whether a timeline clip is active at a specific frame index.
 */
export function isClipActiveAtFrame(
  clipAt: number,
  clipLength: number,
  currentFrame: number,
): boolean {
  return currentFrame >= clipAt && currentFrame < clipAt + clipLength
}

/**
 * Calculates the local source media time (in seconds) for an active clip at a specific frame.
 */
export function calculateSourceTimeSeconds(
  clipAt: number,
  clipOffset: number,
  currentFrame: number,
  fps: FrameRate,
): number {
  const localFrame = currentFrame - clipAt + clipOffset
  return Math.max(0, localFrame / fps)
}
