import { useEffect, useRef, type KeyboardEvent, type PointerEvent } from 'react'
import type { Clip } from '../state/model.ts'
import { calculateKeyboardTrimDelta, calculateTrimPreview, type TrimEdge } from '../domain/timeline/trim.ts'
import { edgeTrim } from '../state/operations.ts'

export function useTrimHandle({
  clip,
  assetFrames,
  edge,
  scale,
  onPreview,
  onCommit,
}: {
  clip: Clip
  assetFrames: number
  edge: TrimEdge
  scale: number
  onPreview: (clip: Clip | null) => void
  onCommit: (trimUpdate: { at: number; offset: number; length: number }) => void
}) {
  const origin = useRef<number | null>(null)
  const request = useRef<number | null>(null)

  function cancelPreview() {
    if (request.current !== null) cancelAnimationFrame(request.current)
    request.current = null
    origin.current = null
    onPreview(null)
  }

  useEffect(() => {
    return () => {
      if (request.current !== null) cancelAnimationFrame(request.current)
    }
  }, [])

  function handlePointerDown(event: PointerEvent<HTMLButtonElement>) {
    if (event.button !== 0) return
    origin.current = event.clientX
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  function handlePointerMove(event: PointerEvent<HTMLButtonElement>) {
    if (origin.current === null) return
    if (request.current !== null) cancelAnimationFrame(request.current)
    const delta = Math.round((event.clientX - origin.current) / scale)
    request.current = requestAnimationFrame(() => {
      request.current = null
      onPreview(calculateTrimPreview(clip, assetFrames, edge, delta))
    })
  }

  function handlePointerUp(event: PointerEvent<HTMLButtonElement>) {
    if (origin.current !== null) {
      const delta = Math.round((event.clientX - origin.current) / scale)
      onCommit(edgeTrim(clip, assetFrames, edge, delta))
    }
    cancelPreview()
  }

  function handleKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
    event.preventDefault()
    const delta = calculateKeyboardTrimDelta(event.key, event.shiftKey)
    onCommit(edgeTrim(clip, assetFrames, edge, delta))
  }

  return {
    onPointerDown: handlePointerDown,
    onPointerMove: handlePointerMove,
    onPointerUp: handlePointerUp,
    onLostPointerCapture: cancelPreview,
    onKeyDown: handleKeyDown,
  }
}
