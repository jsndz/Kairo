import { memo, useRef, useState } from 'react'
import { AudioLines, Film, Image, Plus, Search, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { timecode, type Asset, type Media } from '../state/model'
import { importMedia, retryMedia, useLocalMedia } from '../media/mediaManager'

export const MediaBin = memo(function MediaBin({
  assets,
  media,
  add,
}: {
  assets: Record<string, Asset>
  media: Media[]
  add: (media: Media) => void
}) {
  const fileInput = useRef<HTMLInputElement>(null)
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<'all' | Media['kind']>('all')
  const files = useLocalMedia(state => state.files)
  const uploads = useLocalMedia(state => state.uploads)
  const warning = useLocalMedia(state => state.warning)
  const drafts = Object.values(files).filter(entry => entry.id.startsWith('draft-'))
  const matches = (name: string, kind: string) =>
    (filter === 'all' || kind === filter) && name.toLowerCase().includes(search.toLowerCase())

  return (
    <aside className="media-panel" aria-label="Media library">
      <div className="panel-heading">
        <h2>Media</h2>
        <span className="technical">{media.length} ASSETS</span>
      </div>
      <div className="media-tools">
        <Button variant="outline" className="import-button" onClick={() => fileInput.current?.click()}>
          <Upload /> Import media
        </Button>
        <input
          ref={fileInput}
          type="file"
          accept="video/*,audio/*,image/*"
          multiple
          hidden
          onChange={event => {
            const selected = Array.from(event.currentTarget.files ?? [])
            if (selected.length) void importMedia(selected)
            event.currentTarget.value = ''
          }}
        />
        <label className="search">
          <Search size={15} />
          <input
            aria-label="Search media"
            placeholder="Search media"
            value={search}
            onChange={event => setSearch(event.target.value)}
          />
        </label>
        <div className="filters" aria-label="Filter media">
          {(['all', 'video', 'audio', 'image'] as const).map(kind => (
            <Button
              key={kind}
              variant="ghost"
              size="sm"
              aria-pressed={filter === kind}
              className={filter === kind ? 'active-filter' : ''}
              onClick={() => setFilter(kind)}
            >
              {kind === 'all'
                ? 'All media'
                : kind === 'video'
                  ? 'Video'
                  : kind === 'audio'
                    ? 'Audio'
                    : 'Images'}
            </Button>
          ))}
        </div>
      </div>
      {warning && (
        <p className="message-error" role="status">
          {warning}
        </p>
      )}
      <div className="media-list">
        {drafts.map(entry => (
          <div className="media-row pending-draft" key={entry.id}>
            <span className="media-thumb audio">
              <Upload size={19} />
            </span>
            <span className="media-details">
              <strong>{entry.file.name}</strong>
              <small>Processing media…</small>
            </span>
          </div>
        ))}
        {media
          .filter(asset => matches(asset.name, asset.kind))
          .map(asset => {
            const upload = uploads[asset.id]
            const record = assets[asset.id]
            const failed = upload?.error || record?.status === 'failed'
            return (
              <button
                type="button"
                className={'media-row ' + (failed ? 'upload-error' : '')}
                key={asset.id}
                onClick={() => add(asset)}
                title={'Add ' + asset.name + ' to timeline'}
              >
                <span className={'media-thumb ' + asset.kind}>
                  {asset.poster ? (
                    <img src={asset.poster} alt="" />
                  ) : asset.kind === 'audio' ? (
                    <AudioLines size={19} />
                  ) : asset.kind === 'image' ? (
                    <Image size={19} />
                  ) : (
                    <Film size={19} />
                  )}
                </span>
                <span className="media-details">
                  <strong>{asset.name}</strong>
                  <small>
                    {timecode(asset.frames, false)} ·{' '}
                    {upload?.active
                      ? `Uploading… ${upload.progress ?? 0}%`
                      : failed
                        ? 'Upload failed'
                        : asset.kind}
                  </small>
                </span>
                {failed ? (
                  <span
                    role="button"
                    tabIndex={0}
                    className="retry-upload"
                    onClick={e => {
                      e.stopPropagation()
                      retryMedia(asset.id)
                    }}
                    onKeyDown={e => {
                      if (e.key === 'Enter') {
                        e.stopPropagation()
                        retryMedia(asset.id)
                      }
                    }}
                  >
                    Retry
                  </span>
                ) : (
                  <Plus size={15} className="asset-add" />
                )}
              </button>
            )
          })}
        {!media.length && !drafts.length && <p className="media-empty">No matching media</p>}
      </div>
      <div className="media-footer">Click an asset to add it to the timeline.</div>
    </aside>
  )
})
