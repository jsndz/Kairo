import { AudioLines, Film, SlidersHorizontal } from 'lucide-react'
import { FPS, timecode, type Media } from './model'
import { useEditor } from './store'

function NumberField({ label, value, change, max, min = 0, step = 1, unit = 's' }: {
  label: string; value: number; change: (value: number) => void; max: number; min?: number; step?: number; unit?: string
}) {
  return <label className="number-field">
    <span>{label}</span>
    <span className="number-input">
      <input key={value} type="number" min={min} max={max} step={step} defaultValue={+value.toFixed(3)}
        onBlur={event => {
          const next = event.currentTarget.valueAsNumber
          if (Number.isFinite(next)) change(Math.max(min, Math.min(max, next)))
          event.currentTarget.value = String(+value.toFixed(3))
        }}
        onKeyDown={event => { if (event.key === 'Enter') event.currentTarget.blur() }} />
      <small>{unit}</small>
    </span>
  </label>
}

export function Inspector({ media }: { media: Media[] }) {
  const clip = useEditor(state => state.project.clips.find(item => item.id === state.selected))
  const asset = media.find(item => item.id === clip?.mediaId)
  const update = useEditor(state => state.update)

  return <aside className="inspector-panel" aria-label="Clip inspector">
    <div className="panel-heading"><h2>Inspector</h2><SlidersHorizontal size={16} /></div>
    {clip && asset ? <>
      <div className="inspector-asset">
        <span className={'inspector-icon ' + asset.kind}>{asset.kind === 'video' ? <Film size={18} /> : <AudioLines size={18} />}</span>
        <div className="media-details"><strong>{asset.name}</strong><small>{asset.kind === 'video' ? 'Video clip' : 'Audio clip'} · {timecode(clip.length)}</small></div>
      </div>
      <div className="inspector-group" key={clip.id}>
        <h3>Timing</h3>
        <NumberField label="Timeline start" value={clip.at / FPS} max={3599.967} step={0.001} change={value => update(clip.id, { at: value * FPS }, asset)} />
        <NumberField label="Source in" value={clip.offset / FPS} max={(asset.frames - 1) / FPS} step={0.001} change={value => update(clip.id, { offset: value * FPS }, asset)} />
        <NumberField label="Duration" value={clip.length / FPS} min={1 / FPS} max={(asset.frames - clip.offset) / FPS} step={0.001} change={value => update(clip.id, { length: value * FPS }, asset)} />
      </div>
      <div className="inspector-group">
        <h3>Audio</h3>
        <NumberField key={clip.id} label="Volume" value={Math.round(clip.volume * 100)} max={100} unit="%" change={value => update(clip.id, { volume: value / 100 }, asset)} />
      </div>
    </> : <p className="media-empty">Select a clip to edit its timing and volume.</p>}
  </aside>
}
