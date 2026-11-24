import { useEffect, useRef } from 'react'
import type { PlayerRef } from '@remotion/player'
import { Aperture, Check, Download, FolderPlus, Redo2, Undo2 } from 'lucide-react'
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
import './App.css'

function App() {
  const player = useRef<PlayerRef>(null)
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

  useEffect(() => { void useEditor.getState().initialize() }, [])
  useEffect(() => { if (ready) void restoreMedia() }, [ready])

  useNoticeDismiss(notice, setNotice)
  useKeyboardShortcuts(player, ready)

  function handleDownloadProject() {
    exportProjectToJSON(project)
    setNotice('Project data downloaded. Media files are stored separately.')
  }

  if (!ready) return <main className="dark editor-shell"><div className="media-empty">
    <p role={error ? 'alert' : 'status'}>{error ?? 'Opening project…'}</p>
    {error && <Button variant="outline" onClick={() => void useEditor.getState().initialize()}>Retry</Button>}
  </div></main>

  return <main className="dark editor-shell">
    <header className="editor-header">
      <div className="brand"><span className="brand-mark"><Aperture size={20} /></span><span>Kairo</span></div>
      <div className="project-heading">
        <input key={project.name} aria-label="Project name" defaultValue={project.name} maxLength={80} onBlur={event => {
          const name = event.currentTarget.value.trim()
          const editor = useEditor.getState()
          if (name && name !== editor.project.name) editor.dispatch({ type: 'rename', name })
          else event.currentTarget.value = editor.project.name
        }} onKeyDown={event => { if (event.key === 'Enter') event.currentTarget.blur() }} />
        <span className={`local-status ${connected ? 'text-emerald-400' : 'text-amber-400'}`}>
          <Check size={13} /> {connected ? 'Live Sync' : 'Reconnecting…'}
        </span>
        {peerPresence && (
          <span className="local-status text-sky-400 font-medium" title="Peer connected">
            ● Peer active
          </span>
        )}
      </div>
      <div className="header-actions">
        <div className="history-actions">
          <Button variant="ghost" size="icon" aria-label="Undo" title="Undo" disabled={!canUndo} onClick={() => useEditor.getState().undo()}><Undo2 /></Button>
          <Button variant="ghost" size="icon" aria-label="Redo" title="Redo" disabled={!canRedo} onClick={() => useEditor.getState().redo()}><Redo2 /></Button>
        </div>
        <Button variant="outline" size="icon" aria-label="Reset timeline" title="Reset timeline" onClick={() => {
          const editor = useEditor.getState()
          player.current?.pause()
          player.current?.seekTo(0)
          if (peerPresence) {
            void editor.proposeConsensus({ type: 'reset' }, 'Reset Timeline')
          } else {
            void editor.dispatch({ type: 'reset' })
          }
        }}><FolderPlus /></Button>
        <Button className="download-button" aria-label="Download project data" onClick={handleDownloadProject}><Download /> <span>Download project</span></Button>
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
      clips={project.clips}
      tracks={project.tracks}
      media={available}
    />
    {error && <div className="editor-toast" role="alert">
      {error}
      {error && <Button variant="ghost" size="sm" onClick={() => useEditor.setState({ error: null })}>Dismiss</Button>}
    </div>}
    {notice && <div className="editor-toast" role="status">{notice} <Button variant="ghost" size="sm" onClick={() => setNotice(null)}>Dismiss</Button></div>}
    {ready && !connected && (
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50"
        role="alert"
      >
        <div className="bg-neutral-900 border border-neutral-700 rounded-lg p-6 max-w-sm text-center shadow-xl">
          <h3 className="font-semibold text-lg text-neutral-100 mb-2">Connection Lost</h3>
          <p className="text-neutral-400 text-sm mb-4">
            Reconnecting to the collaboration server… Editing is paused until connection is restored.
          </p>
          <div className="inline-block animate-spin w-5 h-5 border-2 border-emerald-500 border-t-transparent rounded-full" />
        </div>
      </div>
    )}
  </main>
}

export default App
