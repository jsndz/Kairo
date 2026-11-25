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

let activeProjectId = ''

export function setActiveProjectId(id: string) {
  activeProjectId = id
}

export function getActiveProjectId(): string {
  return activeProjectId
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

export const getProject = (projectId?: string) => {
  const pid = projectId || activeProjectId
  return request<ProjectState>(`/projects/${encodeURIComponent(pid)}`)
}

export function commitEdit(id: string, expectedRevision: number, { type, ...payload }: Edit, projectId?: string) {
  const pid = projectId || activeProjectId
  return request<ProjectState>(`/projects/${encodeURIComponent(pid)}/operations`, { id, expectedRevision, type, payload })
}

export const register = (source: Source, projectId?: string) => {
  const pid = projectId || activeProjectId
  return request<Asset>(`/projects/${encodeURIComponent(pid)}/assets`, source)
}

export const begin = (id: string, projectId?: string) => {
  const pid = projectId || activeProjectId
  return request<{ asset: Asset; upload: Upload }>(`/projects/${encodeURIComponent(pid)}/assets/${id}/upload`, {})
}

export const complete = (id: string, attempt: string, projectId?: string) => {
  const pid = projectId || activeProjectId
  return request<Asset>(`/projects/${encodeURIComponent(pid)}/assets/${id}/complete`, { attempt })
}

export const fail = (id: string, attempt: string, projectId?: string) => {
  const pid = projectId || activeProjectId
  return request<Asset>(`/projects/${encodeURIComponent(pid)}/assets/${id}/fail`, { attempt })
}

export const contentURL = (id: string, projectId?: string) => {
  const pid = projectId || activeProjectId
  return `/api/projects/${encodeURIComponent(pid)}/assets/${id}/content`
}
