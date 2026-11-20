import { useRef, useState } from 'react'
import { AudioLines, Film, Plus, Search, Upload } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { importMedia, mediaKey } from './media'
import { timecode, type Media } from './model'

export function MediaBin({ media, add }: { media: Media[]; add: (media: Media) => void }) {
  const fileInput = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<'all' | Media['kind']>('all')
  const mutation = useMutation({
    mutationFn: importMedia,
    networkMode: 'always',
    onSuccess: () => queryClient.invalidateQueries({ queryKey: mediaKey }),
  })
  const visible = media.filter(asset => (filter === 'all' || asset.kind === filter) &&
    asset.name.toLowerCase().includes(search.toLowerCase()))
  const messages = mutation.error ? [mutation.error instanceof Error ? mutation.error.message : 'Could not save media.'] : mutation.data ?? []

  return <aside className="media-panel" aria-label="Media library">
    <div className="panel-heading"><h2>Media</h2><span className="technical">{media.length} ASSETS</span></div>
    <div className="media-tools">
      <Button variant="outline" className="import-button" disabled={mutation.isPending} onClick={() => fileInput.current?.click()}>
        <Upload /> {mutation.isPending ? 'Importing…' : 'Import media'}
      </Button>
      <input ref={fileInput} type="file" accept="video/*,audio/*" multiple hidden onChange={event => {
        const files = Array.from(event.currentTarget.files ?? [])
        if (files.length) mutation.mutate(files)
        event.currentTarget.value = ''
      }} />
      <label className="search"><Search size={15} /><input aria-label="Search media" placeholder="Search media" value={search} onChange={event => setSearch(event.target.value)} /></label>
      <div className="filters" aria-label="Filter media">
        {(['all', 'video', 'audio'] as const).map(kind => <Button key={kind} variant="ghost" size="sm" aria-pressed={filter === kind}
          className={filter === kind ? 'active-filter' : ''} onClick={() => setFilter(kind)}>{kind === 'all' ? 'All media' : kind === 'video' ? 'Video' : 'Audio'}</Button>)}
      </div>
    </div>
    {messages.length > 0 && <p className="message-error" role="alert">{messages.join(' ')}</p>}
    <div className="media-list">
      {visible.map(asset => <button type="button" className="media-row" key={asset.id} onClick={() => add(asset)} title={'Add ' + asset.name + ' to timeline'}>
        <span className={'media-thumb ' + asset.kind}>{asset.poster ? <img src={asset.poster} alt="" /> : asset.kind === 'audio' ? <AudioLines size={19} /> : <Film size={19} />}</span>
        <span className="media-details"><strong>{asset.name}</strong><small>{timecode(asset.frames, false)} · {asset.kind === 'video' ? 'Video' : 'Audio'}</small></span>
        <Plus size={15} className="asset-add" />
      </button>)}
      {!visible.length && <p className="media-empty">No matching media</p>}
    </div>
    <div className="media-footer">Click an asset to add it to the timeline.</div>
  </aside>
}
