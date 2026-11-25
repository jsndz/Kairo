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

function formatBenchmark(name, iterations, durationMs) {
  const opsPerSec = (iterations / (durationMs / 1000)).toLocaleString('en-US', { maximumFractionDigits: 1 })
  const nsPerOp = ((durationMs * 1e6) / iterations).toLocaleString('en-US', { maximumFractionDigits: 1 })
  console.log(`  [BENCHMARK] ${name.padEnd(45)}: ${opsPerSec.padStart(12)} ops/sec (${nsPerOp.padStart(8)} ns/op)`)
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

  console.log('================================================================================\n')
})
