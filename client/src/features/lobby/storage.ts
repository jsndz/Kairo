import type { RecentRoom } from './types'

const STORAGE_KEY = 'kairo_recent_rooms'

export function getRecentRooms(): RecentRoom[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

export function saveRecentRoom(room: { roomId: string; name?: string }): void {
  try {
    const current = getRecentRooms()
    const filtered = current.filter(r => r.roomId !== room.roomId)
    const updated: RecentRoom = {
      roomId: room.roomId,
      name: room.name?.trim() || 'Collaboration Room',
      lastVisited: new Date().toISOString(),
    }
    const next = [updated, ...filtered].slice(0, 10)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  } catch {}
}

export function removeRecentRoom(roomId: string): void {
  try {
    const current = getRecentRooms()
    const next = current.filter(r => r.roomId !== roomId)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  } catch {}
}
