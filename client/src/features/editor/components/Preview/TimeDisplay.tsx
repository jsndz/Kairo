import { timecode } from '../../state/model.ts'
import { usePlaybackFrame, type Playback } from '../../state/playback.ts'

export function TimeDisplay({ player, frames }: { player: Playback; frames: number }) {
  const frame = usePlaybackFrame(player)
  return (
    <div className="time-display">
      <strong>{timecode(frame)}</strong>
      <span> / {timecode(frames)}</span>
    </div>
  )
}
