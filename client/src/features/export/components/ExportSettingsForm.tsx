import { useState } from 'react'
import { AlertCircle, Film, Sparkles, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useExportStore } from '../state/useExportStore.ts'
import type { FrameRate, Quality, Resolution } from '../domain/types.ts'

type Props = {
  revision: number
  clipsCount: number
  hasIncompleteAssets: boolean
}

export function ExportSettingsForm({
  revision,
  clipsCount,
  hasIncompleteAssets,
}: Props) {
  const settings = useExportStore(state => state.settings)
  const updateSettings = useExportStore(state => state.updateSettings)
  const startExport = useExportStore(state => state.startExport)
  const isSubmitting = useExportStore(state => state.isSubmitting)
  const checkPreflight = useExportStore(state => state.checkPreflight)
  const error = useExportStore(state => state.error)

  const preflight = checkPreflight(clipsCount, hasIncompleteAssets)
  const [filenameInput, setFilenameInput] = useState(settings.filename)

  const handleFilenameBlur = () => {
    let clean = filenameInput.trim()
    if (!clean) clean = 'export.mp4'
    if (!clean.toLowerCase().endsWith('.mp4')) clean += '.mp4'
    setFilenameInput(clean)
    updateSettings({ filename: clean })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    handleFilenameBlur()
    void startExport({
      revision,
      clipsCount,
      hasIncompleteAssets,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-5">
      {/* Preflight Warning or Error */}
      {(!preflight.valid || error) && (
        <div className="flex items-start gap-3 p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg text-amber-300 text-xs">
          <AlertCircle size={16} className="shrink-0 mt-0.5 text-amber-400" />
          <span>{error || preflight.warning}</span>
        </div>
      )}

      {/* Filename Input */}
      <div className="flex flex-col gap-1.5">
        <label htmlFor="export-filename" className="text-xs font-medium text-neutral-300">
          File Name
        </label>
        <div className="flex items-center rounded-lg border border-neutral-700 bg-neutral-950 px-3 py-1.5 focus-within:border-purple-500">
          <input
            id="export-filename"
            type="text"
            className="w-full bg-transparent text-sm text-neutral-100 outline-none"
            value={filenameInput}
            onChange={e => setFilenameInput(e.target.value)}
            onBlur={handleFilenameBlur}
            placeholder="my_video.mp4"
          />
        </div>
      </div>

      {/* Resolution Selector */}
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-medium text-neutral-300">Resolution</label>
        <div className="grid grid-cols-3 gap-2">
          {(
            [
              { id: '720p', label: '720p', desc: '1280 × 720' },
              { id: '1080p', label: '1080p (FHD)', desc: '1920 × 1080' },
              { id: '4k', label: '4K (UHD)', desc: '3840 × 2160' },
            ] as const
          ).map(res => (
            <button
              type="button"
              key={res.id}
              onClick={() => updateSettings({ resolution: res.id as Resolution })}
              className={`flex flex-col items-center justify-center p-2.5 rounded-lg border text-xs transition-colors ${
                settings.resolution === res.id
                  ? 'border-purple-500 bg-purple-500/15 text-purple-200 font-semibold shadow-xs'
                  : 'border-neutral-800 bg-neutral-950/60 text-neutral-400 hover:border-neutral-700 hover:text-neutral-200'
              }`}
            >
              <span>{res.label}</span>
              <span className="text-[10px] text-neutral-500 mt-0.5">{res.desc}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Frame Rate & Quality */}
      <div className="grid grid-cols-2 gap-4">
        {/* Frame Rate */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-neutral-300">Frame Rate</label>
          <div className="flex gap-1.5 bg-neutral-950 p-1 rounded-lg border border-neutral-800">
            {([24, 30, 60] as const).map(fps => (
              <button
                type="button"
                key={fps}
                onClick={() => updateSettings({ frameRate: fps as FrameRate })}
                className={`flex-1 py-1 text-xs rounded-md transition-colors ${
                  settings.frameRate === fps
                    ? 'bg-neutral-800 text-neutral-100 font-semibold'
                    : 'text-neutral-400 hover:text-neutral-200'
                }`}
              >
                {fps} fps
              </button>
            ))}
          </div>
        </div>

        {/* Quality */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-neutral-300">Quality</label>
          <div className="flex gap-1.5 bg-neutral-950 p-1 rounded-lg border border-neutral-800">
            {(
              [
                { id: 'standard', label: 'Standard' },
                { id: 'high', label: 'High (CRF 18)' },
              ] as const
            ).map(q => (
              <button
                type="button"
                key={q.id}
                onClick={() => updateSettings({ quality: q.id as Quality })}
                className={`flex-1 py-1 text-xs rounded-md transition-colors ${
                  settings.quality === q.id
                    ? 'bg-neutral-800 text-neutral-100 font-semibold'
                    : 'text-neutral-400 hover:text-neutral-200'
                }`}
              >
                {q.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Export Method */}
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-medium text-neutral-300">Export Method</label>
        <div className="grid grid-cols-2 gap-2">
          <div className="flex items-center justify-between p-3 rounded-lg border border-purple-500/50 bg-purple-500/10 text-xs text-purple-200">
            <div className="flex items-center gap-2">
              <Sparkles size={16} className="text-purple-400" />
              <div>
                <div className="font-semibold">Server Rendering</div>
                <div className="text-[10px] text-purple-300/70">FFmpeg cloud worker</div>
              </div>
            </div>
            <Check size={14} className="text-purple-400" />
          </div>

          <div
            className="flex items-center justify-between p-3 rounded-lg border border-neutral-800 bg-neutral-950/40 text-xs text-neutral-500 cursor-not-allowed opacity-60"
            title="Local browser export will be available in Stage 2"
          >
            <div>
              <div className="font-medium">Local Browser</div>
              <div className="text-[10px] text-neutral-600">Coming in Stage 2</div>
            </div>
            <span className="text-[10px] uppercase font-mono bg-neutral-800 px-1.5 py-0.5 rounded text-neutral-400">
              Disabled
            </span>
          </div>
        </div>
      </div>

      {/* Submit Button */}
      <div className="pt-2">
        <Button
          type="submit"
          disabled={!preflight.valid || isSubmitting}
          className="w-full bg-purple-600 hover:bg-purple-500 text-white font-medium py-2 rounded-lg transition-all"
        >
          {isSubmitting ? (
            'Preparing Export…'
          ) : (
            <span className="flex items-center justify-center gap-2">
              <Film size={16} /> Start Export
            </span>
          )}
        </Button>
      </div>
    </form>
  )
}
