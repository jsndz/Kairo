export type Resolution = '720p' | '1080p' | '4k'
export type FrameRate = 24 | 30 | 60
export type Quality = 'standard' | 'high'
export type Format = 'mp4'
export type Method = 'server' | 'local'

export type ExportSettings = {
  format: Format
  filename: string
  resolution: Resolution
  frameRate: FrameRate
  quality: Quality
  method: Method
}

export type JobStatus = 'queued' | 'rendering' | 'completed' | 'failed' | 'cancelled'

export type ExportJob = {
  id: string
  requestId: string
  projectId: string
  revision: number
  status: JobStatus
  progress: number // 0 to 100
  attempts: number
  error?: string
  outputKey?: string
  createdAt: string
  startedAt?: string
  endedAt?: string
}

export type CreateExportRequest = {
  requestId: string
  revision: number
  settings: ExportSettings
}

export const DEFAULT_EXPORT_SETTINGS: ExportSettings = {
  format: 'mp4',
  filename: 'export.mp4',
  resolution: '1080p',
  frameRate: 30,
  quality: 'standard',
  method: 'server',
}
