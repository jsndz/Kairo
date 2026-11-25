import { useState, useEffect } from 'react'
import { Aperture, Plus, LogIn, Clock, Trash2, ArrowUpRight, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { createRoom } from '../api/client'
import { getRecentRooms, removeRecentRoom } from '../storage'
import type { RecentRoom } from '../types'

interface LobbyProps {
  onEnterRoom: (roomId: string) => void
}

function extractRoomId(input: string): string {
  const clean = input.trim()
  if (!clean) return ''
  // If user pasted a full URL or hash, e.g. "http://localhost:5173/#/room/xyz" or "#/room/xyz"
  const roomMatch = clean.match(/(?:#\/room\/|#\/project\/|\/room\/)([a-zA-Z0-9_-]+)/)
  if (roomMatch && roomMatch[1]) {
    return roomMatch[1]
  }
  return clean
}

function formatRelativeTime(isoStr: string): string {
  try {
    const d = new Date(isoStr)
    if (isNaN(d.getTime())) return 'Recently'
    const diffMs = Date.now() - d.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  } catch {
    return 'Recently'
  }
}

export function Lobby({ onEnterRoom }: LobbyProps) {
  const [newRoomName, setNewRoomName] = useState('')
  const [isCreating, setIsCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const [joinInput, setJoinInput] = useState('')
  const [joinError, setJoinError] = useState<string | null>(null)

  const [recentRooms, setRecentRooms] = useState<RecentRoom[]>([])

  useEffect(() => {
    setRecentRooms(getRecentRooms())
  }, [])

  async function handleCreateRoom(e: React.FormEvent) {
    e.preventDefault()
    setIsCreating(true)
    setCreateError(null)
    try {
      const res = await createRoom(newRoomName.trim() || undefined)
      onEnterRoom(res.roomId)
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Could not create room')
    } finally {
      setIsCreating(false)
    }
  }

  function handleJoinRoom(e: React.FormEvent) {
    e.preventDefault()
    setJoinError(null)
    const id = extractRoomId(joinInput)
    if (!id) {
      setJoinError('Please enter a valid room code or link')
      return
    }
    onEnterRoom(id)
  }

  function handleForgetRoom(roomId: string, e: React.MouseEvent) {
    e.stopPropagation()
    removeRecentRoom(roomId)
    setRecentRooms(prev => prev.filter(r => r.roomId !== roomId))
  }

  return (
    <div className="min-h-screen bg-[#101014] text-white flex flex-col font-sans">
      {/* Header */}
      <header className="border-b border-[#282830] bg-[#121216]/90 backdrop-blur-xs">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-[#c2aeff] text-[#191027] shadow-sm">
              <Aperture size={22} />
            </span>
            <div className="flex items-baseline gap-2">
              <h1 className="text-xl font-bold tracking-tight text-white">Kairo</h1>
              <span className="text-xs text-[#9b9aa8] font-medium">Collaboration Rooms</span>
            </div>
          </div>
        </div>
      </header>

      {/* Main Container */}
      <main className="flex-1 max-w-5xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-12 flex flex-col items-center">
        {/* Hero Section */}
        <div className="text-center max-w-2xl mx-auto mb-12">
          <div className="inline-flex items-center gap-2 rounded-full border border-[#282830] bg-[#141419] px-3.5 py-1 text-xs text-[#cabaff] mb-4">
            <Sparkles size={13} />
            <span>Isolated 2-Peer Timeline Collaboration</span>
          </div>
          <h2 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-white">
            Create or Join a Video Room
          </h2>
          <p className="mt-3 text-sm text-[#9b9aa8]">
            Each room is completely isolated with its own real-time timeline, multi-track editing, and media cache. Share your invite link to collaborate.
          </p>
        </div>

        {/* Dual Actions Grid: Create vs Join */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 w-full mb-14">
          {/* Create Room Card */}
          <div className="rounded-2xl border border-[#282830] bg-[#141419] p-6 sm:p-8 flex flex-col justify-between shadow-xl">
            <div>
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-[#241f36] text-[#b8a6ff] mb-5">
                <Plus size={24} />
              </div>
              <h3 className="text-xl font-bold text-white tracking-tight">Create a Room</h3>
              <p className="text-xs text-[#9b9aa8] mt-1.5 mb-6">
                Start a private, unlisted editing session. You will be able to share the direct invite link with your peer.
              </p>

              <form onSubmit={handleCreateRoom} className="space-y-4">
                <div>
                  <label htmlFor="create-name" className="block text-xs font-medium text-[#9b9aa8] mb-1.5">
                    Room Name (Optional)
                  </label>
                  <input
                    id="create-name"
                    type="text"
                    value={newRoomName}
                    onChange={e => setNewRoomName(e.target.value)}
                    placeholder="e.g. Travel Vlog Edit"
                    maxLength={80}
                    disabled={isCreating}
                    className="w-full rounded-lg border border-[#282830] bg-[#101014] px-3.5 py-2.5 text-sm text-white placeholder-[#9b9aa8]/50 focus:border-[#b8a6ff] focus:outline-none focus:ring-1 focus:ring-[#b8a6ff]"
                  />
                </div>

                {createError && (
                  <p className="text-xs text-rose-400 bg-rose-950/30 border border-rose-900/50 rounded-md p-2">
                    {createError}
                  </p>
                )}

                <Button
                  type="submit"
                  disabled={isCreating}
                  className="w-full bg-[#cabaff] text-[#1e1437] hover:bg-[#b8a6ff] font-semibold h-10 gap-2 shadow-sm"
                >
                  <Plus size={16} />
                  <span>{isCreating ? 'Creating Room…' : 'Start New Room'}</span>
                </Button>
              </form>
            </div>
          </div>

          {/* Join Room Card */}
          <div className="rounded-2xl border border-[#282830] bg-[#141419] p-6 sm:p-8 flex flex-col justify-between shadow-xl">
            <div>
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-[#1b2636] text-[#82b4ff] mb-5">
                <LogIn size={22} />
              </div>
              <h3 className="text-xl font-bold text-white tracking-tight">Join a Room</h3>
              <p className="text-xs text-[#9b9aa8] mt-1.5 mb-6">
                Have a Room Code or an invite link from a collaborator? Enter or paste it below to enter the live session.
              </p>

              <form onSubmit={handleJoinRoom} className="space-y-4">
                <div>
                  <label htmlFor="join-code" className="block text-xs font-medium text-[#9b9aa8] mb-1.5">
                    Room Code or Invite Link
                  </label>
                  <input
                    id="join-code"
                    type="text"
                    value={joinInput}
                    onChange={e => setJoinInput(e.target.value)}
                    placeholder="e.g. room-alpha-99 or paste full URL"
                    className="w-full rounded-lg border border-[#282830] bg-[#101014] px-3.5 py-2.5 text-sm text-white placeholder-[#9b9aa8]/50 focus:border-[#82b4ff] focus:outline-none focus:ring-1 focus:ring-[#82b4ff]"
                  />
                </div>

                {joinError && (
                  <p className="text-xs text-rose-400 bg-rose-950/30 border border-rose-900/50 rounded-md p-2">
                    {joinError}
                  </p>
                )}

                <Button
                  type="submit"
                  className="w-full bg-[#82b4ff] text-[#0d1c33] hover:bg-[#6ba3f7] font-semibold h-10 gap-2 shadow-sm"
                >
                  <LogIn size={16} />
                  <span>Join Session</span>
                </Button>
              </form>
            </div>
          </div>
        </div>

        {/* Recent Rooms (Private to device) */}
        <div className="w-full">
          <div className="flex items-center justify-between mb-4 border-b border-[#282830] pb-2">
            <div className="flex items-center gap-2">
              <Clock size={16} className="text-[#9b9aa8]" />
              <h4 className="text-sm font-semibold text-white">Recent Rooms on this Device</h4>
            </div>
            <span className="text-xs text-[#9b9aa8]">Private to this browser</span>
          </div>

          {recentRooms.length === 0 ? (
            <div className="rounded-xl border border-dashed border-[#282830] bg-[#141419]/30 p-8 text-center">
              <p className="text-xs text-[#9b9aa8]">
                Rooms you create or join will be remembered here on this device for quick access.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3.5">
              {recentRooms.map(room => (
                <div
                  key={room.roomId}
                  role="button"
                  tabIndex={0}
                  onClick={() => onEnterRoom(room.roomId)}
                  onKeyDown={e => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      onEnterRoom(room.roomId)
                    }
                  }}
                  className="group flex items-center justify-between rounded-xl border border-[#282830] bg-[#141419] p-4 text-left cursor-pointer hover:border-[#b8a6ff]/50 hover:bg-[#181820] transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-[#b8a6ff]"
                >
                  <div className="min-w-0 pr-2">
                    <h5 className="text-sm font-semibold text-white truncate group-hover:text-[#cabaff]">
                      {room.name || 'Collaboration Room'}
                    </h5>
                    <p className="text-xs font-mono text-[#9b9aa8]/70 truncate mt-0.5">
                      {room.roomId}
                    </p>
                    <span className="text-[11px] text-[#9b9aa8]/60 mt-1 block">
                      {formatRelativeTime(room.lastVisited)}
                    </span>
                  </div>

                  <div className="flex items-center gap-1.5 flex-shrink-0">
                    <Button
                      variant="ghost"
                      size="icon"
                      title="Forget this room"
                      onClick={e => handleForgetRoom(room.roomId, e)}
                      className="h-7 w-7 text-[#9b9aa8] hover:text-rose-400 hover:bg-rose-500/10"
                    >
                      <Trash2 size={13} />
                    </Button>
                    <ArrowUpRight size={15} className="text-[#cabaff] group-hover:translate-x-0.5 transition-transform" />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
