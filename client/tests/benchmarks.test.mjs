import test from 'node:test'
import assert from 'node:assert/strict'
import { performance } from 'node:perf_hooks'
import {
  calculateScale,
  calculateExtent,
  calculateTickStep,
  generateRulerTicks,
  calculateDragTarget,
  calculateDragTransformX,
  orderCompositionClips,
} from '../src/features/editor/domain/index.ts'
import { useExportStore } from '../src/features/export/state/useExportStore.ts'
import {
  calculateLetterbox,
  calculateSourceTimeSeconds,
  isClipActiveAtFrame,
  RESOLUTION_DIMENSIONS,
} from '../src/features/export/engine/compositor.ts'

function formatBenchmark(name, iterations, durationMs, unit = 'ops/sec', perOpUnit = 'ns/op', multiplier = 1e6) {
  const opsPerSec = (iterations / (durationMs / 1000)).toLocaleString('en-US', { maximumFractionDigits: 1 })
  const rawTime = (durationMs * multiplier) / iterations
  const timePerOp = rawTime < 1 ? rawTime.toFixed(2) : rawTime.toLocaleString('en-US', { maximumFractionDigits: 1 })
  console.log(`  [BENCHMARK] ${name.padEnd(45)}: ${opsPerSec.padStart(12)} ${unit} (${timePerOp.padStart(8)} ${perOpUnit})`)
}

test('Frontend Benchmark Suite', async (t) => {
  console.log('\n================================================================================')
  console.log('              KAIRO FRONTEND ENGINE PERFORMANCE BENCHMARKS                      ')
  console.log('================================================================================')

  await t.test('Timeline Composition Ordering (1,000 clips)', () => {
    const trackCount = 8
    const tracks = []
    for (let i = 0; i < trackCount; i++) {
      tracks.push({ id: `track_${i}`, name: `Track ${i}`, kind: i % 2 === 0 ? 'video' : 'audio', muted: false, locked: false })
    }
    const media = []
    for (let i = 0; i < 20; i++) {
      media.push({ id: `m_${i}`, name: `Asset ${i}`, kind: 'video', duration: 10, fps: 30, frames: 300 })
    }

    const clipCount = 1000
    const clips = []
    for (let i = 0; i < clipCount; i++) {
      clips.push({
        id: `c_${i}`,
        trackId: `track_${i % trackCount}`,
        mediaId: `m_${i % 20}`,
        at: (i * 15) % 1800,
        offset: 0,
        length: 60,
        volume: 1.0,
      })
    }

    const iterations = 500
    const t0 = performance.now()
    for (let i = 0; i < iterations; i++) {
      const ordered = orderCompositionClips(clips, tracks, media)
      assert.equal(ordered.length, clipCount)
    }
    const t1 = performance.now()
    formatBenchmark('Clip Composition Ordering (1,000 clips)', iterations, t1 - t0)
  })

  await t.test('Timeline Zoom & Ruler Coordinate Transforms (100,000 iterations)', () => {
    const iterations = 100_000
    const t0 = performance.now()
    let sum = 0
    for (let i = 0; i < iterations; i++) {
      const zoom = 50 + (i % 250)
      const scale = calculateScale(zoom, 30)
      const step = calculateTickStep(scale, 30)
      sum += scale + step
    }
    const t1 = performance.now()
    assert.ok(sum > 0)
    formatBenchmark('Zoom & Tick Step Calculation', iterations, t1 - t0)
  })

  await t.test('Ruler Tick Generation for 60-Second Timeline (10,000 iterations)', () => {
    const iterations = 10_000
    const t0 = performance.now()
    let totalTicks = 0
    for (let i = 0; i < iterations; i++) {
      const ticks = generateRulerTicks(1800, 1.5, 30)
      totalTicks += ticks.length
    }
    const t1 = performance.now()
    assert.ok(totalTicks > 0)
    formatBenchmark('Ruler Tick Array Generation (1.8k frames)', iterations, t1 - t0)
  })

  await t.test('Interactive Clip Drag Bounds & Constraint Math (100,000 iterations)', () => {
    const iterations = 100_000
    const t0 = performance.now()
    let sum = 0
    for (let i = 0; i < iterations; i++) {
      const at = (i * 7) % 1000
      const deltaX = (i % 400) - 200
      const target = calculateDragTarget(at, deltaX, 1.5, 60, 2000)
      const transX = calculateDragTransformX(at, 1.5, deltaX)
      sum += target + transX
    }
    const t1 = performance.now()
    assert.ok(sum !== 0)
    formatBenchmark('Drag Target & Transform Math', iterations, t1 - t0)
  })

  await t.test('Export Store Preflight & Settings Updates (20,000 cycles)', () => {
    const iterations = 20_000
    const store = useExportStore.getState()
    const t0 = performance.now()
    for (let i = 0; i < iterations; i++) {
      store.checkPreflight(i % 50, false)
      store.updateSettings({ frameRate: (i % 2 === 0) ? 30 : 60 })
    }
    const t1 = performance.now()
    formatBenchmark('Export Store Preflight & Settings Dispatch', iterations, t1 - t0)
  })

  await t.test('Export Letterbox Aspect Ratio Transforms (100,000 iterations)', () => {
    const iterations = 100_000
    const target = RESOLUTION_DIMENSIONS['1080p'] // 1920x1080
    const srcAspectRatios = [
      { w: 1920, h: 1080 }, // 16:9 1080p
      { w: 3840, h: 2160 }, // 16:9 4k
      { w: 1080, h: 1920 }, // 9:16 vertical
      { w: 1280, h: 720 },  // 16:9 720p
      { w: 1440, h: 1080 }, // 4:3 SD
      { w: 2560, h: 1080 }, // 21:9 ultrawide
    ]

    const t0 = performance.now()
    let checkSum = 0
    for (let i = 0; i < iterations; i++) {
      const src = srcAspectRatios[i % srcAspectRatios.length]
      const rect = calculateLetterbox(target.width, target.height, src.w, src.h)
      checkSum += rect.dx + rect.dy + rect.dWidth + rect.dHeight
    }
    const t1 = performance.now()
    assert.ok(checkSum > 0)
    formatBenchmark('Export Letterbox & Aspect Ratio Math', iterations, t1 - t0)
  })

  await t.test('Client Export Compositing Loop Simulation (60s Timeline / 1,800 frames)', () => {
    // Simulate a realistic multi-track timeline: 1,800 frames (60s @ 30fps) with 12 layered video & image clips
    const totalFrames = 1800
    const fps = 30
    const { width, height } = RESOLUTION_DIMENSIONS['1080p']
    const layers = [
      { clip: { at: 0, length: 600, offset: 0 }, asset: { kind: 'video', width: 1920, height: 1080 } },
      { clip: { at: 500, length: 700, offset: 15 }, asset: { kind: 'video', width: 3840, height: 2160 } },
      { clip: { at: 1100, length: 700, offset: 0 }, asset: { kind: 'video', width: 1280, height: 720 } },
      { clip: { at: 300, length: 300, offset: 0 }, asset: { kind: 'image', width: 1080, height: 1080 } },
      { clip: { at: 1400, length: 400, offset: 0 }, asset: { kind: 'image', width: 800, height: 600 } },
    ]

    const runs = 50
    const totalSimulatedFrames = runs * totalFrames
    const t0 = performance.now()

    let activeDrawOps = 0
    for (let r = 0; r < runs; r++) {
      for (let frameIndex = 0; frameIndex < totalFrames; frameIndex++) {
        // Frame compositing pipeline matching clientExporter.ts:
        // 1. Determine active visual clips at current frame
        for (let l = 0; l < layers.length; l++) {
          const { clip, asset } = layers[l]
          if (isClipActiveAtFrame(clip.at, clip.length, frameIndex)) {
            // 2. Source time computation
            const sourceTime = calculateSourceTimeSeconds(clip.at, clip.offset, frameIndex, fps)
            // 3. Aspect-ratio letterbox destination rect computation
            const rect = calculateLetterbox(width, height, asset.width, asset.height)
            activeDrawOps += (rect.dWidth > 0 && sourceTime >= 0) ? 1 : 0
          }
        }
      }
    }

    const t1 = performance.now()
    const durationMs = t1 - t0
    const avgMsPer60sExport = durationMs / runs
    const realTimeFactor = (60 * 1000) / avgMsPer60sExport

    assert.ok(activeDrawOps > 0)
    formatBenchmark('Client Export Frame Compositing Loop', totalSimulatedFrames, durationMs, 'frames/sec', 'µs/frame', 1e3)
    console.log(`     ↳ 60s Video (1.8k frames) evaluated in ${avgMsPer60sExport.toFixed(2)} ms (${realTimeFactor.toFixed(0)}x faster than real-time)`)
  })

  console.log('================================================================================\n')
})

