import { create } from 'zustand'
import {
  DEFAULT_EXPORT_SETTINGS,
  type AccelerationMode,
  type ExportJob,
  type ExportSettings,
} from '../domain/types.ts'
import { exportApi } from '../api/client.ts'
import { detectVideoAcceleration } from '../engine/hardware.ts'
import { runClientExport } from '../engine/clientExporter.ts'
import type { Clip, Media, Track } from '../../editor/state/model.ts'

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
  accelerationMode: AccelerationMode
  cachedClips: Clip[]
  cachedTracks: Track[]
  cachedMedia: Media[]

  // Actions
  openModal: () => void
  closeModal: () => void
  updateSettings: (partial: Partial<ExportSettings>) => void
  probeHardware: () => Promise<void>
  setTimelineContext: (clips: Clip[], tracks: Track[], media: Media[]) => void
  startExport: (params: {
    revision: number
    clipsCount: number
    hasIncompleteAssets: boolean
    clips?: Clip[]
    tracks?: Track[]
    media?: Media[]
  }) => Promise<boolean>
  cancelExport: () => Promise<void>
  retryExport: () => Promise<void>
  dismissJob: () => void
  checkPreflight: (clipsCount: number, hasIncompleteAssets: boolean) => PreflightCheck
}

let clientAbortController: AbortController | null = null

export const useExportStore = create<ExportState>((set, get) => {
  // Start polling an active server job
  const startPolling = (jobId: string) => {
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
        console.error('Failed to poll export job status:', err)
      }
    }

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
    accelerationMode: 'checking',
    cachedClips: [],
    cachedTracks: [],
    cachedMedia: [],

    openModal: () => {
      set({ isOpen: true, error: null })
      void get().probeHardware()
    },

    closeModal: () => set({ isOpen: false }),

    setTimelineContext: (clips, tracks, media) => {
      set({ cachedClips: clips, cachedTracks: tracks, cachedMedia: media })
    },

    probeHardware: async () => {
      try {
        const res = await detectVideoAcceleration()
        set({ accelerationMode: res.mode })
      } catch {
        set({ accelerationMode: 'unsupported' })
      }
    },

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

    startExport: async ({
      revision,
      clipsCount,
      hasIncompleteAssets,
      clips,
      tracks,
      media,
    }) => {
      const { checkPreflight, settings } = get()
      const preflight = checkPreflight(clipsCount, hasIncompleteAssets)
      if (!preflight.valid) {
        set({ error: preflight.warning })
        return false
      }

      set({ isSubmitting: true, error: null })

      let filename = settings.filename.trim()
      if (!filename) filename = 'export.mp4'
      if (!filename.toLowerCase().endsWith('.mp4')) filename += '.mp4'

      // --- LOCAL IN-BROWSER EXPORT ---
      if (settings.method === 'local') {
        const finalClips = clips || get().cachedClips
        const finalTracks = tracks || get().cachedTracks
        const finalMedia = media || get().cachedMedia

        clientAbortController = new AbortController()

        const localJobId = 'client_' + Math.random().toString(36).substring(2, 10)
        const localJob: ExportJob = {
          id: localJobId,
          requestId: 'req_' + localJobId,
          projectId: '',
          revision,
          status: 'rendering',
          progress: 0,
          attempts: 1,
          outputKey: filename,
          createdAt: new Date().toISOString(),
          startedAt: new Date().toISOString(),
          accelerationMode: get().accelerationMode,
          stageMessage: 'Starting local browser render…',
        }

        set({ activeJob: localJob, isSubmitting: false })

        try {
          const result = await runClientExport({
            settings: { ...settings, filename },
            clips: finalClips,
            tracks: finalTracks,
            media: finalMedia,
            signal: clientAbortController.signal,
            onProgress: progress => {
              const currentJob = get().activeJob
              if (!currentJob || currentJob.id !== localJobId) return
              set({
                activeJob: {
                  ...currentJob,
                  progress: progress.percent,
                  stageMessage:
                    progress.stage === 'audio'
                      ? 'Mixing timeline audio tracks…'
                      : progress.stage === 'rendering'
                        ? `Rendering frame ${progress.frame} of ${progress.totalFrames}…`
                        : progress.stage === 'muxing'
                          ? 'Finalizing MP4 container…'
                          : 'Preparing…',
                },
              })
            },
          })

          set(state => {
            if (!state.activeJob || state.activeJob.id !== localJobId) return state
            return {
              activeJob: {
                ...state.activeJob,
                status: 'completed',
                progress: 100,
                blobUrl: result.blobUrl,
                endedAt: new Date().toISOString(),
                stageMessage: 'Completed',
              },
            }
          })
          return true
        } catch (err: unknown) {
          if (clientAbortController?.signal.aborted) {
            set(state => ({
              activeJob: state.activeJob
                ? { ...state.activeJob, status: 'cancelled', error: 'Export was cancelled.' }
                : null,
            }))
            return false
          }
          const msg = err instanceof Error ? err.message : 'Local browser export failed.'
          set(state => ({
            error: msg,
            activeJob: state.activeJob
              ? { ...state.activeJob, status: 'failed', error: msg }
              : null,
          }))
          return false
        }
      }

      // --- SERVER RENDERING ---
      try {
        const requestId = 'req_' + Math.random().toString(36).substring(2, 12) + '_' + Date.now()
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

      if (job.id.startsWith('client_')) {
        if (clientAbortController) {
          clientAbortController.abort()
          clientAbortController = null
        }
        set({
          activeJob: {
            ...job,
            status: 'cancelled',
            error: 'Export was cancelled by user.',
          },
        })
        return
      }

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

      if (job.id.startsWith('client_')) {
        set({ activeJob: null, error: null })
        return
      }

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
      const current = get().activeJob
      if (current?.blobUrl) {
        URL.revokeObjectURL(current.blobUrl)
      }
      set({ activeJob: null, pollingTimer: null, error: null })
    },
  }
})
