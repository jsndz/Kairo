import { lazy, Suspense, useEffect, useRef, useState } from 'react'
import type { PlayerRef } from '@remotion/player'
import { Aperture, ArrowLeft, Check, Download, FolderPlus, Redo2, Share2, Undo2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  MediaBin,
  Preview,
  Timeline,
  Inspector,
  PermissionPrompt,
  ConsensusModal,
  restoreMedia,
  useLocalMedia,
  useEditor,
  useAvailableMedia,
  useNoticeDismiss,
  useKeyboardShortcuts,
  exportProjectToJSON,
} from './features/editor'
import { ExportModal, ExportTriggerButton } from './features/export'
import { Lobby, saveRecentRoom } from './features/lobby'
import { parseRoute, type Route } from './route'
import './App.css'

const LandingPage = lazy(() => import('./features/landing/LandingPage'))

function EditorView({ roomId }: { roomId: string }) {
  const player = useRef<PlayerRef>(null)
  const [copied, setCopied] = useState(false)

  const project = useEditor(state => state.project)
  const canUndo = useEditor(state => state.canUndo)
  const canRedo = useEditor(state => state.canRedo)
  const connected = useEditor(state => state.connected)
  const peerPresence = useEditor(state => state.peerPresence)
  const files = useLocalMedia(state => state.files)
  const notice = useEditor(state => state.notice)
  const setNotice = useEditor(state => state.setNotice)
  const ready = useEditor(state => state.ready)
  const error = useEditor(state => state.error)
  const add = useEditor(state => state.add)
  const revision = useEditor(state => state.revision)

  const { available, markFailed } = useAvailableMedia(project.assets, files)
  const hasIncompleteAssets = project.clips.some(
    c => project.assets[c.mediaId]?.status !== 'completed'
  )

  useEffect(() => {
    void useEditor.getState().initialize(roomId)
  }, [roomId])

  useEffect(() => {
    if (ready) {
      void restoreMedia()
      saveRecentRoom({ roomId, name: project.name })
    }
  }, [ready, roomId, project.name])

  useNoticeDismiss(notice, setNotice)
  useKeyboardShortcuts(player, ready)

  function handleDownloadProject() {
    exportProjectToJSON(project)
    setNotice('Project data downloaded. Media files are stored separately.')
  }

  function handleShareLink() {
    const inviteUrl =
      window.location.origin + window.location.pathname + '#/room/' + encodeURIComponent(roomId)
    navigator.clipboard.writeText(inviteUrl).catch(() => {})
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
    setNotice('Invite link copied to clipboard! Share it with your collaborator.')
  }

  function handleLeaveRoom() {
    window.location.hash = '#/'
  }

  if (!ready) {
    return (
      <main className="dark editor-shell">
        <div className="media-empty">
          <p role={error ? 'alert' : 'status'}>{error ?? 'Joining collaboration room…'}</p>
          <div className="mt-4 flex items-center justify-center gap-3">
            {error && (
              <Button variant="outline" onClick={() => void useEditor.getState().initialize(roomId)}>
                Retry
              </Button>
            )}
            <Button variant="ghost" onClick={handleLeaveRoom}>
              Leave Room
            </Button>
          </div>
        </div>
      </main>
    )
  }

  return (
    <main className="dark editor-shell">
      <header className="editor-header">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleLeaveRoom}
            className="gap-1.5 px-2 text-xs text-[#9b9aa8] hover:text-white"
            title="Leave Room"
          >
            <ArrowLeft size={15} />
            <span>Leave Room</span>
          </Button>
          <div className="h-4 w-px bg-[#282830]" />
          <div className="brand">
            <span className="brand-mark">
              <Aperture size={18} />
            </span>
            <span>Kairo</span>
          </div>
        </div>

        <div className="project-heading">
          <input
            key={project.name}
            aria-label="Room name"
            defaultValue={project.name}
            maxLength={80}
            onBlur={event => {
              const name = event.currentTarget.value.trim()
              const editor = useEditor.getState()
              if (name && name !== editor.project.name) editor.dispatch({ type: 'rename', name })
              else event.currentTarget.value = editor.project.name
            }}
            onKeyDown={event => {
              if (event.key === 'Enter') event.currentTarget.blur()
            }}
          />
          <span className={`local-status ${connected ? 'text-emerald-400' : 'text-amber-400'}`}>
            <Check size={13} /> {connected ? 'Room Active' : 'Connecting…'}
          </span>
          {peerPresence && (
            <span className="local-status text-sky-400 font-medium" title="Peer connected">
              ● Peer in room
            </span>
          )}
        </div>

        <div className="header-actions">
          <Button
            variant="outline"
            size="sm"
            onClick={handleShareLink}
            className="gap-1.5 px-2.5 text-xs border-[#282830] text-[#9b9aa8] hover:text-white hover:bg-[#202027]"
            title="Copy invite link to clipboard"
          >
            {copied ? <Check size={14} className="text-emerald-400" /> : <Share2 size={14} />}
            <span>{copied ? 'Copied Link!' : 'Share Invite'}</span>
          </Button>
          <div className="history-actions">
            <Button
              variant="ghost"
              size="icon"
              aria-label="Undo"
              title="Undo"
              disabled={!canUndo}
              onClick={() => useEditor.getState().undo()}
            >
              <Undo2 />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Redo"
              title="Redo"
              disabled={!canRedo}
              onClick={() => useEditor.getState().redo()}
            >
              <Redo2 />
            </Button>
          </div>
          <Button
            variant="outline"
            size="icon"
            aria-label="Reset timeline"
            title="Reset timeline"
            onClick={() => {
              const editor = useEditor.getState()
              player.current?.pause()
              player.current?.seekTo(0)
              if (peerPresence) {
                void editor.proposeConsensus({ type: 'reset' }, 'Reset Timeline')
              } else {
                void editor.dispatch({ type: 'reset' })
              }
            }}
          >
            <FolderPlus />
          </Button>
          <Button className="download-button" aria-label="Download project data" onClick={handleDownloadProject}>
            <Download /> <span>Download</span>
          </Button>
          <ExportTriggerButton />
        </div>
      </header>
      <div className="workspace">
        <MediaBin assets={project.assets} media={available} add={add} />
        <Preview clips={project.clips} tracks={project.tracks} media={available} player={player} onMediaError={markFailed} />
        <Inspector media={available} />
      </div>
      <Timeline media={available} player={player} />
      <PermissionPrompt />
      <ConsensusModal />
      <ExportModal
        revision={revision}
        clipsCount={project.clips.length}
        hasIncompleteAssets={hasIncompleteAssets}
        projectId={roomId}
        clips={project.clips}
        tracks={project.tracks}
        media={available}
      />
      {error && (
        <div className="editor-toast" role="alert">
          {error}
          <Button variant="ghost" size="sm" onClick={() => useEditor.setState({ error: null })}>
            Dismiss
          </Button>
        </div>
      )}
      {notice && (
        <div className="editor-toast" role="status">
          {notice}{' '}
          <Button variant="ghost" size="sm" onClick={() => setNotice(null)}>
            Dismiss
          </Button>
        </div>
      )}
      {ready && !connected && (
        <div
          className="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50"
          role="alert"
        >
          <div className="bg-neutral-900 border border-neutral-700 rounded-lg p-6 max-w-sm text-center shadow-xl">
            <h3 className="font-semibold text-lg text-neutral-100 mb-2">Connection Lost</h3>
            <p className="text-neutral-400 text-sm mb-4">
              Reconnecting to the collaboration room… Editing is paused until connection is restored.
            </p>
            <div className="inline-block animate-spin w-5 h-5 border-2 border-emerald-500 border-t-transparent rounded-full" />
          </div>
        </div>
      )}
    </main>
  )
}

function App() {
  const [route, setRoute] = useState<Route>(() => parseRoute(window.location.hash))

  useEffect(() => {
    const handleHashChange = () => {
      setRoute(parseRoute(window.location.hash))
    }
    window.addEventListener('hashchange', handleHashChange)
    return () => window.removeEventListener('hashchange', handleHashChange)
  }, [])

  useEffect(() => {
    if (route.view !== 'editor') {
      useEditor.getState().reset()
    }
  }, [route.view])

  if (route.view === 'home') {
    return (
      <Suspense fallback={null}>
        <LandingPage />
      </Suspense>
    )
  }

  if (route.view === 'editor') {
    return <EditorView roomId={route.roomId} />
  }

  return (
    <Lobby
      onEnterRoom={id => {
        window.location.hash = `#/room/${encodeURIComponent(id)}`
      }}
    />
  )
}

export default App
