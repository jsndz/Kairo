import { create } from 'zustand'
import { ApiError, begin, complete, fail, register, uploadObject } from '../api/index.ts'
import { cacheFile, readFiles, type CachedFile } from './localMedia'
import { FPS, type Asset, type Source } from '../state/model'
import { errorMessage, serialize, useEditor } from '../state/store'

type LocalFile = CachedFile & { src: string }
type UploadState = { active: boolean; progress?: number; error?: string }

export const useLocalMedia = create<{
  files: Record<string, LocalFile>
  uploads: Record<string, UploadState>
  warning: string
}>(() => ({ files: {}, uploads: {}, warning: '' }))

const progress = (id: string, state: UploadState) =>
  useLocalMedia.setState(s => ({ uploads: { ...s.uploads, [id]: state } }))

const local = (entry: CachedFile): LocalFile => ({ ...entry, src: URL.createObjectURL(entry.file) })

function saveLocal(file: LocalFile, previous?: string) {
  useLocalMedia.setState(state => {
    const files = { ...state.files, [file.id]: file }
    if (!previous) return { files }
    delete files[previous]
    const uploads = { ...state.uploads }
    delete uploads[previous]
    return { files, uploads }
  })
}

async function cache(entry: CachedFile, previous?: string) {
  try {
    const { id, file, replacementId } = entry
    await cacheFile({ id, file, replacementId }, previous)
  } catch {
    useLocalMedia.setState({
      warning: 'The browser could not cache this file. Keep this tab open until the upload finishes.',
    })
  }
}

function applyAsset(work: () => Promise<Asset>) {
  return serialize(async () => {
    const asset = await work()
    useEditor.getState().putAsset(asset)
    return asset
  })
}

const active = new Set<string>()
let restored: Promise<void> | undefined

export function restoreMedia() {
  if (restored) return restored
  restored = (async () => {
    try {
      for (const file of await readFiles()) {
        if (!useLocalMedia.getState().files[file.id]) saveLocal(local(file))
      }
    } catch {
      useLocalMedia.setState({
        warning: 'Local media cache is unavailable. Completed uploads can still play from MinIO.',
      })
    }
    for (const asset of Object.values(useEditor.getState().project.assets)) {
      if (asset.status === 'uploading') void reconcile(asset)
    }
  })()
  return restored
}

async function reconcile(asset: Asset) {
  if (!asset.attempt || active.has(asset.id)) return
  active.add(asset.id)
  progress(asset.id, { active: true })
  try {
    await applyAsset(() => complete(asset.id, asset.attempt!))
    progress(asset.id, { active: false })
  } catch (error) {
    progress(asset.id, {
      active: false,
      error:
        error instanceof ApiError && error.status === 409
          ? useLocalMedia.getState().files[asset.id]
            ? 'Upload interrupted. Retry using the cached file.'
            : 'Upload interrupted. The local file is unavailable.'
          : errorMessage(error),
    })
  } finally {
    active.delete(asset.id)
  }
}

async function inspect(entry: LocalFile): Promise<Source> {
  const { file, src } = entry
  let mime = file.type
  if (!mime) {
    const ext = file.name.split('.').pop()?.toLowerCase() ?? ''
    const mimeMap: Record<string, string> = {
      mp4: 'video/mp4',
      webm: 'video/webm',
      mov: 'video/quicktime',
      mkv: 'video/x-matroska',
      mp3: 'audio/mpeg',
      wav: 'audio/wav',
      aac: 'audio/aac',
      ogg: 'audio/ogg',
      flac: 'audio/flac',
      png: 'image/png',
      jpg: 'image/jpeg',
      jpeg: 'image/jpeg',
      gif: 'image/gif',
      webp: 'image/webp',
    }
    mime = mimeMap[ext] ?? ''
  }
  const category = mime.split('/')[0]
  if (category !== 'image' && category !== 'audio' && category !== 'video') {
    throw new Error('Choose a video, audio, or image file.')
  }
  const element = category === 'image' ? document.createElement('img') : document.createElement(category)
  try {
    await new Promise<void>((resolve, reject) => {
      const timeout = window.setTimeout(() => reject(new Error('Media could not be loaded.')), 10000)
      const loaded = () => {
        clearTimeout(timeout)
        resolve()
      }
      element.onerror = () => {
        clearTimeout(timeout)
        reject(new Error('Unsupported media file.'))
      }
      if (element instanceof HTMLImageElement) element.onload = loaded
      else {
        element.preload = 'metadata'
        element.onloadedmetadata = loaded
      }
      element.src = src
    })
    const duration = element instanceof HTMLImageElement ? 5 : element.duration
    if (!Number.isFinite(duration) || duration <= 0) throw new Error('Media duration is unavailable.')
    const dimensions =
      element instanceof HTMLImageElement
        ? { width: element.naturalWidth, height: element.naturalHeight }
        : element instanceof HTMLVideoElement
          ? { width: element.videoWidth, height: element.videoHeight }
          : {}
    return {
      name: file.name,
      kind: category,
      mime,
      bytes: file.size,
      duration,
      frames: Math.max(1, Math.floor(duration * FPS)),
      ...dimensions,
    }
  } finally {
    element.removeAttribute('src')
    if (element instanceof HTMLMediaElement) element.load()
  }
}

async function transfer(entry: LocalFile, asset: Asset) {
  let attempt = ''
  try {
    progress(asset.id, { active: true, progress: 0 })
    const { asset: startedAsset, upload } = await begin(asset.id)
    if (startedAsset.status === 'completed') {
      useEditor.getState().putAsset(startedAsset)
      progress(asset.id, { active: false })
      return
    }
    attempt = startedAsset.attempt ?? ''
    useEditor.getState().putAsset(startedAsset)
    await uploadObject(upload, entry.file, percent => progress(asset.id, { active: true, progress: percent }))
    await applyAsset(() => complete(asset.id, attempt))
    progress(asset.id, { active: false })
  } catch (error) {
    if (attempt) void fail(asset.id, attempt).catch(() => {})
    progress(asset.id, { active: false, error: errorMessage(error) })
  }
}

export async function importMedia(files: File[], replacement?: Asset) {
  for (const file of files) {
    const draftId = 'draft-' + crypto.randomUUID()
    const entry = local({ id: draftId, file, replacementId: replacement?.id })
    saveLocal(entry)
    progress(draftId, { active: true })
    try {
      const source = await inspect(entry)
      if (replacement && (replacement.kind !== source.kind || source.frames < replacement.frames)) {
        throw new Error('Choose media of the same type and at least the original duration.')
      }
      const asset = await applyAsset(() => register(source))
      const assigned = { ...entry, id: asset.id }
      saveLocal(assigned, draftId)
      await cache({ id: asset.id, file, replacementId: replacement?.id }, draftId)
      if (replacement) {
        await useEditor.getState().dispatch({ type: 'relink', mediaId: replacement.id, replacementId: asset.id })
      }
      void transfer(assigned, asset)
    } catch (error) {
      progress(draftId, { active: false, error: errorMessage(error) })
    }
  }
}

export function retryMedia(id: string) {
  const file = useLocalMedia.getState().files[id]
  const asset = useEditor.getState().project.assets[id]
  if (!file) {
    progress(id, { active: false, error: 'The local file is no longer available in this browser.' })
    return
  }
  if (!asset) {
    void importMedia([file.file])
    return
  }
  void transfer(file, asset)
}
