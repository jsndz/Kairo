import test from 'node:test'
import assert from 'node:assert/strict'
import {
  calculateScale,
  calculateExtent,
  calculateTickStep,
  generateRulerTicks,
  calculateDragTarget,
  calculateDragTransformX,
  orderCompositionClips,
  matchesMediaFilter,
} from '../src/features/editor/domain/index.ts'

test('timeline scale and extent calculations', () => {
  // 100% zoom with 30fps = 1.5 px/frame
  assert.equal(calculateScale(100, 30), 1.5)
  // 200% zoom with 30fps = 3.0 px/frame
  assert.equal(calculateScale(200, 30), 3.0)

  // Empty clips: minimum 20 seconds (600 frames at 30fps)
  assert.equal(calculateExtent([], 30), 600)

  // With a clip extending to frame 900: 900 + 60 = 960 frames
  const clips = [{ id: '1', trackId: 'v1', mediaId: 'm1', at: 0, offset: 0, length: 900, volume: 1 }]
  assert.equal(calculateExtent(clips, 30), 960)

  // Ruler ticks generation
  const ticks = generateRulerTicks(600, 1.5, 30)
  assert.ok(ticks.length > 0)
  assert.equal(ticks[0], 0)
})

test('clip drag bounds and transform math', () => {
  // Dragging right by 30px with scale 1.5 adds 20 frames
  const target1 = calculateDragTarget(10, 30, 1.5, 50, 1000)
  assert.equal(target1, 30)

  // Dragging left beyond frame 0 clamps to 0
  const target2 = calculateDragTarget(10, -100, 1.5, 50, 1000)
  assert.equal(target2, 0)

  // Dragging beyond maxFrames clamps to maxFrames - length
  const target3 = calculateDragTarget(980, 100, 1.5, 50, 1000)
  assert.equal(target3, 950)

  // Transform offset clamping
  assert.equal(calculateDragTransformX(50, 1.5, -200), -75)
  assert.equal(calculateDragTransformX(50, 1.5, 100), 100)
})

test('composition clip ordering by track order', () => {
  const tracks = [
    { id: 't-audio', kind: 'audio' },
    { id: 't-video', kind: 'video' },
  ]
  const media = [
    { id: 'm-vid', name: 'video.mp4', kind: 'video', src: 'blob:1', frames: 300, duration: 10, bytes: 100 },
    { id: 'm-aud', name: 'audio.mp3', kind: 'audio', src: 'blob:2', frames: 600, duration: 20, bytes: 50 },
  ]
  const clips = [
    { id: 'c2', trackId: 't-video', mediaId: 'm-vid', at: 0, offset: 0, length: 150, volume: 1 },
    { id: 'c1', trackId: 't-audio', mediaId: 'm-aud', at: 0, offset: 0, length: 300, volume: 1 },
  ]

  const ordered = orderCompositionClips(clips, tracks, media)
  assert.equal(ordered.length, 2)
  // Audio track index 0 comes before video track index 1
  assert.equal(ordered[0].clip.id, 'c1')
  assert.equal(ordered[1].clip.id, 'c2')
})

test('media search and kind filter matching', () => {
  assert.equal(matchesMediaFilter('Interview.mp4', 'video', 'inter', 'all'), true)
  assert.equal(matchesMediaFilter('Interview.mp4', 'video', 'audio', 'all'), false)
  assert.equal(matchesMediaFilter('Song.mp3', 'audio', 'song', 'video'), false)
  assert.equal(matchesMediaFilter('Song.mp3', 'audio', 'song', 'audio'), true)
})
