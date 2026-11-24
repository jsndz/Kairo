import { CheckCircle2, Download, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useExportStore } from '../state/useExportStore.ts'
import { exportApi } from '../api/client.ts'
import type { ExportJob } from '../domain/types.ts'

type Props = {
  job: ExportJob
}

export function ExportCompleted({ job }: Props) {
  const dismissJob = useExportStore(state => state.dismissJob)
  const downloadURL = exportApi.getDownloadURL(job.id)

  return (
    <div className="flex flex-col items-center text-center py-4 gap-6">
      {/* Success Badge */}
      <div className="flex items-center justify-center w-16 h-16 rounded-full bg-emerald-500/15 border border-emerald-500/30 text-emerald-400">
        <CheckCircle2 size={36} />
      </div>

      {/* Title & Description */}
      <div className="flex flex-col gap-1.5 max-w-sm">
        <h3 className="text-base font-semibold text-neutral-100">Export Complete!</h3>
        <p className="text-xs text-neutral-400 leading-relaxed">
          Your video has been rendered and verified. It is ready for download.
        </p>
      </div>

      {/* Download Action */}
      <div className="flex flex-col gap-3 w-full max-w-md pt-2">
        <a
          href={downloadURL}
          download
          className="w-full inline-flex items-center justify-center gap-2 bg-emerald-600 hover:bg-emerald-500 text-white font-medium py-2.5 px-4 rounded-lg shadow-md transition-all text-sm"
        >
          <Download size={16} /> Download Video (.mp4)
        </a>

        <Button
          variant="ghost"
          size="sm"
          onClick={dismissJob}
          className="text-neutral-400 hover:text-neutral-100 hover:bg-neutral-800"
        >
          <RotateCcw size={14} className="mr-1.5" /> Export Another Video
        </Button>
      </div>
    </div>
  )
}
