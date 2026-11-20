import { memo, useEffect, useRef } from 'react'
import WaveSurfer from 'wavesurfer.js'

export const Waveform = memo(function Waveform({ src, offset, length, frames }: { src: string; offset: number; length: number; frames: number }) {
  const container = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!container.current) return
    const waveform = WaveSurfer.create({
      container: container.current, height: 30, waveColor: '#78d4ca', cursorWidth: 0,
      interact: false, barWidth: 2, barGap: 1, normalize: true, sampleRate: 8000,
    })
    void waveform.load(src).catch(() => { /* Playback remains available if decoding fails. */ })
    return () => waveform.destroy()
  }, [src])
  return <div className="waveform-window" aria-hidden="true">
    <div ref={container} style={{ width: (frames / length * 100) + '%', left: (-offset / length * 100) + '%' }} />
  </div>
})
