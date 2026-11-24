import { Loader2, X, Clock, Info } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useExportStore } from '../state/useExportStore.ts'
import type { ExportJob } from '../domain/types.ts'

type Props = {
  job: ExportJob
}

export function ExportProgress({ job }: Props) {
  const cancelExport = useExportStore(state => state.cancelExport)
  const isQueued = job.status === 'queued'
  const progressPercent = Math.min(100, Math.max(0, Math.round(job.progress)))

  return (
    <div className="flex flex-col items-center text-center py-4 gap-6">
      {/* Icon / Loader */}
      <div className="relative flex items-center justify-center w-16 h-16 rounded-full bg-purple-500/15 border border-purple-500/30">
        {isQueued ? (
          <Clock size={32} className="text-purple-400 animate-pulse" />
        ) : (
          <Loader2 size={32} className="text-purple-400 animate-spin" />
        )}
      </div>

      {/* Status Information */}
      <div className="flex flex-col gap-1.5 max-w-sm">
        <h3 className="text-base font-semibold text-neutral-100">
          {isQueued ? 'Waiting in Server Queue…' : 'Rendering Video…'}
        </h3>
        <p className="text-xs text-neutral-400 leading-relaxed">
          {isQueued
            ? 'Your job is registered and will begin processing sequentially.'
            : 'Compositing video tracks, mixing audio, and encoding H.264 / AAC MP4.'}
        </p>
      </div>

      {/* Progress Bar */}
      <div className="w-full max-w-md flex flex-col gap-2">
        <div className="w-full h-2 bg-neutral-800 rounded-full overflow-hidden border border-neutral-700">
          <div
            className="h-full bg-linear-to-r from-purple-500 to-indigo-400 transition-all duration-300 ease-out"
            style={{ width: `${isQueued ? 5 : Math.max(5, progressPercent)}%` }}
          />
        </div>
        <div className="flex justify-between text-xs text-neutral-500 font-mono">
          <span>{isQueued ? 'Queued' : 'Rendering'}</span>
          <span>{isQueued ? '0%' : `${progressPercent}%`}</span>
        </div>
      </div>

      {/* Background execution tip */}
      <div className="flex items-center gap-2 p-3 bg-neutral-950 border border-neutral-800 rounded-lg text-xs text-neutral-400 max-w-md text-left">
        <Info size={16} className="text-sky-400 shrink-0" />
        <span>
          You can safely close this dialog. Rendering continues on the server and you can monitor
          progress from the header.
        </span>
      </div>

      {/* Cancel Action */}
      <div className="pt-2 w-full max-w-md">
        <Button
          variant="outline"
          size="sm"
          onClick={() => void cancelExport()}
          className="w-full border-neutral-700 text-neutral-300 hover:bg-neutral-800"
        >
          <X size={14} className="mr-1.5" /> Cancel Export
        </Button>
      </div>
    </div>
  )
}
