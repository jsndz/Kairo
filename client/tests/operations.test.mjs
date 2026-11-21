import test from 'node:test'
import assert from 'node:assert/strict'
import { edgeTrim } from '../src/features/editor/state/operations.ts'

const clip = { at: 4, offset: 8, length: 30 }

test('edge trims preserve the opposite edge and respect source bounds', () => {
  assert.deepEqual(edgeTrim(clip, 120, 'left', -100), { at: 0, offset: 4, length: 34 })
  assert.deepEqual(edgeTrim(clip, 120, 'left', 100), { at: 33, offset: 37, length: 1 })
  assert.deepEqual(edgeTrim(clip, 120, 'right', 1000), { at: 4, offset: 8, length: 112 })
})
