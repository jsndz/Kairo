import { memo, useEffect, useMemo, useRef, useState, type PointerEvent } from 'react'
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useDraggable,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import { Film, AudioLines, Minus, Plus, Scissors, Trash2, Image } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { duration, FPS, MAX_FRAMES, timecode, type Asset, type Clip, type Media } from '../state/model'
import { useEditor } from '../state/store'
import { usePlaybackFrame, type Playback } from '../state/playback'
import { edgeTrim } from '../state/operations'
import { Waveform } from './Waveform'

function TrimHandle({
  edge,
  clip,
  asset,
  scale,
  preview,
}: {
  edge: 'left' | 'right'
  clip: Clip
  asset: Asset
  scale: number
  preview: (clip: Clip | null) => void
}) {
  const origin = useRef<number | null>(null)
  const request = useRef<number | null>(null)
  const dispatch = useEditor(state => state.dispatch)
  const trim = (delta: number) => edgeTrim(clip, asset.frames, edge, delta)
  const commit = (delta: number) => void dispatch({ type: 'trim', clipId: clip.id, ...trim(delta) })

  function cancelPreview() {
    if (request.current !== null) cancelAnimationFrame(request.current)
    request.current = null
    origin.current = null
    preview(null)
  }

  useEffect(
    () => () => {
      if (request.current !== null) cancelAnimationFrame(request.current)
    },
    [],
  )

  return (
    <button
      type="button"
      className={`trim-handle ${edge}`}
      aria-label={edge === 'left' ? 'Trim clip start' : 'Trim clip end'}
      title={edge === 'left' ? 'Drag to trim start' : 'Drag to trim end'}
      onPointerDown={event => {
        if (event.button !== 0) return
        origin.current = event.clientX
        event.currentTarget.setPointerCapture(event.pointerId)
      }}
      onPointerMove={event => {
        if (origin.current === null) return
        if (request.current !== null) cancelAnimationFrame(request.current)
        const delta = Math.round((event.clientX - origin.current) / scale)
        request.current = requestAnimationFrame(() => {
          request.current = null
          preview({ ...clip, ...trim(delta) })
        })
      }}
      onPointerUp={event => {
        if (origin.current !== null) commit(Math.round((event.clientX - origin.current) / scale))
        cancelPreview()
      }}
      onLostPointerCapture={cancelPreview}
      onKeyDown={event => {
        if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
        event.preventDefault()
        commit((event.key === 'ArrowLeft' ? -1 : 1) * (event.shiftKey ? FPS : 1))
      }}
    />
  )
}

const ClipContent = memo(function ClipContent({
  asset,
  source,
  clip,
}: {
  asset: Asset
  source?: Media
  clip: Clip
}) {
  return (
    <>
      <span className="clip-name">
        {asset.kind === 'video' ? (
          <Film size={12} />
        ) : asset.kind === 'audio' ? (
          <AudioLines size={12} />
        ) : (
          <Image size={12} />
        )}
        {asset.name}
        {!source && ' · Missing media'}
      </span>
      {asset.kind === 'audio' && source && asset.frames <= FPS * 600 && (
        <Waveform src={source.src} offset={clip.offset} length={clip.length} frames={asset.frames} />
      )}
    </>
  )
})

const TimelineClip = memo(function TimelineClip({
  clip,
  asset,
  source,
  scale,
}: {
  clip: Clip
  asset: Asset
  source?: Media
  scale: number
}) {
  const selected = useEditor(state => state.selected === clip.id)
  const select = useEditor(state => state.select)
  const { attributes, listeners, setNodeRef, transform } = useDraggable({ id: clip.id })
  const [preview, setPreview] = useState<Clip | null>(null)
  const shown = preview ?? clip
  return (
    <div
      ref={setNodeRef}
      className={`timeline-clip ${asset.kind} ${selected ? 'selected' : ''} ${source ? '' : 'missing'}`}
      style={{
        left: shown.at * scale,
        width: Math.max(4, shown.length * scale),
        transform: transform ? `translateX(${Math.max(-clip.at * scale, transform.x)}px)` : undefined,
        backgroundImage: source?.poster ? `linear-gradient(180deg,#171327db,#251c3f69),url("${source.poster}")` : undefined,
      }}
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        aria-pressed={selected}
        className="clip-body"
        title={asset.name + ' · ' + timecode(shown.length)}
        onClick={() => select(clip.id)}
      >
        <ClipContent asset={asset} source={source} clip={shown} />
      </button>
      {selected &&
        (['left', 'right'] as const).map(edge => (
          <TrimHandle key={edge} edge={edge} clip={clip} asset={asset} scale={scale} preview={setPreview} />
        ))}
    </div>
  )
})

const RulerTicks = memo(function RulerTicks({ ticks, scale }: { ticks: number[]; scale: number }) {
  return ticks.map(tick => (
    <span key={tick} className="tick technical" style={{ left: tick * scale }}>
      {timecode(tick, false)}
    </span>
  ))
})

function Ruler({
  player,
  scale,
  frames,
  ticks,
}: {
  player: Playback
  scale: number
  frames: number
  ticks: number[]
}) {
  const frame = usePlaybackFrame(player)
  function seek(value: number) {
    player.current?.seekTo(Math.max(0, Math.min(frames - 1, Math.round(value))))
  }
  function scrub(event: PointerEvent<HTMLInputElement>) {
    seek((event.clientX - event.currentTarget.getBoundingClientRect().left) / scale)
  }
  return (
    <div className="ruler">
      <RulerTicks ticks={ticks} scale={scale} />
      <input
        type="range"
        aria-label="Timeline playhead"
        aria-valuetext={timecode(frame)}
        min={0}
        max={frames - 1}
        step={1}
        value={Math.min(frame, frames - 1)}
        onChange={event => seek(+event.target.value)}
        onPointerDown={event => {
          player.current?.pause()
          scrub(event)
        }}
        onPointerMove={event => {
          if (event.buttons === 1) scrub(event)
        }}
        className="playhead-scrubber"
      />
      <div className="playhead-line" style={{ left: Math.min(frame, frames - 1) * scale }}>
        <span className="playhead-needle" />
      </div>
    </div>
  )
}

export const Timeline = memo(function Timeline({
  media,
  player,
}: {
  media: Media[]
  player: Playback
}) {
  const project = useEditor(state => state.project)
  const dispatch = useEditor(state => state.dispatch)
  const selected = useEditor(state => state.selected)
  const [zoom, setZoom] = useState(1)
  const frames = Math.max(FPS * 10, duration(project.clips))
  const scale = (zoom * 4) / FPS
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 3 } }), useSensor(KeyboardSensor))

  const ticks = useMemo(() => {
    const step = zoom < 0.5 ? FPS * 10 : zoom < 1.5 ? FPS * 5 : zoom < 3 ? FPS * 2 : FPS
    const marks: number[] = []
    for (let f = 0; f <= frames; f += step) marks.push(f)
    return marks
  }, [frames, zoom])

  const assets = useMemo(() => new Map(Object.values(project.assets).map(a => [a.id, a])), [project.assets])
  const sources = useMemo(() => new Map(media.map(m => [m.id, m])), [media])

  function dragEnd(event: DragEndEvent) {
    const clip = project.clips.find(item => item.id === event.active.id)
    if (!clip) return
    const shift = Math.round(event.delta.x / scale)
    if (!shift) return
    const at = Math.max(0, Math.min(MAX_FRAMES - clip.length, clip.at + shift))
    if (at !== clip.at) void dispatch({ type: 'move', clipId: clip.id, trackId: clip.trackId, at })
  }

  return (
    <section className="timeline-panel" aria-label="Timeline">
      <div className="timeline-toolbar">
        <div className="timeline-actions">
          <Button
            variant="ghost"
            size="sm"
            disabled={!selected}
            onClick={() => {
              if (selected) void dispatch({ type: 'split', clipId: selected, frame: player.current?.getCurrentFrame() ?? 0, rightId: crypto.randomUUID() })
            }}
          >
            <Scissors size={14} /> Split (S)
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={!selected}
            onClick={() => {
              if (selected) void dispatch({ type: 'delete', clipId: selected })
            }}
          >
            <Trash2 size={14} /> Delete
          </Button>
        </div>
        <div className="timeline-zoom">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Zoom out"
            title="Zoom out"
            disabled={zoom <= 0.25}
            onClick={() => setZoom(z => Math.max(0.25, +(z / 1.5).toFixed(2)))}
          >
            <Minus size={14} />
          </Button>
          <span className="technical zoom-level">{Math.round(zoom * 100)}%</span>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Zoom in"
            title="Zoom in"
            disabled={zoom >= 8}
            onClick={() => setZoom(z => Math.min(8, +(z * 1.5).toFixed(2)))}
          >
            <Plus size={14} />
          </Button>
        </div>
      </div>
      <div className="timeline-scroll">
        <div className="timeline-canvas" style={{ width: Math.max(800, frames * scale + 100) }}>
          <Ruler player={player} scale={scale} frames={frames} ticks={ticks} />
          <DndContext sensors={sensors} onDragEnd={dragEnd}>
            <div className="timeline-tracks">
              {project.tracks.map(track => {
                const trackClips = project.clips.filter(c => c.trackId === track.id)
                return (
                  <div key={track.id} className={`track-lane ${track.kind}`}>
                    <div className="track-header">
                      <span>{track.kind === 'video' ? 'Video' : 'Audio'}</span>
                    </div>
                    <div className="track-content">
                      {trackClips.map(clip => {
                        const asset = assets.get(clip.mediaId)
                        if (!asset) return null
                        return (
                          <TimelineClip
                            key={clip.id}
                            clip={clip}
                            asset={asset}
                            source={sources.get(clip.mediaId)}
                            scale={scale}
                          />
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </div>
          </DndContext>
        </div>
      </div>
    </section>
  )
})
