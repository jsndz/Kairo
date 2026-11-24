import { Film, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useExportStore } from '../state/useExportStore.ts'

export function ExportTriggerButton() {
  const openModal = useExportStore(state => state.openModal)
  const activeJob = useExportStore(state => state.activeJob)

  const isRendering = activeJob && (activeJob.status === 'rendering' || activeJob.status === 'queued')
  const progressPercent = activeJob ? Math.round(activeJob.progress) : 0

  return (
    <Button
      variant="outline"
      onClick={openModal}
      className={`relative inline-flex items-center gap-2 border transition-all ${
        isRendering
          ? 'border-purple-500/60 bg-purple-500/10 text-purple-200 hover:bg-purple-500/20'
          : 'border-neutral-700 hover:bg-neutral-800 text-neutral-200'
      }`}
      aria-label="Open video export dialog"
    >
      {isRendering ? (
        <>
          <Loader2 size={15} className="text-purple-400 animate-spin" />
          <span className="font-mono text-xs">
            {activeJob.status === 'queued' ? 'Queued' : `${progressPercent}%`}
          </span>
        </>
      ) : (
        <>
          <Film size={15} className="text-purple-400" />
          <span>Export Video</span>
        </>
      )}
    </Button>
  )
}
