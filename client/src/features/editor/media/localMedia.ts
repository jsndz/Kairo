export type CachedFile = { id: string; file: File; replacementId?: string }

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('kairo-stage1-media', 1)
    request.onupgradeneeded = () => request.result.createObjectStore('files', { keyPath: 'id' })
    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve(request.result)
  })
}

export async function readFiles(): Promise<CachedFile[]> {
  const db = await open()
  try {
    return await new Promise((resolve, reject) => {
      const request = db.transaction('files').objectStore('files').getAll()
      request.onerror = () => reject(request.error)
      request.onsuccess = () => resolve(request.result)
    })
  } finally {
    db.close()
  }
}

export async function cacheFile(entry: CachedFile, previous?: string) {
  const db = await open()
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction('files', 'readwrite')
      tx.onabort = () => reject(tx.error)
      tx.oncomplete = () => resolve()
      const files = tx.objectStore('files')
      files.put(entry)
      if (previous) files.delete(previous)
    })
  } finally {
    db.close()
  }
}
