import { useEffect } from 'react'
import { projectSocket } from '../api/ws.ts'

export function useTimelinePresence(currentFrame: number, selected: string | null) {
  useEffect(() => {
    projectSocket.sendPresence({ playhead: currentFrame, selected })
  }, [currentFrame, selected])
}
