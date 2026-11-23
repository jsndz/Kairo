import { memo, useMemo, type CSSProperties } from 'react'
import { DndContext } from '@dnd-kit/core'
import { AudioLines, Film, Minus, Plus, Scissors, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { duration, type Media } from '../../state/model.ts'
import { useEditor } from '../../state/store.ts'
import { projectSocket } from '../../api/ws.ts'
import { usePlaybackFrame, type Playback } from '../../state/playback.ts'
import { useTimelineScale } from '../../hooks/useTimelineScale.ts'
import { useTimelineDrag } from '../../hooks/useTimelineDrag.ts'
import { useTimelinePresence } from '../../hooks/useTimelinePresence.ts'
import { calculateTickStep } from '../../domain/timeline/math.ts'
import { Ruler } from './Ruler.tsx'
import { Playhead } from './Playhead.tsx'
import { TimelineClip } from './TimelineClip.tsx'

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
  const proposeConsensus = useEditor(state => state.proposeConsensus)
  const currentFrame = usePlaybackFrame(player)

  useTimelinePresence(currentFrame, selected)

  const { zoom, setZoom, zoomIn, zoomOut, scale, extent, ticks } = useTimelineScale(project.clips)
  const tickStep = useMemo(() => calculateTickStep(extent, scale), [extent, scale])

  const { sensors, handleDragEnd } = useTimelineDrag({
    clips: project.clips,
    scale,
    onMoveClip: (clipId, trackId, at) => {
      void dispatch({ type: 'move', clipId, trackId, at })
    },
  })

  const assets = useMemo(() => new Map(Object.values(project.assets).map(a => [a.id, a])), [project.assets])
  const sources = useMemo(() => new Map(media.map(m => [m.id, m])), [media])

  const videoTracksCount = project.tracks.filter(t => t.kind === 'video').length
  const audioTracksCount = project.tracks.filter(t => t.kind === 'audio').length

  let videoIndex = 0
  let audioIndex = 0

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
            onClick={() => {
              const op = { type: 'add-track' as const, trackId: crypto.randomUUID(), kind: 'video' as const }
              if (peerPresence) {
                void proposeConsensus(op, 'Add Video Track')
              } else {
                void dispatch(op)
              }
            }}
          >
            <Plus />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Add audio track"
            aria-label="Add audio track"
            onClick={() => {
              const op = { type: 'add-track' as const, trackId: crypto.randomUUID(), kind: 'audio' as const }
              if (peerPresence) {
                void proposeConsensus(op, 'Add Audio Track')
              } else {
                void dispatch(op)
              }
            }}
          >
            <AudioLines size={15} />
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
            onClick={zoomOut}
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
            onClick={zoomIn}
          >
            <Plus />
          </Button>
          <span className="technical zoom-value">{zoom}%</span>
        </div>
      </div>

      <div className="timeline-grid">
        <div className="track-labels">
          <div className="ruler-label technical">TRACKS</div>
          {project.tracks.map(track => {
            const count = track.kind === 'video' ? ++videoIndex : ++audioIndex
            const canDelete = track.kind === 'video' ? videoTracksCount > 1 : audioTracksCount > 1

            return (
              <div key={track.id} className="track-label-row">
                <div className="track-label-info">
                  {track.kind === 'video' ? <Film size={15} /> : <AudioLines size={15} />}
                  <span>{track.kind === 'video' ? `Video ${count}` : `Audio ${count}`}</span>
                </div>
                {canDelete && (
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="delete-track-btn"
                    title={`Delete ${track.kind} track`}
                    aria-label={`Delete ${track.kind} track`}
                    onClick={() => {
                      const op = { type: 'delete-track' as const, trackId: track.id }
                      if (peerPresence) {
                        void proposeConsensus(op, `Delete ${track.kind === 'video' ? 'Video' : 'Audio'} Track`)
                      } else {
                        void dispatch(op)
                      }
                    }}
                  >
                    <Trash2 size={13} />
                  </Button>
                )}
              </div>
            )
          })}
        </div>

        <div className="timeline-scroll">
          <div
            className="timeline-content"
            style={{
              width: Math.max(800, extent * scale),
              minWidth: '100%',
              '--tick': `${tickStep * scale}px`,
            } as CSSProperties}
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
              onDragEnd={handleDragEnd}
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
