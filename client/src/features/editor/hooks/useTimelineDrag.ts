import { KeyboardSensor, PointerSensor, useSensor, useSensors, type DragEndEvent } from '@dnd-kit/core'
import type { Clip } from '../state/model.ts'
import { calculateDragTarget } from '../domain/timeline/drag.ts'

export function useTimelineDrag({
  clips,
  scale,
  onMoveClip,
}: {
  clips: Clip[]
  scale: number
  onMoveClip: (clipId: string, trackId: string, at: number) => void
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 3 } }),
    useSensor(KeyboardSensor),
  )

  function handleDragEnd(event: DragEndEvent) {
    const clip = clips.find(item => item.id === event.active.id)
    if (!clip || !event.delta.x) return

    const at = calculateDragTarget(clip.at, event.delta.x, scale, clip.length)
    if (at !== clip.at) {
      onMoveClip(clip.id, clip.trackId, at)
    }
  }

  return {
    sensors,
    handleDragEnd,
  }
}
