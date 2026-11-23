import type { Asset, Media } from '../../state/model.ts'

/**
 * Resolves registered project assets to playable Media objects by prioritizing local files
 * and falling back to signed MinIO content URLs.
 */
export function resolveAvailableMedia(
  assets: Record<string, Asset>,
  localFiles: Record<string, { src?: string }>,
  failedUrls: ReadonlySet<string>,
  contentUrlFn: (id: string) => string,
): Media[] {
  return Object.values(assets).flatMap(asset => {
    const local = localFiles[asset.id]?.src
    const remote = contentUrlFn(asset.id)
    const src =
      local && !failedUrls.has(local)
        ? local
        : asset.status === 'completed' && !failedUrls.has(remote)
          ? remote
          : null

    return src ? [{ ...asset, src }] : []
  })
}
