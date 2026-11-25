import test from 'node:test'
import assert from 'node:assert/strict'

function extractRoomId(input) {
  const clean = input.trim()
  if (!clean) return ''
  const roomMatch = clean.match(/(?:#\/room\/|#\/project\/|\/room\/)([a-zA-Z0-9_-]+)/)
  if (roomMatch && roomMatch[1]) {
    return roomMatch[1]
  }
  return clean
}

test('extractRoomId handles raw codes and URLs', () => {
  assert.equal(extractRoomId('room-abc-123'), 'room-abc-123')
  assert.equal(
    extractRoomId('http://localhost:5173/#/room/room-abc-123'),
    'room-abc-123'
  )
  assert.equal(
    extractRoomId('https://kairo.app/#/project/proj-456?peer=user1'),
    'proj-456'
  )
  assert.equal(extractRoomId('  my-custom-room  '), 'my-custom-room')
  assert.equal(extractRoomId(''), '')
})
