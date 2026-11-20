import { demoMedia, FPS, type Media } from './model'

export const mediaKey = ['project-media'] as const

type StoredMedia = Omit<Media, 'src'> & { file: File }

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('kairo-media', 1)
    request.onupgradeneeded = () => request.result.createObjectStore('media', { keyPath: 'id' })
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

const urls = new Map<string, string>()

export async function loadMedia(): Promise<Media[]> {
  const database = await openDatabase()
  try {
    const files = await new Promise<StoredMedia[]>((resolve, reject) => {
      const request = database.transaction('media').objectStore('media').getAll()
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })
    return [...demoMedia, ...files.map(({ file, ...media }) => {
      const src = urls.get(media.id) ?? URL.createObjectURL(file)
      urls.set(media.id, src)
      return { ...media, src }
    })]
  } finally {
    database.close()
  }
}

export async function importMedia(files: File[]) {
  const imported: StoredMedia[] = []
  const errors: string[] = []
  for (const file of files) {
    try {
      const kind = file.type.startsWith('audio/') ? 'audio' : file.type.startsWith('video/') ? 'video' : null
      if (!kind) throw new Error('Choose a video or audio file.')
      const src = URL.createObjectURL(file)
      const element = document.createElement(kind)
      try {
        const seconds = await new Promise<number>((resolve, reject) => {
          element.preload = 'metadata'
          const timeout = window.setTimeout(() => reject(new Error('Media could not be loaded.')), 10000)
          element.onloadedmetadata = () => { clearTimeout(timeout); resolve(element.duration) }
          element.onerror = () => { clearTimeout(timeout); reject(new Error('Unsupported media format.')) }
          element.src = src
        })
        if (!Number.isFinite(seconds) || seconds <= 0) throw new Error('Media duration is unavailable.')
        imported.push({ id: crypto.randomUUID(), name: file.name, kind, frames: Math.max(1, Math.floor(seconds * FPS)), file })
      } finally {
        element.removeAttribute('src')
        element.load()
        URL.revokeObjectURL(src)
      }
    } catch (error) {
      errors.push(file.name + ' — ' + (error instanceof Error ? error.message : 'Import failed.'))
    }
  }
  if (imported.length) {
    const database = await openDatabase()
    try {
      await new Promise<void>((resolve, reject) => {
        const transaction = database.transaction('media', 'readwrite')
        transaction.oncomplete = () => resolve()
        transaction.onabort = () => reject(transaction.error ?? new Error('Could not save media.'))
        for (const item of imported) transaction.objectStore('media').put(item)
      })
    } finally {
      database.close()
    }
  }
  return errors
}
