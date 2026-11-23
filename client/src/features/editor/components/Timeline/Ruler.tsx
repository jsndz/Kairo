import { timecode } from '../../state/model.ts'
import type { Playback } from '../../state/playback.ts'
import { useRulerScrub } from '../../hooks/useRulerScrub.ts'
import { RulerTicks } from './RulerTicks.tsx'

export function Ruler({
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
  const { frame, seek, handlePointerDown, handlePointerMove } = useRulerScrub({
    player,
    scale,
    frames,
  })

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
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
      />
    </div>
  )
}
