import type { Clip, Media } from '../../editor/state/model.ts'
import type { FrameRate } from '../domain/types.ts'

export type AudioEncodeSink = {
  addAudioChunk: (chunk: EncodedAudioChunk, meta?: EncodedAudioChunkMetadata) => void
}

export type SupportedAudioCodec = {
  muxerCodec: 'aac' | 'opus'
  encoderCodec: string
}

/**
 * Probes browser WebCodecs for supported audio codecs.
 * Tests AAC (mp4a.40.2) first, then falls back to Opus (supported on Linux & Firefox).
 */
export async function detectSupportedAudioCodec(): Promise<SupportedAudioCodec | null> {
  if (
    typeof window === 'undefined' ||
    typeof (window as unknown as { AudioEncoder?: unknown }).AudioEncoder !== 'function'
  ) {
    return null
  }

  // 1. Try AAC (mp4a.40.2)
  try {
    const aacSupport = await AudioEncoder.isConfigSupported({
      codec: 'mp4a.40.2',
      numberOfChannels: 2,
      sampleRate: 48000,
      bitrate: 128_000,
    })
    if (aacSupport.supported) {
      return { muxerCodec: 'aac', encoderCodec: 'mp4a.40.2' }
    }
  } catch {
    // AAC not supported
  }

  // 2. Try Opus (natively supported on Linux and Firefox WebCodecs)
  try {
    const opusSupport = await AudioEncoder.isConfigSupported({
      codec: 'opus',
      numberOfChannels: 2,
      sampleRate: 48000,
      bitrate: 128_000,
    })
    if (opusSupport.supported) {
      return { muxerCodec: 'opus', encoderCodec: 'opus' }
    }
  } catch {
    // Opus not supported
  }

  return null
}

/**
 * Renders timeline audio clips into an AudioBuffer using OfflineAudioContext,
 * then encodes to AAC or Opus chunks using WebCodecs AudioEncoder.
 */
export async function renderAndEncodeAudio({
  clips,
  mediaMap,
  totalFrames,
  fps,
  audioCodec,
  sink,
  signal,
}: {
  clips: Clip[]
  mediaMap: Map<string, Media>
  totalFrames: number
  fps: FrameRate
  audioCodec: SupportedAudioCodec
  sink: AudioEncodeSink
  signal?: AbortSignal
}): Promise<boolean> {
  if (typeof window === 'undefined' || typeof window.OfflineAudioContext === 'undefined') {
    return false
  }

  // Find clips that produce audio (audio clips and video clips)
  const audioClips = clips.filter(c => {
    const asset = mediaMap.get(c.mediaId)
    return asset && (asset.kind === 'audio' || asset.kind === 'video') && c.volume > 0
  })

  if (audioClips.length === 0) {
    return false
  }

  const sampleRate = 48000
  const durationSec = Math.max(0.1, totalFrames / fps)
  const totalSamples = Math.ceil(durationSec * sampleRate)

  // 1. Synthesize timeline audio mix via OfflineAudioContext
  const offlineCtx = new OfflineAudioContext(2, totalSamples, sampleRate)

  let attachedTracks = 0
  for (const clip of audioClips) {
    if (signal?.aborted) return false
    const asset = mediaMap.get(clip.mediaId)
    if (!asset || !asset.src) continue

    try {
      const response = await fetch(asset.src, { signal })
      const arrayBuffer = await response.arrayBuffer()
      const audioBuffer = await offlineCtx.decodeAudioData(arrayBuffer)

      const sourceNode = offlineCtx.createBufferSource()
      sourceNode.buffer = audioBuffer

      const gainNode = offlineCtx.createGain()
      gainNode.gain.value = clip.volume

      sourceNode.connect(gainNode)
      gainNode.connect(offlineCtx.destination)

      const startWhen = clip.at / fps
      const offsetSeconds = clip.offset / fps
      const clipDurationSeconds = clip.length / fps

      sourceNode.start(startWhen, offsetSeconds, clipDurationSeconds)
      attachedTracks++
    } catch (err) {
      console.warn(`Could not decode audio from source ${asset.name}:`, err)
    }
  }

  if (attachedTracks === 0) {
    return false
  }

  if (signal?.aborted) return false

  const renderedBuffer = await offlineCtx.startRendering()

  // 2. Setup AudioEncoder with detected codec
  let encoderError: Error | null = null
  const audioEncoder = new AudioEncoder({
    output: (chunk, meta) => sink.addAudioChunk(chunk, meta),
    error: err => {
      console.error('AudioEncoder error:', err)
      encoderError = err
    },
  })

  audioEncoder.configure({
    codec: audioCodec.encoderCodec,
    numberOfChannels: 2,
    sampleRate: 48000,
    bitrate: 128_000,
  })

  // Frame size: 960 for Opus (20ms @ 48kHz), 1024 for AAC
  const frameSize = audioCodec.muxerCodec === 'opus' ? 960 : 1024
  const channelLeft = renderedBuffer.getChannelData(0)
  const channelRight =
    renderedBuffer.numberOfChannels > 1
      ? renderedBuffer.getChannelData(1)
      : renderedBuffer.getChannelData(0)

  for (let offset = 0; offset < totalSamples; offset += frameSize) {
    if (signal?.aborted || encoderError) break

    const currentFrames = Math.min(frameSize, totalSamples - offset)
    const interleavedData = new Float32Array(currentFrames * 2)

    // Correct interleaved stereo layout: [L0, R0, L1, R1, ...]
    for (let i = 0; i < currentFrames; i++) {
      interleavedData[i * 2] = channelLeft[offset + i] ?? 0
      interleavedData[i * 2 + 1] = channelRight[offset + i] ?? 0
    }

    const timestampMicroseconds = Math.round((offset / sampleRate) * 1_000_000)

    const audioData = new AudioData({
      format: 'f32',
      sampleRate,
      numberOfFrames: currentFrames,
      numberOfChannels: 2,
      timestamp: timestampMicroseconds,
      data: interleavedData,
    })

    audioEncoder.encode(audioData)
    audioData.close()

    // Backpressure check for audio encoder
    if (audioEncoder.encodeQueueSize > 10) {
      await new Promise(resolve => setTimeout(resolve, 5))
    }
  }

  await audioEncoder.flush()
  audioEncoder.close()

  return !encoderError && !signal?.aborted
}
