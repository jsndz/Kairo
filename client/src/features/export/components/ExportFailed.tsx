import { AlertTriangle, RefreshCw, Settings2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useExportStore } from '../state/useExportStore.ts'
import type { ExportJob } from '../domain/types.ts'

type Props = {
  job: ExportJob
}

export function ExportFailed({ job }: Props) {
  const retryExport = useExportStore(state => state.retryExport)
  const dismissJob = useExportStore(state => state.dismissJob)
  const isSubmitting = useExportStore(state => state.isSubmitting)

  return (
    <div className="flex flex-col items-center text-center py-4 gap-6">
      {/* Alert Icon */}
      <div className="flex items-center justify-center w-16 h-16 rounded-full bg-rose-500/15 border border-rose-500/30 text-rose-400">
        <AlertTriangle size={36} />
      </div>

      {/* Title & Reason */}
      <div className="flex flex-col gap-1.5 max-w-sm">
        <h3 className="text-base font-semibold text-neutral-100">Export Failed</h3>
        <p className="text-xs text-neutral-400 leading-relaxed">
          The render was interrupted or could not be completed.
        </p>
        {job.error && (
          <div className="mt-2 p-2.5 bg-neutral-950 border border-neutral-800 rounded-lg text-xs font-mono text-rose-300 text-left overflow-x-auto max-h-24">
            {job.error}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex flex-col gap-2.5 w-full max-w-md pt-2">
        <Button
          onClick={() => void retryExport()}
          disabled={isSubmitting}
          className="w-full bg-purple-600 hover:bg-purple-500 text-white font-medium py-2 rounded-lg transition-all text-sm"
        >
          <RefreshCw size={15} className={`mr-1.5 ${isSubmitting ? 'animate-spin' : ''}`} />
          {isSubmitting ? 'Retrying…' : 'Retry Export'}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={dismissJob}
          className="border-neutral-700 text-neutral-300 hover:bg-neutral-800"
        >
          <Settings2 size={14} className="mr-1.5" /> Adjust Export Settings
        </Button>
      </div>
    </div>
  )
}
