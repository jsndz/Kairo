import type { Asset, Project, Source } from '../state/model'
import type { Edit } from '../state/operations'

export type Upload = { url: string; fields: Record<string, string> }
export type ProjectState = { id: string; revision: number; project: Project; canUndo: boolean; canRedo: boolean }

export class ApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, body?: object): Promise<T> {
  const response = await fetch('/api' + path, {
    ...(body ? { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) } : {}),
    signal: AbortSignal.timeout(30000),
  })
  if (!response.ok) {
    const failure = await response.json().catch(() => null)
    throw new ApiError(response.status, failure?.error ?? 'Could not reach the Kairo server.')
  }
  return response.json()
}

export const getProject = () => request<ProjectState>('/project')
export function commitEdit(id: string, expectedRevision: number, { type, ...payload }: Edit) {
  return request<ProjectState>('/project/operations', { id, expectedRevision, type, payload })
}
export const register = (source: Source) => request<Asset>('/assets', source)
export const begin = (id: string) => request<{ asset: Asset; upload: Upload }>('/assets/' + id + '/upload', {})
export const complete = (id: string, attempt: string) => request<Asset>('/assets/' + id + '/complete', { attempt })
export const fail = (id: string, attempt: string) => request<Asset>('/assets/' + id + '/fail', { attempt })
export const contentURL = (id: string) => '/api/assets/' + id + '/content'
