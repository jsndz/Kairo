import { memo } from 'react'
import { timecode } from '../../state/model.ts'

export const RulerTicks = memo(function RulerTicks({
  ticks,
  scale,
}: {
  ticks: number[]
  scale: number
}) {
  return ticks.map(tick => (
    <span key={tick} className="tick technical" style={{ left: tick * scale }}>
      {timecode(tick, false)}
    </span>
  ))
})
