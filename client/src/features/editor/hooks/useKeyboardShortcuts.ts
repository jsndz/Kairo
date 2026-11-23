import { useEffect, type RefObject } from 'react'
import type { PlayerRef } from '@remotion/player'
import { useEditor } from '../state/store.ts'

export function useKeyboardShortcuts(player: RefObject<PlayerRef | null>, enabled: boolean = true) {
  useEffect(() => {
    if (!enabled) return

    function handleKeyDown(event: KeyboardEvent) {
      const target = event.target
      if (
        event.defaultPrevented ||
        (target instanceof HTMLElement && target.closest('input, textarea, select, [contenteditable=true]'))
      ) {
        return
      }

      const editor = useEditor.getState()

      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'z') {
        event.preventDefault()
        if (event.shiftKey) {
          editor.redo()
        } else {
          editor.undo()
        }
      } else if (!event.metaKey && !event.ctrlKey && !event.altKey) {
        if (event.key === ' ' && !(target instanceof HTMLElement && target.closest('button'))) {
          event.preventDefault()
          if (editor.project.clips.length) {
            player.current?.toggle()
          }
        } else if (event.key.toLowerCase() === 's') {
          editor.split(player.current?.getCurrentFrame() ?? 0)
        } else if (event.key === 'Delete' || event.key === 'Backspace') {
          event.preventDefault()
          editor.remove()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [player, enabled])
}
