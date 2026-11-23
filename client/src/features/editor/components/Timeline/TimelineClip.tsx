import { memo, useState } from 'react'
import { useDraggable } from '@dnd-kit/core'
import { timecode, type Asset, type Clip, type Media } from '../../state/model.ts'
import { useEditor } from '../../state/store.ts'
import { calculateDragTransformX } from '../../domain/timeline/drag.ts'
import { ClipContent } from './ClipContent.tsx'
import { TrimHandle } from './TrimHandle.tsx'

export const TimelineClip = memo(function TimelineClip({
  clip,
  asset,
  source,
  scale,
}: {
  clip: Clip
  asset: Asset
  source?: Media
  scale: number
}) {
  const selected = useEditor(state => state.selected === clip.id)
  const peerSelected = useEditor(state => state.peerPresence?.selected === clip.id)
  const select = useEditor(state => state.select)
  const owner = useEditor(state => state.owners[clip.id])
  const myClientId = useEditor(state => state.myClientId)
  const isPending = useEditor(state => state.outgoingPermission?.clipId === clip.id)
  const requestPermission = useEditor(state => state.requestPermission)
  const dispatch = useEditor(state => state.dispatch)

  const isOwner = Boolean(myClientId && owner === myClientId)
  const isPeerOwner = Boolean(owner && owner !== myClientId)

  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: clip.id,
    disabled: isPeerOwner,
  })
  const [preview, setPreview] = useState<Clip | null>(null)
  const shown = preview ?? clip

  const ownershipClass = isPending
    ? 'pending-transfer'
    : isOwner
      ? 'owner-local'
      : isPeerOwner
        ? 'owner-peer'
        : 'owner-unassigned'

  return (
    <div
      ref={setNodeRef}
      className={`timeline-clip ${asset.kind} ${selected ? 'selected' : ''} ${peerSelected ? 'peer-selected' : ''} ${ownershipClass} ${source ? '' : 'missing'}`}
      style={{
        left: shown.at * scale,
        width: Math.max(4, shown.length * scale),
        transform: transform
          ? `translateX(${calculateDragTransformX(clip.at, scale, transform.x)}px)`
          : undefined,
        backgroundImage: source?.poster
          ? `linear-gradient(180deg,#171327db,#251c3f69),url("${source.poster}")`
          : undefined,
      }}
    >
      <button
        type="button"
        {...(isPeerOwner ? {} : attributes)}
        {...(isPeerOwner ? {} : listeners)}
        aria-pressed={selected}
        className="clip-body"
        title={asset.name + ' · ' + timecode(shown.length)}
        onClick={() => select(clip.id)}
      >
        <ClipContent asset={asset} source={source} clip={shown} />
      </button>

      {/* Ownership tag badge */}
      {isPeerOwner && <span className="clip-ownership-badge peer">Peer</span>}
      {isOwner && <span className="clip-ownership-badge local">You</span>}
      {isPending && <span className="clip-ownership-badge pending">Requesting…</span>}

      {/* Peer-owned context action to request edit */}
      {isPeerOwner && selected && (
        <button
          type="button"
          className="request-edit-btn"
          onClick={e => {
            e.stopPropagation()
            void requestPermission(clip.id)
          }}
          disabled={isPending}
        >
          {isPending ? 'Requesting…' : 'Request Edit'}
        </button>
      )}

      {selected &&
        !isPeerOwner &&
        (['left', 'right'] as const).map(edge => (
          <TrimHandle
            key={edge}
            edge={edge}
            clip={clip}
            assetFrames={asset.frames}
            scale={scale}
            onPreview={setPreview}
            onCommit={trimUpdate => {
              void dispatch({ type: 'trim', clipId: clip.id, ...trimUpdate })
            }}
          />
        ))}
    </div>
  )
})
