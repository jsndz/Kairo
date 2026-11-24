import { create } from 'zustand'
import {
  DEFAULT_EXPORT_SETTINGS,
  type ExportJob,
  type ExportSettings,
} from '../domain/types.ts'
import { exportApi } from '../api/client.ts'

const SESSION_KEY = 'kairo_active_export_id'

type PreflightCheck = {
  valid: boolean
  warning: string | null
}

type ExportState = {
  isOpen: boolean
  settings: ExportSettings
  activeJob: ExportJob | null
  isSubmitting: boolean
  error: string | null
  pollingTimer: number | null

  // Actions
  openModal: () => void
  closeModal: () => void
  updateSettings: (partial: Partial<ExportSettings>) => void
  startExport: (params: {
    revision: number
    clipsCount: number
    hasIncompleteAssets: boolean
  }) => Promise<boolean>
  cancelExport: () => Promise<void>
  retryExport: () => Promise<void>
  dismissJob: () => void
  checkPreflight: (clipsCount: number, hasIncompleteAssets: boolean) => PreflightCheck
}

export const useExportStore = create<ExportState>((set, get) => {
  // Start polling an active job
  const startPolling = (jobId: string) => {
    // Clear any existing timer
    const currentTimer = get().pollingTimer
    if (currentTimer) window.clearInterval(currentTimer)

    const poll = async () => {
      try {
        const job = await exportApi.getExport(jobId)
        set({ activeJob: job })

        if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
          const t = get().pollingTimer
          if (t) window.clearInterval(t)
          set({ pollingTimer: null })
          if (job.status === 'completed') {
            sessionStorage.removeItem(SESSION_KEY)
          }
        }
      } catch (err) {
        // If error fetching status, don't crash poller immediately
        console.error('Failed to poll export job status:', err)
      }
    }

    // Immediate first poll
    void poll()
    const timer = window.setInterval(poll, 1500)
    set({ pollingTimer: timer })
  }

  // Restore session job on load
  const savedJobId = typeof window !== 'undefined' ? sessionStorage.getItem(SESSION_KEY) : null
  if (savedJobId) {
    void (async () => {
      try {
        const job = await exportApi.getExport(savedJobId)
        set({ activeJob: job })
        if (job.status === 'queued' || job.status === 'rendering') {
          startPolling(savedJobId)
        } else {
          sessionStorage.removeItem(SESSION_KEY)
        }
      } catch {
        sessionStorage.removeItem(SESSION_KEY)
      }
    })()
  }

  return {
    isOpen: false,
    settings: { ...DEFAULT_EXPORT_SETTINGS },
    activeJob: null,
    isSubmitting: false,
    error: null,
    pollingTimer: null,

    openModal: () => set({ isOpen: true, error: null }),
    closeModal: () => set({ isOpen: false }),

    updateSettings: partial =>
      set(state => ({
        settings: { ...state.settings, ...partial },
      })),

    checkPreflight: (clipsCount: number, hasIncompleteAssets: boolean): PreflightCheck => {
      if (clipsCount === 0) {
        return {
          valid: false,
          warning: 'Timeline is empty. Add at least one clip to export.',
        }
      }
      if (hasIncompleteAssets) {
        return {
          valid: false,
          warning: 'Some media files are still uploading. Please wait for uploads to complete.',
        }
      }
      return { valid: true, warning: null }
    },

    startExport: async ({ revision, clipsCount, hasIncompleteAssets }) => {
      const { checkPreflight, settings } = get()
      const preflight = checkPreflight(clipsCount, hasIncompleteAssets)
      if (!preflight.valid) {
        set({ error: preflight.warning })
        return false
      }

      set({ isSubmitting: true, error: null })

      try {
        const requestId = 'req_' + Math.random().toString(36).substring(2, 12) + '_' + Date.now()
        let filename = settings.filename.trim()
        if (!filename) filename = 'export.mp4'
        if (!filename.toLowerCase().endsWith('.mp4')) filename += '.mp4'

        const job = await exportApi.createExport({
          requestId,
          revision,
          settings: { ...settings, filename },
        })

        sessionStorage.setItem(SESSION_KEY, job.id)
        set({ activeJob: job, isSubmitting: false })
        startPolling(job.id)
        return true
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : 'Export submission failed.'
        set({ error: msg, isSubmitting: false })
        return false
      }
    },

    cancelExport: async () => {
      const job = get().activeJob
      if (!job) return

      try {
        const updated = await exportApi.cancelExport(job.id)
        const t = get().pollingTimer
        if (t) window.clearInterval(t)
        sessionStorage.removeItem(SESSION_KEY)
        set({ activeJob: updated, pollingTimer: null })
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : 'Failed to cancel export.'
        set({ error: msg })
      }
    },

    retryExport: async () => {
      const job = get().activeJob
      if (!job) return

      set({ isSubmitting: true, error: null })
      try {
        const updated = await exportApi.retryExport(job.id)
        sessionStorage.setItem(SESSION_KEY, updated.id)
        set({ activeJob: updated, isSubmitting: false })
        startPolling(updated.id)
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : 'Failed to retry export.'
        set({ error: msg, isSubmitting: false })
      }
    },

    dismissJob: () => {
      const t = get().pollingTimer
      if (t) window.clearInterval(t)
      sessionStorage.removeItem(SESSION_KEY)
      set({ activeJob: null, pollingTimer: null, error: null })
    },
  }
})
