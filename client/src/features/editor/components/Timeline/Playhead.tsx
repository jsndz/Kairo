import { usePlaybackFrame, type Playback } from '../../state/playback.ts'
import { frameToPixel } from '../../domain/timeline/math.ts'

export function Playhead({ player, scale }: { player: Playback; scale: number }) {
  const frame = usePlaybackFrame(player)
  return <div className="playhead" style={{ left: frameToPixel(frame, scale) }} aria-hidden="true" />
}
