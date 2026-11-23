import { type PointerEvent } from 'react'
import { clampFrame, pixelToFrame } from '../domain/timeline/math.ts'
import { usePlaybackFrame, type Playback } from '../state/playback.ts'

export function useRulerScrub({
  player,
  scale,
  frames,
}: {
  player: Playback
  scale: number
  frames: number
}) {
  const frame = usePlaybackFrame(player)

  function seek(value: number) {
    const target = clampFrame(value, frames - 1)
    player.current?.seekTo(target)
  }

  function scrub(event: PointerEvent<HTMLInputElement>) {
    const rect = event.currentTarget.getBoundingClientRect()
    const offsetX = event.clientX - rect.left
    seek(pixelToFrame(offsetX, scale))
  }

  function handlePointerDown(event: PointerEvent<HTMLInputElement>) {
    if (event.button !== 0) return
    event.preventDefault()
    event.currentTarget.focus()
    event.currentTarget.setPointerCapture(event.pointerId)
    scrub(event)
  }

  function handlePointerMove(event: PointerEvent<HTMLInputElement>) {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      scrub(event)
    }
  }

  return {
    frame,
    seek,
    handlePointerDown,
    handlePointerMove,
  }
}
