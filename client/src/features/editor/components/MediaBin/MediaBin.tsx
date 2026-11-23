import { memo, useRef, useState } from 'react'
import { Search, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { Asset, Media } from '../../state/model.ts'
import { importMedia, useLocalMedia } from '../../media/mediaManager.ts'
import { matchesMediaFilter, type MediaFilterKind } from '../../domain/media/filter.ts'
import { MediaItem } from './MediaItem.tsx'

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
  const [filter, setFilter] = useState<MediaFilterKind>('all')
  const files = useLocalMedia(state => state.files)
  const uploads = useLocalMedia(state => state.uploads)
  const warning = useLocalMedia(state => state.warning)
  const drafts = Object.values(files).filter(entry => entry.id.startsWith('draft-'))
  const remotePending = Object.values(assets).filter(
    a => (a.status === 'uploading' || a.status === 'pending') && !media.some(m => m.id === a.id),
  )

  const isMatching = (name: string, kind: string) => matchesMediaFilter(name, kind, search, filter)

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
        {remotePending
          .filter(asset => isMatching(asset.name, asset.kind))
          .map(asset => (
            <div
              className="media-row pending-draft opacity-50 grayscale cursor-not-allowed"
              key={asset.id}
              title="Asset is uploading…"
            >
              <span className={'media-thumb ' + asset.kind}>
                <Upload size={19} />
              </span>
              <span className="media-details">
                <strong>{asset.name}</strong>
                <small>Uploading… (Syncing)</small>
              </span>
            </div>
          ))}
        {media
          .filter(asset => isMatching(asset.name, asset.kind))
          .map(asset => (
            <MediaItem
              key={asset.id}
              asset={asset}
              record={assets[asset.id]}
              upload={uploads[asset.id]}
              onAdd={add}
            />
          ))}
        {!media.length && !drafts.length && <p className="media-empty">No matching media</p>}
      </div>
      <div className="media-footer">Click an asset to add it to the timeline.</div>
    </aside>
  )
})
