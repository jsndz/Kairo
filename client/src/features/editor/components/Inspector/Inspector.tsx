import { memo, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { AudioLines, Film, SlidersHorizontal } from 'lucide-react'
import { FPS, MAX_FRAMES, timecode, trackKind, type Media } from '../../state/model.ts'
import { useEditor } from '../../state/store.ts'
import { importMedia } from '../../media/mediaManager.ts'
import { NumberField } from './NumberField.tsx'

export const Inspector = memo(function Inspector({ media }: { media: Media[] }) {
  const clip = useEditor(state => state.project.clips.find(item => item.id === state.selected))
  const asset = useEditor(state => clip && state.project.assets[clip.mediaId])
  const tracks = useEditor(state => state.project.tracks)
  const dispatch = useEditor(state => state.dispatch)
  const missing = asset && !media.some(item => item.id === asset.id)
  const input = useRef<HTMLInputElement>(null)

  return (
    <aside className="inspector-panel" aria-label="Clip inspector">
      <div className="panel-heading">
        <h2>Inspector</h2>
        <SlidersHorizontal size={16} />
      </div>
      {clip && asset ? (
        <>
          <div className="inspector-asset">
            <span className={'inspector-icon ' + asset.kind}>
              {asset.kind === 'video' ? <Film size={18} /> : <AudioLines size={18} />}
            </span>
            <div className="media-details">
              <strong>{asset.name}</strong>
              <small>
                {asset.kind === 'video' ? 'Video clip' : 'Audio clip'} · {timecode(clip.length)}
              </small>
            </div>
          </div>
          {missing && (
            <div className="inspector-group">
              <p role="status">Media unavailable. Relink the original file to restore playback.</p>
              <Button variant="outline" onClick={() => input.current?.click()}>
                Relink media
              </Button>
              <input
                ref={input}
                type="file"
                hidden
                accept={asset.kind === 'video' ? 'video/*' : asset.kind === 'audio' ? 'audio/*' : 'image/*'}
                onChange={event => {
                  const file = event.currentTarget.files?.[0]
                  if (file) void importMedia([file], asset)
                  event.currentTarget.value = ''
                }}
              />
            </div>
          )}
          <div className="inspector-group" key={clip.id}>
            <h3>Timing</h3>
            <label className="number-field">
              <span>Track</span>
              <select
                aria-label="Track"
                className="number-input"
                value={clip.trackId}
                onChange={event =>
                  void dispatch({ type: 'move', clipId: clip.id, trackId: event.target.value, at: clip.at })
                }
              >
                {tracks
                  .filter(track => track.kind === trackKind(asset.kind))
                  .map((track, i) => (
                    <option key={track.id} value={track.id}>
                      {track.kind === 'video' ? 'Video' : 'Audio'} {i + 1}
                    </option>
                  ))}
              </select>
            </label>
            <NumberField
              label="Timeline start"
              value={clip.at / FPS}
              max={(MAX_FRAMES - clip.length) / FPS}
              step={0.001}
              change={value =>
                void dispatch({
                  type: 'move',
                  clipId: clip.id,
                  trackId: clip.trackId,
                  at: Math.round(value * FPS),
                })
              }
            />
            <NumberField
              label="Source in"
              value={clip.offset / FPS}
              max={(asset.frames - 1) / FPS}
              step={0.001}
              change={value => {
                const offset = Math.round(value * FPS)
                void dispatch({
                  type: 'trim',
                  clipId: clip.id,
                  at: clip.at,
                  offset,
                  length: Math.min(clip.length, asset.frames - offset),
                })
              }}
            />
            <NumberField
              label="Duration"
              value={clip.length / FPS}
              min={1 / FPS}
              max={Math.min(asset.frames - clip.offset, MAX_FRAMES - clip.at) / FPS}
              step={0.001}
              change={value =>
                void dispatch({
                  type: 'trim',
                  clipId: clip.id,
                  at: clip.at,
                  offset: clip.offset,
                  length: Math.round(value * FPS),
                })
              }
            />
          </div>
          <div className="inspector-group" key={clip.id + '-audio'}>
            <h3>Audio</h3>
            <NumberField
              key={clip.id}
              label="Volume"
              value={Math.round(clip.volume * 100)}
              max={100}
              unit="%"
              change={value => void dispatch({ type: 'volume', clipId: clip.id, volume: value / 100 })}
            />
          </div>
        </>
      ) : (
        <p className="media-empty">Select a clip to edit its timing and volume.</p>
      )}
    </aside>
  )
})
