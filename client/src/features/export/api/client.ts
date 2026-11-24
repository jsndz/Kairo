import type { CreateExportRequest, ExportJob } from '../domain/types.ts'

export class ExportApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch('/api' + path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    signal: AbortSignal.timeout(15000),
  })

  if (!response.ok) {
    const failure = await response.json().catch(() => null)
    throw new ExportApiError(
      response.status,
      failure?.error ?? 'Failed to communicate with the export service.',
    )
  }

  return response.json()
}

export const exportApi = {
  createExport: (req: CreateExportRequest) =>
    request<ExportJob>('/exports', {
      method: 'POST',
      body: JSON.stringify(req),
    }),

  getExport: (id: string) => request<ExportJob>('/exports/' + encodeURIComponent(id)),

  cancelExport: (id: string) =>
    request<ExportJob>('/exports/' + encodeURIComponent(id) + '/cancel', {
      method: 'POST',
    }),

  retryExport: (id: string) =>
    request<ExportJob>('/exports/' + encodeURIComponent(id) + '/retry', {
      method: 'POST',
    }),

  getDownloadURL: (id: string) => '/api/exports/' + encodeURIComponent(id) + '/download',
}
