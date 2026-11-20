import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { PlayerRef } from '@remotion/player'
import { Aperture, Check, Download, FolderPlus, Redo2, Undo2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { MediaBin } from './features/editor/MediaBin'
import { Preview } from './features/editor/Preview'
import { Timeline } from './features/editor/Timeline'
import { Inspector } from './features/editor/Inspector'
import { loadMedia, mediaKey } from './features/editor/media'
import { demoMedia } from './features/editor/demo'
import { useEditor } from './features/editor/store'
import './App.css'

function App() {
  const player = useRef<PlayerRef>(null)
  const project = useEditor(state => state.project)
  const canUndo = useEditor(state => state.past.length > 0)
  const canRedo = useEditor(state => state.future.length > 0)
  const { data: media = demoMedia, isError, isPending } = useQuery({ queryKey: mediaKey, queryFn: loadMedia, staleTime: Infinity, retry: false, networkMode: 'always' })
  const [notice, setNotice] = useState('')
  const ready = useEditor(state => state.ready)
  const [failed, setFailed] = useState<ReadonlySet<string>>(() => new Set())
  const error = useEditor(state => state.error)
  const storageError = useEditor(state => state.storageError)
  const add = useEditor(state => state.add)
  const available = useMemo(() => media.filter(asset => !failed.has(asset.src)), [media, failed])
  const markFailed = useCallback((src: string) => setFailed(previous => previous.has(src) ? previous : new Set([...previous, src])), [])

  useEffect(() => {
    if (isPending || ready) return
    useEditor.getState().initialize(media)
  }, [isPending, media, ready])

  useEffect(() => {
    if (!ready) return
    function keydown(event: KeyboardEvent) {
      const target = event.target
      if (event.defaultPrevented || (target instanceof HTMLElement && target.closest('input, textarea, select, [contenteditable=true]'))) return
      const editor = useEditor.getState()
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'z') {
        event.preventDefault()
        if (event.shiftKey) editor.redo()
        else editor.undo()
      } else if (!event.metaKey && !event.ctrlKey && !event.altKey) {
        if (event.key === ' ' && !(target instanceof HTMLElement && target.closest('button'))) {
          event.preventDefault()
          if (editor.project.clips.length) player.current?.toggle()
        } else if (event.key.toLowerCase() === 's') editor.split(player.current?.getCurrentFrame() ?? 0)
        else if (event.key === 'Delete' || event.key === 'Backspace') {
          event.preventDefault()
          editor.remove()
        }
      }
    }
    window.addEventListener('keydown', keydown)
    return () => window.removeEventListener('keydown', keydown)
  }, [ready])

  function downloadProject() {
    const blob = new Blob([JSON.stringify({ version: 2, project }, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = project.name + '.kairo.json'
    anchor.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    setNotice('Project data downloaded. Media files are stored separately.')
  }

  if (!ready) return <main className="dark editor-shell"><p className="media-empty">Opening local project…</p></main>

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
        <span className="local-status"><Check size={13} /> Local project</span>
      </div>
      <div className="header-actions">
        <div className="history-actions">
          <Button variant="ghost" size="icon" aria-label="Undo" title="Undo" disabled={!canUndo} onClick={() => useEditor.getState().undo()}><Undo2 /></Button>
          <Button variant="ghost" size="icon" aria-label="Redo" title="Redo" disabled={!canRedo} onClick={() => useEditor.getState().redo()}><Redo2 /></Button>
        </div>
        <Button variant="outline" size="icon" aria-label="New empty project" title="New empty project" onClick={() => {
          const editor = useEditor.getState()
          player.current?.pause()
          player.current?.seekTo(0)
          editor.dispatch({ type: 'reset' })
        }}><FolderPlus /></Button>
        <Button className="download-button" aria-label="Download project data" onClick={downloadProject}><Download /> <span>Download project</span></Button>
      </div>
    </header>
    <div className="workspace">
      <MediaBin media={available} add={add} />
      <Preview clips={project.clips} tracks={project.tracks} media={available} player={player} onMediaError={markFailed} />
      <Inspector media={available} />
    </div>
    <Timeline media={available} player={player} />
    {(error || isError || storageError) && <div className="editor-toast" role="alert">
      {error ?? (storageError ? 'Project changes could not be saved to this browser.' : 'Local media storage could not be opened.')}
      {error && <Button variant="ghost" size="sm" onClick={() => useEditor.setState({ error: null })}>Dismiss</Button>}
    </div>}
    {notice && <div className="editor-toast" role="status">{notice} <Button variant="ghost" size="sm" onClick={() => setNotice('')}>Dismiss</Button></div>}
  </main>
}

export default App
