import type { AccelerationMode } from '../domain/types.ts'

export type AccelerationResult = {
  mode: AccelerationMode
  codec: string
  error?: string
}

export function isWebCodecsSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof (window as unknown as { VideoEncoder?: unknown }).VideoEncoder === 'function' &&
    typeof (window as unknown as { VideoFrame?: unknown }).VideoFrame === 'function'
  )
}

/**
 * Probes browser WebCodecs support for H.264 video encoding.
 * Evaluates 'prefer-hardware' first (GPU); if unsupported, falls back to 'prefer-software' (CPU).
 */
export async function detectVideoAcceleration(
  width = 1920,
  height = 1080,
  fps = 30,
): Promise<AccelerationResult> {
  if (!isWebCodecsSupported()) {
    return {
      mode: 'unsupported',
      codec: '',
      error: 'WebCodecs VideoEncoder API is not supported in this browser.',
    }
  }

  // Profile candidates in order of preference:
  // avc1.640028 -> High Profile, Level 4.0 (1080p@30)
  // avc1.4d0028 -> Main Profile, Level 4.0
  // avc1.42001f -> Baseline Profile, Level 3.1
  const candidateCodecs = ['avc1.640028', 'avc1.4d0028', 'avc1.42001f']

  // 1. Check GPU Hardware Acceleration
  for (const codec of candidateCodecs) {
    try {
      const support = await VideoEncoder.isConfigSupported({
        codec,
        width,
        height,
        bitrate: 8_000_000,
        framerate: fps,
        hardwareAcceleration: 'prefer-hardware',
      })
      if (support.supported) {
        return {
          mode: 'hardware',
          codec,
        }
      }
    } catch {
      // Continue to next candidate
    }
  }

  // 2. Check CPU Software Fallback
  for (const codec of candidateCodecs) {
    try {
      const support = await VideoEncoder.isConfigSupported({
        codec,
        width,
        height,
        bitrate: 8_000_000,
        framerate: fps,
        hardwareAcceleration: 'prefer-software',
      })
      if (support.supported) {
        return {
          mode: 'software',
          codec,
        }
      }
    } catch {
      // Continue to next candidate
    }
  }

  return {
    mode: 'unsupported',
    codec: '',
    error: 'No compatible H.264 video encoder configuration found.',
  }
}
