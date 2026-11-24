import { Film, X } from 'lucide-react'
import { useExportStore } from '../state/useExportStore.ts'
import { ExportSettingsForm } from './ExportSettingsForm.tsx'
import { ExportProgress } from './ExportProgress.tsx'
import { ExportCompleted } from './ExportCompleted.tsx'
import { ExportFailed } from './ExportFailed.tsx'

type Props = {
  revision: number
  clipsCount: number
  hasIncompleteAssets: boolean
}

export function ExportModal({
  revision,
  clipsCount,
  hasIncompleteAssets,
}: Props) {
  const isOpen = useExportStore(state => state.isOpen)
  const closeModal = useExportStore(state => state.closeModal)
  const activeJob = useExportStore(state => state.activeJob)

  if (!isOpen) return null

  const renderContent = () => {
    if (!activeJob) {
      return (
        <ExportSettingsForm
          revision={revision}
          clipsCount={clipsCount}
          hasIncompleteAssets={hasIncompleteAssets}
        />
      )
    }

    switch (activeJob.status) {
      case 'queued':
      case 'rendering':
        return <ExportProgress job={activeJob} />
      case 'completed':
        return <ExportCompleted job={activeJob} />
      case 'failed':
      case 'cancelled':
        return <ExportFailed job={activeJob} />
      default:
        return (
          <ExportSettingsForm
            revision={revision}
            clipsCount={clipsCount}
            hasIncompleteAssets={hasIncompleteAssets}
          />
        )
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/65 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-in fade-in duration-150"
      role="dialog"
      aria-modal="true"
    >
      <div className="bg-neutral-900 border border-neutral-700 rounded-xl p-6 max-w-lg w-full shadow-2xl flex flex-col gap-4">
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-neutral-800">
          <div className="flex items-center gap-2.5">
            <div className="p-1.5 bg-purple-500/15 text-purple-400 rounded-lg">
              <Film size={18} />
            </div>
            <h2 className="text-base font-semibold text-neutral-100">Export Video</h2>
            <span className="text-[11px] font-mono text-neutral-400 bg-neutral-800 px-2 py-0.5 rounded-full">
              Rev {revision}
            </span>
          </div>

          <button
            type="button"
            onClick={closeModal}
            className="text-neutral-400 hover:text-neutral-100 p-1 rounded-md hover:bg-neutral-800 transition-colors"
            aria-label="Close export dialog"
          >
            <X size={18} />
          </button>
        </div>

        {/* Dynamic State View */}
        <div className="pt-1">{renderContent()}</div>
      </div>
    </div>
  )
}
