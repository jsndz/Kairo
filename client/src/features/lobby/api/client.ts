export class LobbyApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function createRoom(name?: string): Promise<{ roomId: string; name: string }> {
  const res = await fetch('/api/rooms', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: name?.trim() || 'Collaboration Room' }),
    signal: AbortSignal.timeout(15000),
  })

  if (!res.ok) {
    const failure = await res.json().catch(() => null)
    throw new LobbyApiError(res.status, failure?.error ?? 'Failed to create room')
  }

  return res.json()
}
