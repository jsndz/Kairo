import type { Upload } from './client'

export function uploadObject(upload: Upload, file: File, progress: (percent: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const body = new FormData()
    for (const [name, value] of Object.entries(upload.fields)) body.append(name, value)
    body.append('file', file)
    const xhr = new XMLHttpRequest()
    xhr.open('POST', upload.url)
    xhr.timeout = 30 * 60 * 1000
    xhr.upload.onprogress = event => {
      if (event.lengthComputable) progress(Math.min(99, Math.round((event.loaded / event.total) * 100)))
    }
    xhr.onload = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error('MinIO rejected the upload.')))
    xhr.onerror = xhr.ontimeout = xhr.onabort = () => reject(new Error('The upload was interrupted.'))
    xhr.send(body)
  })
}
