import { useRef, useState, type PointerEvent } from 'react'
import { DndContext, KeyboardSensor, PointerSensor, useDraggable, useSensor, useSensors, type DragEndEvent } from '@dnd-kit/core'
import { Film, AudioLines, Minus, Plus, Scissors, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { duration, FPS, MAX_FRAMES, timecode, type Clip, type Media } from './model'
import { useEditor } from './store'
import { usePlaybackFrame, type Playback } from './playback'
import { Waveform } from './Waveform'

function TrimHandle({ edge, clip, asset, scale, preview }: {
  edge: 'left' | 'right'; clip: Clip; asset: Media; scale: number; preview: (clip: Clip | null) => void
}) {
  const origin = useRef<number | null>(null)
  const update = useEditor(state => state.update)

  function trim(delta: number) {
    if (edge === 'left') {
      const shift = Math.max(-Math.min(clip.at, clip.offset), Math.min(clip.length - 1, delta))
      return { ...clip, at: clip.at + shift, offset: clip.offset + shift, length: clip.length - shift }
    }
    return { ...clip, length: Math.max(1, Math.min(asset.frames - clip.offset, MAX_FRAMES - clip.at, clip.length + delta)) }
  }

  return <button type="button" className={`trim-handle ${edge}`} aria-label={edge === 'left' ? 'Trim clip start' : 'Trim clip end'}
    title={edge === 'left' ? 'Drag to trim start' : 'Drag to trim end'}
    onPointerDown={event => {
      if (event.button !== 0) return
      origin.current = event.clientX
      event.currentTarget.setPointerCapture(event.pointerId)
    }}
    onPointerMove={event => {
      if (origin.current !== null) preview(trim(Math.round((event.clientX - origin.current) / scale)))
    }}
    onPointerUp={event => {
      if (origin.current !== null) {
        const { at, offset, length } = trim(Math.round((event.clientX - origin.current) / scale))
        update(clip.id, { at, offset, length }, asset)
      }
      origin.current = null
      preview(null)
    }}
    onLostPointerCapture={() => { origin.current = null; preview(null) }}
    onKeyDown={event => {
      if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
      event.preventDefault()
      const { at, offset, length } = trim((event.key === 'ArrowLeft' ? -1 : 1) * (event.shiftKey ? FPS : 1))
      update(clip.id, { at, offset, length }, asset)
    }} />
}

function TimelineClip({ clip, asset, scale }: { clip: Clip; asset: Media; scale: number }) {
  const selected = useEditor(state => state.selected === clip.id)
  const select = useEditor(state => state.select)
  const { attributes, listeners, setNodeRef, transform } = useDraggable({ id: clip.id })
  const [preview, setPreview] = useState<Clip | null>(null)
  const shown = preview ?? clip
  return <div ref={setNodeRef} className={`timeline-clip ${asset.kind} ${selected ? 'selected' : ''}`}
    style={{ left: shown.at * scale, width: Math.max(4, shown.length * scale), transform: transform ? `translateX(${Math.max(-clip.at * scale, transform.x)}px)` : undefined,
      backgroundImage: asset.poster ? `linear-gradient(180deg,#171327db,#251c3f69),url("${asset.poster}")` : undefined }}>
    <button type="button" {...attributes} {...listeners} aria-pressed={selected} className="clip-body"
      title={asset.name + ' · ' + timecode(shown.length)} onClick={() => select(clip.id)}>
      <span className="clip-name">{asset.kind === 'video' ? <Film size={12} /> : <AudioLines size={12} />}{asset.name}</span>
      {asset.kind === 'audio' && asset.frames <= FPS * 600 && <Waveform src={asset.src} {...shown} frames={asset.frames} />}
    </button>
    {selected && (['left', 'right'] as const).map(edge =>
      <TrimHandle key={edge} edge={edge} clip={clip} asset={asset} scale={scale} preview={setPreview} />)}
  </div>
}

function Ruler({ player, scale, frames, ticks }: { player: Playback; scale: number; frames: number; ticks: number[] }) {
  const frame = usePlaybackFrame(player)
  function seek(value: number) {
    player.current?.seekTo(Math.max(0, Math.min(frames - 1, Math.round(value))))
  }
  function scrub(event: PointerEvent<HTMLInputElement>) {
    seek((event.clientX - event.currentTarget.getBoundingClientRect().left) / scale)
  }
  return <div className="ruler">
    {ticks.map(tick => <span key={tick} className="tick technical" style={{ left: tick * scale }}>{timecode(tick, false)}</span>)}
    <input type="range" aria-label="Timeline playhead" aria-valuetext={timecode(frame)} min={0} max={frames - 1} step={1}
      value={Math.min(frame, frames - 1)} onChange={event => seek(+event.target.value)}
      onPointerDown={event => {
        if (event.button !== 0) return
        event.preventDefault()
        event.currentTarget.focus()
        event.currentTarget.setPointerCapture(event.pointerId)
        scrub(event)
      }}
      onPointerMove={event => { if (event.currentTarget.hasPointerCapture(event.pointerId)) scrub(event) }} />
  </div>
}

function Playhead({ player, scale }: { player: Playback; scale: number }) {
  const frame = usePlaybackFrame(player)
  return <div className="playhead" style={{ left: frame * scale }} aria-hidden="true" />
}

export function Timeline({ media, player }: { media: Media[]; player: Playback }) {
  const clips = useEditor(state => state.project.clips)
  const selected = useEditor(state => state.selected)
  const split = useEditor(state => state.split)
  const remove = useEditor(state => state.remove)
  const update = useEditor(state => state.update)
  const select = useEditor(state => state.select)
  const [zoom, setZoom] = useState(100)
  const scale = (45 / FPS) * zoom / 100
  const extent = Math.max(20 * FPS, duration(clips) + FPS * 2)
  const tickStep = FPS * Math.max(1, Math.ceil(66 / (scale * FPS)), Math.ceil(extent / FPS / 240))
  const ticks = Array.from({ length: Math.floor(extent / tickStep) + 1 }, (_, i) => i * tickStep)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: (event, { currentCoordinates }) => {
      if (event.code !== 'ArrowLeft' && event.code !== 'ArrowRight') return undefined
      return { ...currentCoordinates, x: currentCoordinates.x + (event.code === 'ArrowLeft' ? -scale : scale) * (event.shiftKey ? FPS : 1) }
    } }))

  function move({ active, activatorEvent, delta }: DragEndEvent) {
    const clip = clips.find(item => item.id === active.id)
    const asset = media.find(item => item.id === clip?.mediaId)
    if (!clip || !asset || !delta.x) return
    const at = Math.round(clip.at + delta.x / scale)
    const edges = [0, ...clips.filter(item => item.id !== clip.id).flatMap(item => [item.at, item.at + item.length])]
    const nearest = edges.flatMap(edge => [edge, edge - clip.length]).filter(edge => edge >= 0)
      .reduce((best, edge) => Math.abs(edge - at) < Math.abs(best - at) ? edge : best, Infinity)
    update(clip.id, { at: !(activatorEvent instanceof KeyboardEvent) && Math.abs(nearest - at) * scale <= 6 ? nearest : at }, asset)
  }

  return <section className="timeline-panel" aria-label="Timeline">
    <div className="panel-heading timeline-heading">
      <div className="timeline-tools">
        <h2>Timeline</h2>
        <Button variant="ghost" size="icon" title="Split at playhead (S)" aria-label="Split at playhead" disabled={!selected} onClick={() => split(player.current?.getCurrentFrame() ?? 0)}><Scissors /></Button>
        <Button variant="ghost" size="icon" title="Delete clip" aria-label="Delete clip" disabled={!selected} onClick={remove}><Trash2 /></Button>
      </div>
      <div className="zoom-controls">
        <Button variant="ghost" size="icon-sm" aria-label="Zoom out" disabled={zoom <= 50} onClick={() => setZoom(Math.max(50, zoom - 25))}><Minus /></Button>
        <input aria-label="Timeline zoom" type="range" min={50} max={250} step={25} value={zoom} onChange={event => setZoom(+event.target.value)} />
        <Button variant="ghost" size="icon-sm" aria-label="Zoom in" disabled={zoom >= 250} onClick={() => setZoom(Math.min(250, zoom + 25))}><Plus /></Button>
        <span className="technical zoom-value">{zoom}%</span>
      </div>
    </div>
    <div className="timeline-grid">
      <div className="track-labels">
        <div className="ruler-label technical">TRACKS</div>
        <div><Film size={15} /> Video</div>
        <div><AudioLines size={15} /> Audio</div>
      </div>
      <div className="timeline-scroll">
        <div className="timeline-content" style={{ width: extent * scale, '--tick': tickStep * scale + 'px' } as React.CSSProperties}>
          <Ruler player={player} scale={scale} frames={duration(clips)} ticks={ticks} />
          <DndContext sensors={sensors} onDragStart={event => select(String(event.active.id))} onDragEnd={move}>
            {(['video', 'audio'] as const).map(kind => <div className="track-lane" key={kind}>
              {clips.map(clip => {
                const asset = media.find(item => item.id === clip.mediaId && item.kind === kind)
                return asset && <TimelineClip key={clip.id} clip={clip} asset={asset} scale={scale} />
              })}
            </div>)}
          </DndContext>
          <Playhead player={player} scale={scale} />
        </div>
      </div>
    </div>
    <div className="timeline-footer"><span>Drag clips to move · Select a clip to trim or split</span><span className="technical">{clips.length} {clips.length === 1 ? 'CLIP' : 'CLIPS'}</span></div>
  </section>
}
