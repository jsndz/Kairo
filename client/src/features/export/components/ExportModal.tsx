import { Film, X } from 'lucide-react'
import { useExportStore } from '../state/useExportStore.ts'
import { ExportSettingsForm } from './ExportSettingsForm.tsx'
import { ExportProgress } from './ExportProgress.tsx'
import { ExportCompleted } from './ExportCompleted.tsx'
import { ExportFailed } from './ExportFailed.tsx'
import type { Clip, Media, Track } from '../../editor/state/model.ts'

type Props = {
  revision: number
  clipsCount: number
  hasIncompleteAssets: boolean
  projectId?: string
  clips?: Clip[]
  tracks?: Track[]
  media?: Media[]
}

export function ExportModal({
  revision,
  clipsCount,
  hasIncompleteAssets,
  projectId,
  clips,
  tracks,
  media,
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
          projectId={projectId}
          clips={clips}
          tracks={tracks}
          media={media}
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
            clips={clips}
            tracks={tracks}
            media={media}
          />
        )
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs p-4 animate-in fade-in duration-200"
      role="dialog"
      aria-modal="true"
    >
      <div className="relative w-full max-w-lg bg-neutral-900 border border-neutral-800 rounded-xl shadow-2xl overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-neutral-800 bg-neutral-900/50">
          <div className="flex items-center gap-2.5">
            <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-purple-500/10 border border-purple-500/20 text-purple-400">
              <Film size={18} />
            </div>
            <h2 className="text-sm font-semibold text-neutral-100">Export Video</h2>
          </div>
          <button
            type="button"
            onClick={closeModal}
            className="text-neutral-400 hover:text-neutral-200 p-1 rounded-md hover:bg-neutral-800 transition-colors"
            aria-label="Close dialog"
          >
            <X size={16} />
          </button>
        </div>

        {/* Content Body */}
        <div className="p-6">{renderContent()}</div>
      </div>
    </div>
  )
}
