import { useMemo, useState } from 'react'
import type { Clip } from '../state/model.ts'
import { calculateExtent, calculateScale, generateRulerTicks } from '../domain/timeline/math.ts'

export function useTimelineScale(clips: Clip[], initialZoom: number = 100) {
  const [zoom, setZoom] = useState(initialZoom)

  const scale = useMemo(() => calculateScale(zoom), [zoom])
  const extent = useMemo(() => calculateExtent(clips), [clips])
  const ticks = useMemo(() => generateRulerTicks(extent, scale), [extent, scale])

  const zoomIn = () => setZoom(z => Math.min(250, z + 25))
  const zoomOut = () => setZoom(z => Math.max(25, z - 25))

  return {
    zoom,
    setZoom,
    zoomIn,
    zoomOut,
    scale,
    extent,
    ticks,
  }
}
