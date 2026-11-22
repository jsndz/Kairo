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
import { projectSocket } from '../api'
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
  const peerSelected = useEditor(state => state.peerPresence?.selected === clip.id)
  const select = useEditor(state => state.select)
  const { attributes, listeners, setNodeRef, transform } = useDraggable({ id: clip.id })
  const [preview, setPreview] = useState<Clip | null>(null)
  const shown = preview ?? clip
  return (
    <div
      ref={setNodeRef}
      className={`timeline-clip ${asset.kind} ${selected ? 'selected' : ''} ${peerSelected ? 'peer-selected' : ''} ${source ? '' : 'missing'}`}
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
          if (event.button !== 0) return
          event.preventDefault()
          event.currentTarget.focus()
          event.currentTarget.setPointerCapture(event.pointerId)
          scrub(event)
        }}
        onPointerMove={event => {
          if (event.currentTarget.hasPointerCapture(event.pointerId)) scrub(event)
        }}
      />
    </div>
  )
}

function Playhead({ player, scale }: { player: Playback; scale: number }) {
  const frame = usePlaybackFrame(player)
  return <div className="playhead" style={{ left: frame * scale }} aria-hidden="true" />
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
  const split = useEditor(state => state.split)
  const remove = useEditor(state => state.remove)
  const selected = useEditor(state => state.selected)
  const select = useEditor(state => state.select)
  const peerPresence = useEditor(state => state.peerPresence)
  const currentFrame = usePlaybackFrame(player)

  useEffect(() => {
    projectSocket.sendPresence({ playhead: currentFrame, selected })
  }, [currentFrame, selected])

  const [zoom, setZoom] = useState(100)
  const scale = (45 / FPS) * (zoom / 100)
  const extent = Math.max(20 * FPS, duration(project.clips) + FPS * 2)
  const tickStep = FPS * Math.max(1, Math.ceil(66 / (scale * FPS)), Math.ceil(extent / FPS / 240))
  const ticks = useMemo(
    () => Array.from({ length: Math.floor(extent / tickStep) + 1 }, (_, i) => i * tickStep),
    [extent, tickStep],
  )

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 3 } }),
    useSensor(KeyboardSensor),
  )

  const assets = useMemo(() => new Map(Object.values(project.assets).map(a => [a.id, a])), [project.assets])
  const sources = useMemo(() => new Map(media.map(m => [m.id, m])), [media])

  function dragEnd(event: DragEndEvent) {
    const clip = project.clips.find(item => item.id === event.active.id)
    if (!clip || !event.delta.x) return
    const at = Math.max(0, Math.min(MAX_FRAMES - clip.length, Math.round(clip.at + event.delta.x / scale)))
    if (at !== clip.at) void dispatch({ type: 'move', clipId: clip.id, trackId: clip.trackId, at })
  }

  return (
    <section className="timeline-panel" aria-label="Timeline">
      <div className="panel-heading timeline-heading">
        <div className="timeline-tools">
          <h2>Timeline</h2>
          <Button
            variant="ghost"
            size="icon"
            title="Add video track"
            aria-label="Add video track"
            onClick={() => void dispatch({ type: 'add-track', trackId: crypto.randomUUID() })}
          >
            <Plus />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Split at playhead (S)"
            aria-label="Split at playhead"
            disabled={!selected}
            onClick={() => {
              if (selected) split(player.current?.getCurrentFrame() ?? 0)
            }}
          >
            <Scissors />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Delete clip"
            aria-label="Delete clip"
            disabled={!selected}
            onClick={() => {
              if (selected) remove()
            }}
          >
            <Trash2 />
          </Button>
        </div>
        <div className="zoom-controls">
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Zoom out"
            disabled={zoom <= 25}
            onClick={() => setZoom(z => Math.max(25, z - 25))}
          >
            <Minus />
          </Button>
          <input
            aria-label="Timeline zoom"
            type="range"
            min={25}
            max={250}
            step={25}
            value={zoom}
            onChange={event => setZoom(+event.target.value)}
          />
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Zoom in"
            disabled={zoom >= 250}
            onClick={() => setZoom(z => Math.min(250, z + 25))}
          >
            <Plus />
          </Button>
          <span className="technical zoom-value">{zoom}%</span>
        </div>
      </div>
      <div className="timeline-grid">
        <div className="track-labels">
          <div className="ruler-label technical">TRACKS</div>
          {project.tracks.map((track, i) => (
            <div key={track.id}>
              {track.kind === 'video' ? <Film size={15} /> : <AudioLines size={15} />}
              <span>{track.kind === 'video' ? `Video ${i + 1}` : `Audio ${i + 1}`}</span>
            </div>
          ))}
        </div>
        <div className="timeline-scroll">
          <div
            className="timeline-content"
            style={{
              width: Math.max(800, extent * scale),
              minWidth: '100%',
              '--tick': `${tickStep * scale}px`,
            } as React.CSSProperties}
            onPointerMove={event => {
              const rect = event.currentTarget.getBoundingClientRect()
              const x = Math.round(event.clientX - rect.left)
              const y = Math.round(event.clientY - rect.top)
              projectSocket.sendPresence({ cursor: { x, y } })
            }}
            onPointerLeave={() => {
              projectSocket.sendPresence({ cursor: null })
            }}
          >
            <Ruler
              player={player}
              scale={scale}
              frames={duration(project.clips)}
              ticks={ticks}
            />
            <DndContext
              sensors={sensors}
              onDragStart={event => select(String(event.active.id))}
              onDragEnd={dragEnd}
            >
              {project.tracks.map(track => {
                const trackClips = project.clips.filter(c => c.trackId === track.id)
                return (
                  <div key={track.id} className="track-lane" aria-label={track.kind}>
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
                )
              })}
            </DndContext>
            <Playhead player={player} scale={scale} />
            {peerPresence?.playhead !== undefined && (
              <div
                className="peer-playhead"
                style={{ left: Math.min(peerPresence.playhead, duration(project.clips) - 1) * scale }}
                title="Peer playhead"
              />
            )}
            {peerPresence?.cursor && (
              <div
                className="peer-cursor"
                style={{
                  transform: `translate(${peerPresence.cursor.x}px, ${peerPresence.cursor.y}px)`,
                }}
                aria-hidden="true"
              >
                <svg
                  width="18"
                  height="18"
                  viewBox="0 0 24 24"
                  fill="none"
                  className="peer-cursor-pointer"
                >
                  <path
                    d="M5.65376 12.3673H5.46026L5.31717 12.4976L0.500002 16.8829L0.500002 1.19841L11.7841 12.3673H5.65376Z"
                    fill="#10b981"
                    stroke="#047857"
                    strokeWidth="1.2"
                  />
                </svg>
                <span className="peer-cursor-tag">Peer</span>
              </div>
            )}
          </div>
        </div>
      </div>
      <div className="timeline-footer">
        <span>Drag clips to move · Select clip to trim or split</span>
        <span className="technical">
          {project.clips.length} {project.clips.length === 1 ? 'CLIP' : 'CLIPS'}
        </span>
      </div>
    </section>
  )
})
