import { Muxer, ArrayBufferTarget } from 'mp4-muxer'
import type { Clip, Media, Track } from '../../editor/state/model.ts'
import { orderCompositionClips } from '../../editor/domain/timeline/composition.ts'
import type { ClientExportProgress, ExportSettings } from '../domain/types.ts'
import {
  BITRATE_PRESETS,
  calculateLetterbox,
  calculateSourceTimeSeconds,
  isClipActiveAtFrame,
  RESOLUTION_DIMENSIONS,
} from './compositor.ts'
import { detectVideoAcceleration } from './hardware.ts'
import { detectSupportedAudioCodec, renderAndEncodeAudio } from './audioPipeline.ts'

export type ClientExportParams = {
  settings: ExportSettings
  clips: Clip[]
  tracks: Track[]
  media: Media[]
  onProgress: (progress: ClientExportProgress) => void
  signal?: AbortSignal
}

export type ClientExportResult = {
  blob: Blob
  blobUrl: string
}

function seekVideo(video: HTMLVideoElement, timeSeconds: number): Promise<void> {
  return new Promise(resolve => {
    if (Math.abs(video.currentTime - timeSeconds) < 0.015) {
      resolve()
      return
    }
    const onSeeked = () => {
      video.removeEventListener('seeked', onSeeked)
      resolve()
    }
    video.addEventListener('seeked', onSeeked, { once: true })
    video.currentTime = Math.max(0, timeSeconds)
  })
}

/**
 * Executes high-efficiency in-browser export using WebCodecs (GPU/CPU fallback),
 * zero-copy OffscreenCanvas / 2D Canvas compositing, backpressure throttling,
 * and streaming MP4 muxing via mp4-muxer.
 */
export async function runClientExport({
  settings,
  clips,
  tracks,
  media,
  onProgress,
  signal,
}: ClientExportParams): Promise<ClientExportResult> {
  if (clips.length === 0) {
    throw new Error('Timeline is empty. Cannot export an empty timeline.')
  }

  // 1. Determine composition bounds and frame metrics
  const { width, height } = RESOLUTION_DIMENSIONS[settings.resolution]
  const fps = settings.frameRate
  const bitrate = BITRATE_PRESETS[settings.resolution][settings.quality]

  // Total frames is max (clip.at + clip.length) across all clips
  const totalFrames = clips.reduce((acc, c) => Math.max(acc, c.at + c.length), 1)

  onProgress({
    frame: 0,
    totalFrames,
    percent: 0,
    stage: 'preparing',
  })

  // 2. Detect hardware acceleration mode
  const hardwareProbe = await detectVideoAcceleration(width, height, fps)
  if (hardwareProbe.mode === 'unsupported') {
    throw new Error(
      hardwareProbe.error || 'Your browser does not support WebCodecs H.264 video encoding.',
    )
  }

  const mediaMap = new Map(media.map(m => [m.id, m]))
  const orderedLayers = orderCompositionClips(clips, tracks, media)

  // 3. Preload visual media elements (video, images)
  const visualElements = new Map<string, HTMLVideoElement | HTMLImageElement>()
  for (const { asset } of orderedLayers) {
    if (visualElements.has(asset.id)) continue

    if (asset.kind === 'video') {
      const v = document.createElement('video')
      v.crossOrigin = 'anonymous'
      v.muted = true
      v.playsInline = true
      v.preload = 'auto'
      v.src = asset.src
      await new Promise<void>((resolve, reject) => {
        v.onloadeddata = () => resolve()
        v.onerror = () => reject(new Error(`Failed to load video source: ${asset.name}`))
        v.load()
      })
      visualElements.set(asset.id, v)
    } else if (asset.kind === 'image') {
      const img = document.createElement('img')
      img.crossOrigin = 'anonymous'
      img.src = asset.src
      await new Promise<void>((resolve, reject) => {
        img.onload = () => resolve()
        img.onerror = () => reject(new Error(`Failed to load image source: ${asset.name}`))
      })
      visualElements.set(asset.id, img)
    }
  }

  if (signal?.aborted) {
    throw new DOMException('Export cancelled by user', 'AbortError')
  }

  // 4. Detect audio codec and check for active timeline audio tracks
  const target = new ArrayBufferTarget()
  const hasAudioClips = clips.some(c => {
    const a = mediaMap.get(c.mediaId)
    return a && (a.kind === 'audio' || a.kind === 'video') && c.volume > 0
  })

  const supportedAudioCodec = hasAudioClips ? await detectSupportedAudioCodec() : null

  const muxer = new Muxer({
    target,
    video: {
      codec: 'avc',
      width,
      height,
      frameRate: fps,
    },
    audio: supportedAudioCodec
      ? {
          codec: supportedAudioCodec.muxerCodec,
          numberOfChannels: 2,
          sampleRate: 48000,
        }
      : undefined,
    fastStart: 'in-memory',
  })

  // 5. Render & encode audio if timeline contains audio tracks and codec is available
  if (supportedAudioCodec) {
    onProgress({
      frame: 0,
      totalFrames,
      percent: 5,
      stage: 'audio',
    })

    try {
      await renderAndEncodeAudio({
        clips,
        mediaMap,
        totalFrames,
        fps,
        audioCodec: supportedAudioCodec,
        sink: {
          addAudioChunk: (chunk, meta) => muxer.addAudioChunk(chunk, meta),
        },
        signal,
      })
    } catch (err) {
      console.warn('Audio rendering failed or skipped, proceeding with video:', err)
    }
  }

  if (signal?.aborted) {
    throw new DOMException('Export cancelled by user', 'AbortError')
  }

  // 6. Setup VideoEncoder
  let encoderError: Error | null = null
  const videoEncoder = new VideoEncoder({
    output: (chunk, meta) => {
      muxer.addVideoChunk(chunk, meta)
    },
    error: err => {
      encoderError = err
    },
  })

  videoEncoder.configure({
    codec: hardwareProbe.codec,
    width,
    height,
    bitrate,
    framerate: fps,
    hardwareAcceleration:
      hardwareProbe.mode === 'hardware' ? 'prefer-hardware' : 'prefer-software',
    avc: { format: 'avc' },
  })

  // 7. Setup Canvas Compositor
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d', { alpha: false, desynchronized: true })
  if (!ctx) {
    throw new Error('Failed to create 2D canvas rendering context.')
  }

  const frameDurationUs = Math.round(1_000_000 / fps)

  // 8. Main Frame Compositing & Encoding Loop
  for (let frameIndex = 0; frameIndex < totalFrames; frameIndex++) {
    if (signal?.aborted) {
      videoEncoder.close()
      throw new DOMException('Export cancelled by user', 'AbortError')
    }
    if (encoderError) {
      videoEncoder.close()
      throw encoderError
    }

    // A. Clear canvas with baseline background color (#050607)
    ctx.fillStyle = '#050607'
    ctx.fillRect(0, 0, width, height)

    // B. Composite active visual layers in track order
    for (const { clip, asset } of orderedLayers) {
      if (!isClipActiveAtFrame(clip.at, clip.length, frameIndex)) continue

      const element = visualElements.get(asset.id)
      if (!element) continue

      if (asset.kind === 'video' && element instanceof HTMLVideoElement) {
        const sourceTime = calculateSourceTimeSeconds(clip.at, clip.offset, frameIndex, fps)
        await seekVideo(element, sourceTime)
        const rect = calculateLetterbox(width, height, element.videoWidth, element.videoHeight)
        ctx.drawImage(element, rect.dx, rect.dy, rect.dWidth, rect.dHeight)
      } else if (asset.kind === 'image' && element instanceof HTMLImageElement) {
        const rect = calculateLetterbox(width, height, element.naturalWidth, element.naturalHeight)
        ctx.drawImage(element, rect.dx, rect.dy, rect.dWidth, rect.dHeight)
      }
    }

    // C. Capture zero-copy VideoFrame
    const timestampUs = frameIndex * frameDurationUs
    const videoFrame = new VideoFrame(canvas, {
      timestamp: timestampUs,
      duration: frameDurationUs,
    })

    // Keyframe every 2 seconds (or first frame)
    const isKeyframe = frameIndex % (fps * 2) === 0
    videoEncoder.encode(videoFrame, { keyFrame: isKeyframe })
    videoFrame.close() // Reclaim GPU memory immediately

    // D. Backpressure control: throttle loop if encoder queue is backed up
    if (videoEncoder.encodeQueueSize > 5) {
      await new Promise(resolve => {
        videoEncoder.ondequeue = () => {
          if (videoEncoder.encodeQueueSize <= 2) {
            videoEncoder.ondequeue = null
            resolve(true)
          }
        }
      })
    }

    // E. Emit progress
    const percent = Math.min(99, Math.round(((frameIndex + 1) / totalFrames) * 90) + 5)
    onProgress({
      frame: frameIndex + 1,
      totalFrames,
      percent,
      stage: 'rendering',
    })
  }

  // 9. Flush VideoEncoder
  onProgress({
    frame: totalFrames,
    totalFrames,
    percent: 98,
    stage: 'muxing',
  })

  await videoEncoder.flush()
  videoEncoder.close()

  // 10. Finalize MP4 Container
  muxer.finalize()

  const blob = new Blob([target.buffer], { type: 'video/mp4' })
  const blobUrl = URL.createObjectURL(blob)

  onProgress({
    frame: totalFrames,
    totalFrames,
    percent: 100,
    stage: 'completed',
  })

  return { blob, blobUrl }
}
