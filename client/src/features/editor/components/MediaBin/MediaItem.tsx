import { AudioLines, Film, Image, Plus } from 'lucide-react'
import { timecode, type Asset, type Media } from '../../state/model.ts'
import { retryMedia } from '../../media/mediaManager.ts'

export function MediaItem({
  asset,
  record,
  upload,
  onAdd,
}: {
  asset: Media
  record?: Asset
  upload?: { active?: boolean; progress?: number; error?: unknown }
  onAdd: (media: Media) => void
}) {
  const failed = Boolean(upload?.error || record?.status === 'failed')

  return (
    <button
      type="button"
      className={'media-row ' + (failed ? 'upload-error' : '')}
      onClick={() => onAdd(asset)}
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
}
