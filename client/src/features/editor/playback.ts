import { useEffect, useState, type RefObject } from 'react'
import type { CallbackListener, PlayerRef } from '@remotion/player'

export type Playback = RefObject<PlayerRef | null>

export function usePlaybackFrame(player: Playback) {
  const [frame, setFrame] = useState(0)
  useEffect(() => {
    const current = player.current
    if (!current) return
    const onFrame: CallbackListener<'frameupdate'> = event => setFrame(event.detail.frame)
    current.addEventListener('frameupdate', onFrame)
    return () => current.removeEventListener('frameupdate', onFrame)
  }, [player])
  return frame
}

