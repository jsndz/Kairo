import { useCallback, useMemo, useState } from 'react'
import type { Asset, Media } from '../state/model.ts'
import { resolveAvailableMedia } from '../domain/media/resolve.ts'
import { contentURL } from '../api/client.ts'

export function useAvailableMedia(
  assets: Record<string, Asset>,
  localFiles: Record<string, { src?: string }>,
) {
  const [failed, setFailed] = useState<ReadonlySet<string>>(() => new Set())

  const available: Media[] = useMemo(
    () => resolveAvailableMedia(assets, localFiles, failed, contentURL),
    [assets, localFiles, failed],
  )

  const markFailed = useCallback((src: string) => {
    setFailed(previous => (previous.has(src) ? previous : new Set([...previous, src])))
  }, [])

  return { available, markFailed }
}
