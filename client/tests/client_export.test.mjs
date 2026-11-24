import test from 'node:test'
import assert from 'node:assert/strict'
import {
  calculateLetterbox,
  calculateSourceTimeSeconds,
  isClipActiveAtFrame,
  RESOLUTION_DIMENSIONS,
  BITRATE_PRESETS,
} from '../src/features/export/engine/compositor.ts'
import {
  detectVideoAcceleration,
  isWebCodecsSupported,
} from '../src/features/export/engine/hardware.ts'
import { useExportStore } from '../src/features/export/state/useExportStore.ts'

test('letterbox calculations maintain aspect ratios correctly', () => {
  // 1. Same 16:9 aspect ratio fills canvas completely
  const match = calculateLetterbox(1920, 1080, 1920, 1080)
  assert.equal(match.dx, 0)
  assert.equal(match.dy, 0)
  assert.equal(match.dWidth, 1920)
  assert.equal(match.dHeight, 1080)

  // 2. 720p 16:9 scaled up to 1080p canvas
  const upscale16_9 = calculateLetterbox(1920, 1080, 1280, 720)
  assert.equal(upscale16_9.dx, 0)
  assert.equal(upscale16_9.dy, 0)
  assert.equal(upscale16_9.dWidth, 1920)
  assert.equal(upscale16_9.dHeight, 1080)

  // 3. Ultra-wide 3:1 image (3072x1024) in 1920x1080 canvas -> letterboxed top & bottom
  // scale = 1920 / 3072 = 0.625 -> dHeight = 640 -> dy = (1080 - 640)/2 = 220
  const wide = calculateLetterbox(1920, 1080, 3072, 1024)
  assert.equal(wide.dx, 0)
  assert.equal(wide.dy, 220)
  assert.equal(wide.dWidth, 1920)
  assert.equal(wide.dHeight, 640)

  // 4. Portrait 9:16 video (1080x1920) in 1920x1080 canvas -> pillarboxed left & right
  // scale = 1080 / 1920 = 0.5625 -> dWidth = 607.5 -> 608 -> dx = (1920 - 608)/2 = 656
  const portrait = calculateLetterbox(1920, 1080, 1080, 1920)
  assert.equal(portrait.dy, 0)
  assert.equal(portrait.dHeight, 1080)
  assert.equal(portrait.dWidth, 608)
  assert.equal(portrait.dx, 656)

  // 5. Zero / negative dimension safeguards
  const zero = calculateLetterbox(1920, 1080, 0, 0)
  assert.equal(zero.dx, 0)
  assert.equal(zero.dy, 0)
  assert.equal(zero.dWidth, 1920)
  assert.equal(zero.dHeight, 1080)
})

test('clip timeline active frame and time offset calculations', () => {
  const at = 30
  const length = 60
  const offset = 15
  const fps = 30

  // Frame before clip
  assert.equal(isClipActiveAtFrame(at, length, 29), false)

  // First frame of clip
  assert.equal(isClipActiveAtFrame(at, length, 30), true)
  assert.equal(calculateSourceTimeSeconds(at, offset, 30, fps), 0.5) // (30 - 30 + 15)/30 = 0.5s

  // Middle frame of clip
  assert.equal(isClipActiveAtFrame(at, length, 60), true)
  assert.equal(calculateSourceTimeSeconds(at, offset, 60, fps), 1.5) // (60 - 30 + 15)/30 = 1.5s

  // Last active frame of clip (89)
  assert.equal(isClipActiveAtFrame(at, length, 89), true)

  // Exact boundary end frame (90 is outside [30, 90))
  assert.equal(isClipActiveAtFrame(at, length, 90), false)
})

test('resolution presets and bitrates are well-defined', () => {
  assert.deepEqual(RESOLUTION_DIMENSIONS['720p'], { width: 1280, height: 720 })
  assert.deepEqual(RESOLUTION_DIMENSIONS['1080p'], { width: 1920, height: 1080 })
  assert.deepEqual(RESOLUTION_DIMENSIONS['4k'], { width: 3840, height: 2160 })

  assert.ok(BITRATE_PRESETS['1080p'].high > BITRATE_PRESETS['1080p'].standard)
  assert.ok(BITRATE_PRESETS['4k'].standard > BITRATE_PRESETS['1080p'].high)
})

test('hardware acceleration detector handles unsupported environments gracefully', async () => {
  // In Node.js testing environment, window.VideoEncoder is undefined
  assert.equal(isWebCodecsSupported(), false)

  const probe = await detectVideoAcceleration(1920, 1080, 30)
  assert.equal(probe.mode, 'unsupported')
  assert.match(probe.error, /WebCodecs VideoEncoder API is not supported/i)
})

test('export store manages local browser method and cancellation', async () => {
  const store = useExportStore.getState()

  // Select local browser method
  store.updateSettings({ method: 'local' })
  assert.equal(useExportStore.getState().settings.method, 'local')

  // Cancel simulated client export
  useExportStore.setState({
    activeJob: {
      id: 'client_test123',
      requestId: 'req_123',
      projectId: '',
      revision: 1,
      status: 'rendering',
      progress: 45,
      attempts: 1,
      outputKey: 'test.mp4',
      createdAt: new Date().toISOString(),
      accelerationMode: 'hardware',
    },
  })

  await store.cancelExport()
  assert.equal(useExportStore.getState().activeJob?.status, 'cancelled')

  // Dismiss job
  store.dismissJob()
  assert.equal(useExportStore.getState().activeJob, null)
})
