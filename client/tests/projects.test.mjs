import test from 'node:test'
import assert from 'node:assert/strict'
import { setActiveProjectId, getActiveProjectId, contentURL } from '../src/features/editor/api/client.ts'
import { parseRoute } from '../src/route.ts'

test('route parsing distinguishes the home page from the lobby', () => {
  assert.deepEqual(parseRoute(''), { view: 'home' })
  assert.deepEqual(parseRoute('#'), { view: 'home' })
  assert.deepEqual(parseRoute('#/'), { view: 'lobby' })
  assert.deepEqual(parseRoute('#/unknown'), { view: 'lobby' })
})

test('route parsing handles room and project hashes with parameters', () => {
  assert.deepEqual(parseRoute('#/room/room_123'), {
    view: 'editor',
    roomId: 'room_123',
  })
  assert.deepEqual(parseRoute('#/room/room_abc?peer=client_1'), {
    view: 'editor',
    roomId: 'room_abc',
  })
  assert.deepEqual(parseRoute('#/project/proj_123'), {
    view: 'editor',
    roomId: 'proj_123',
  })
})

test('active project id and scoped asset URL generator', () => {
  setActiveProjectId('room_collab_42')
  assert.equal(getActiveProjectId(), 'room_collab_42')
  assert.equal(
    contentURL('asset_1'),
    '/api/projects/room_collab_42/assets/asset_1/content'
  )

  // Direct override
  assert.equal(
    contentURL('asset_2', 'custom_room'),
    '/api/projects/custom_room/assets/asset_2/content'
  )
})
