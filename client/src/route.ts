export type Route =
  | { view: 'home' }
  | { view: 'lobby' }
  | { view: 'editor'; roomId: string }

export function parseRoute(hash: string): Route {
  if (!hash || hash === '#') return { view: 'home' }

  const route = hash.replace(/^#\/?/, '')
  if (route.startsWith('room/')) {
    const roomId = route.slice('room/'.length).split('?')[0].trim()
    if (roomId) return { view: 'editor', roomId }
  }
  // Also support project/ prefix for link backwards compatibility
  if (route.startsWith('project/')) {
    const roomId = route.slice('project/'.length).split('?')[0].trim()
    if (roomId) return { view: 'editor', roomId }
  }
  return { view: 'lobby' }
}
