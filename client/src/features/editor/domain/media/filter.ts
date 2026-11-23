import type { Media } from '../../state/model.ts'

export type MediaFilterKind = 'all' | Media['kind']

/**
 * Tests whether a media item matches current filter kind and search query.
 */
export function matchesMediaFilter(
  name: string,
  kind: string,
  search: string,
  filter: MediaFilterKind,
): boolean {
  const matchesKind = filter === 'all' || kind === filter
  const matchesSearch = name.toLowerCase().includes(search.toLowerCase())
  return matchesKind && matchesSearch
}
