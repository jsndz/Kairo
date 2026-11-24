import { useState } from 'react'
import {
  AlertCircle,
  Film,
  Sparkles,
  Check,
  Cpu,
  Zap,
  HelpCircle,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useExportStore } from '../state/useExportStore.ts'
import type { FrameRate, Quality, Resolution } from '../domain/types.ts'
import type { Clip, Media, Track } from '../../editor/state/model.ts'

type Props = {
  revision: number
  clipsCount: number
  hasIncompleteAssets: boolean
  clips?: Clip[]
  tracks?: Track[]
  media?: Media[]
}

export function ExportSettingsForm({
  revision,
  clipsCount,
  hasIncompleteAssets,
  clips,
  tracks,
  media,
}: Props) {
  const settings = useExportStore(state => state.settings)
  const updateSettings = useExportStore(state => state.updateSettings)
  const startExport = useExportStore(state => state.startExport)
  const isSubmitting = useExportStore(state => state.isSubmitting)
  const checkPreflight = useExportStore(state => state.checkPreflight)
  const accelerationMode = useExportStore(state => state.accelerationMode)
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
      clips,
      tracks,
      media,
    })
  }

  const isLocalUnsupported = accelerationMode === 'unsupported'

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
        <div className="flex items-center bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-1.5 focus-within:border-purple-500 focus-within:ring-1 focus-within:ring-purple-500">
          <input
            id="export-filename"
            type="text"
            value={filenameInput}
            onChange={e => setFilenameInput(e.target.value)}
            onBlur={handleFilenameBlur}
            placeholder="export.mp4"
            className="w-full bg-transparent text-sm text-neutral-100 placeholder-neutral-600 focus:outline-hidden"
          />
        </div>
      </div>

      {/* Export Method Selection */}
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-medium text-neutral-300">Export Method</label>
        <div className="grid grid-cols-2 gap-2">
          {/* Server Rendering */}
          <button
            type="button"
            onClick={() => updateSettings({ method: 'server' })}
            className={`flex items-center justify-between p-3 rounded-lg border text-xs text-left transition-all ${
              settings.method === 'server'
                ? 'border-purple-500/80 bg-purple-500/15 text-purple-100 shadow-sm'
                : 'border-neutral-800 bg-neutral-950/60 text-neutral-400 hover:border-neutral-700 hover:text-neutral-200'
            }`}
          >
            <div className="flex items-center gap-2.5">
              <Sparkles
                size={16}
                className={settings.method === 'server' ? 'text-purple-400' : 'text-neutral-400'}
              />
              <div>
                <div className="font-semibold">Server Cloud</div>
                <div className="text-[10px] text-neutral-400">FFmpeg background worker</div>
              </div>
            </div>
            {settings.method === 'server' && <Check size={14} className="text-purple-400 shrink-0" />}
          </button>

          {/* Local In-Browser Rendering */}
          <button
            type="button"
            disabled={isLocalUnsupported}
            onClick={() => updateSettings({ method: 'local' })}
            className={`flex items-center justify-between p-3 rounded-lg border text-xs text-left transition-all ${
              isLocalUnsupported
                ? 'border-neutral-800/50 bg-neutral-950/30 text-neutral-600 cursor-not-allowed opacity-60'
                : settings.method === 'local'
                  ? 'border-emerald-500/80 bg-emerald-500/15 text-emerald-100 shadow-sm'
                  : 'border-neutral-800 bg-neutral-950/60 text-neutral-400 hover:border-neutral-700 hover:text-neutral-200'
            }`}
          >
            <div className="flex items-center gap-2.5 min-w-0">
              {accelerationMode === 'hardware' ? (
                <Zap size={16} className="text-emerald-400 shrink-0" />
              ) : accelerationMode === 'software' ? (
                <Cpu size={16} className="text-amber-400 shrink-0" />
              ) : (
                <HelpCircle size={16} className="text-neutral-500 shrink-0" />
              )}
              <div className="min-w-0">
                <div className="font-semibold flex items-center gap-1.5 truncate">
                  <span>Local Browser</span>
                </div>
                <div className="text-[10px] truncate">
                  {accelerationMode === 'hardware' ? (
                    <span className="text-emerald-400 font-mono">GPU Hardware</span>
                  ) : accelerationMode === 'software' ? (
                    <span className="text-amber-400 font-mono">CPU Software</span>
                  ) : accelerationMode === 'checking' ? (
                    <span className="text-neutral-400">Probing GPU…</span>
                  ) : (
                    <span className="text-rose-400">Unsupported</span>
                  )}
                </div>
              </div>
            </div>
            {settings.method === 'local' && (
              <Check size={14} className="text-emerald-400 shrink-0" />
            )}
          </button>
        </div>
      </div>

      {/* Resolution Selector */}
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-medium text-neutral-300">Resolution</label>
        <div className="grid grid-cols-3 gap-2">
          {(
            [
              { id: '720p', label: '720p HD', desc: '1280 × 720' },
              { id: '1080p', label: '1080p FHD', desc: '1920 × 1080' },
              { id: '4k', label: '4K UHD', desc: '3840 × 2160' },
            ] as const
          ).map(res => (
            <button
              type="button"
              key={res.id}
              onClick={() => updateSettings({ resolution: res.id as Resolution })}
              className={`flex flex-col items-center justify-center p-2.5 rounded-lg border text-xs transition-all ${
                settings.resolution === res.id
                  ? 'border-purple-500 bg-purple-500/10 text-neutral-100 font-medium shadow-xs'
                  : 'border-neutral-800 bg-neutral-950 text-neutral-400 hover:border-neutral-700 hover:text-neutral-200'
              }`}
            >
              <span>{res.label}</span>
              <span className="text-[10px] text-neutral-500 font-mono mt-0.5">{res.desc}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Frame Rate & Quality Grid */}
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
                { id: 'high', label: 'High' },
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

      {/* Submit Button */}
      <div className="pt-2">
        <Button
          type="submit"
          disabled={!preflight.valid || isSubmitting}
          className={`w-full font-medium py-2 rounded-lg transition-all ${
            settings.method === 'local'
              ? 'bg-emerald-600 hover:bg-emerald-500 text-white'
              : 'bg-purple-600 hover:bg-purple-500 text-white'
          }`}
        >
          {isSubmitting ? (
            'Preparing Export…'
          ) : (
            <span className="flex items-center justify-center gap-2">
              <Film size={16} />
              {settings.method === 'local' ? 'Export in Browser' : 'Start Cloud Export'}
            </span>
          )}
        </Button>
      </div>
    </form>
  )
}
