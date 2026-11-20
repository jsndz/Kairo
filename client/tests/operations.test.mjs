import test from 'node:test'
import assert from 'node:assert/strict'
import { AUDIO_TRACK, MAX_FRAMES, VIDEO_TRACK, emptyProject } from '../src/features/editor/model.ts'
import { applyOperation, edgeTrim } from '../src/features/editor/operations.ts'
import { migrateProject } from '../src/features/editor/projectStorage.ts'

const asset = { id: 'source', name: 'Source', kind: 'video', frames: 120 }
const clip = { id: 'clip', mediaId: asset.id, trackId: VIDEO_TRACK, at: 4, offset: 8, length: 30, volume: 1 }
const add = { id: 'op-1', type: 'add', asset, clip }
const base = emptyProject()
const project = applyOperation(base, add).project

test('serialized operations can be replayed without modifying the previous project', () => {
  const edits = [
    add,
    { id: 'op-2', type: 'move', clipId: 'clip', trackId: VIDEO_TRACK, at: 10 },
    { id: 'op-3', type: 'trim', clipId: 'clip', at: 12, offset: 10, length: 22 },
    { id: 'op-4', type: 'volume', clipId: 'clip', volume: 0.3 },
    { id: 'op-5', type: 'split', clipId: 'clip', frame: 19, rightId: 'right' },
    { id: 'op-6', type: 'delete', clipId: 'clip' },
  ]
  const replay = list => list.reduce((state, edit) => {
    const result = applyOperation(state, edit)
    assert.equal(result.error, undefined)
    return result.project
  }, base)
  assert.deepEqual(replay(edits), replay(JSON.parse(JSON.stringify(edits))))
  assert.deepEqual(replay(edits).clips, [{ ...clip, id: 'right', at: 19, offset: 17, length: 15, volume: 0.3 }])
  assert.deepEqual(base.clips, [])
})

test('touching clip edges are legal, same-track overlaps are rejected, and different tracks can overlap', () => {
  const second = { ...add, id: 'op-2', clip: { ...clip, id: 'second', at: 34 } }
  const touching = applyOperation(project, second).project
  assert.equal(touching.clips.length, 2)
  const move = { id: 'op-3', type: 'move', clipId: 'second', trackId: VIDEO_TRACK, at: 33 }
  const trim = { id: 'op-4', type: 'trim', clipId: 'clip', at: 4, offset: 8, length: 31 }
  for (const attempt of [move, trim, { ...second, clip: { ...second.clip, at: 0 } }]) {
    const rejected = applyOperation(touching, attempt.type === 'add' ? { ...attempt, clip: { ...attempt.clip, id: 'third' } } : attempt)
    assert.equal(rejected.project, touching)
    assert.match(rejected.error, /overlap/)
  }
  const stacked = applyOperation(touching, { id: 'op-5', type: 'add-track', trackId: 'upper' }).project
  const result = applyOperation(stacked, { ...move, trackId: 'upper', at: 4 })
  assert.equal(result.error, undefined)
  assert.equal(result.project.clips[1].trackId, 'upper')
  assert.equal(result.project.clips[1].at, 4)
})

test('invalid edits are atomic and identity is retained on no-op edits', () => {
  const invalid = [
    { ...add },
    { id: 'move', type: 'move', clipId: 'clip', trackId: AUDIO_TRACK, at: 4 },
    { id: 'outside', type: 'move', clipId: 'clip', trackId: VIDEO_TRACK, at: MAX_FRAMES },
    { id: 'split-start', type: 'split', clipId: 'clip', frame: 4, rightId: 'right' },
    { id: 'split-end', type: 'split', clipId: 'clip', frame: 34, rightId: 'right' },
    { id: 'split-id', type: 'split', clipId: 'clip', frame: 10, rightId: 'clip' },
    { id: 'trim', type: 'trim', clipId: 'clip', at: 4, offset: 119, length: 2 },
    { id: 'volume', type: 'volume', clipId: 'clip', volume: NaN },
    { id: 'deleted', type: 'delete', clipId: 'unknown' },
  ]
  for (const attempt of invalid) {
    const result = applyOperation(project, attempt)
    assert.ok(result.error, attempt.id)
    assert.equal(result.project, project)
  }
  assert.deepEqual(applyOperation(project, { id: 'noop', type: 'volume', clipId: 'clip', volume: 1 }), { project })
})

test('edge trims preserve the opposite edge and respect source bounds', () => {
  assert.deepEqual(edgeTrim(clip, 120, 'left', -100), { at: 0, offset: 4, length: 34 })
  assert.deepEqual(edgeTrim(clip, 120, 'left', 100), { at: 33, offset: 37, length: 1 })
  assert.deepEqual(edgeTrim(clip, 120, 'right', 1000), { at: 4, offset: 8, length: 112 })
})

test('legacy overlapping clips retain their layer order and missing clips remain editable', () => {
  const { trackId: _track, ...legacy } = clip
  const existing = migrateProject({ name: 'Saved', clips: [
    legacy, { ...legacy, id: 'upper', at: 10 }, { ...legacy, id: 'missing', mediaId: 'gone' },
  ] }, [asset])
  assert.deepEqual(existing.clips.map(item => item.trackId), ['video-1', 'video-2', 'video-3'])
  assert.equal(existing.assets.gone.frames, 38)
  assert.equal(existing.clips[2].offset, 8)
  const split = applyOperation(existing, { id: 'op', type: 'split', clipId: 'missing', frame: 20, rightId: 'tail' })
  assert.equal(split.error, undefined)
  assert.equal(split.project.clips.at(-1).mediaId, 'gone')
})
