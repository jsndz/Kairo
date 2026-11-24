import test from 'node:test'
import assert from 'node:assert/strict'
import { DEFAULT_EXPORT_SETTINGS } from '../src/features/export/domain/types.ts'
import { exportApi } from '../src/features/export/api/client.ts'
import { useExportStore } from '../src/features/export/state/useExportStore.ts'

test('export default settings invariants', () => {
  assert.equal(DEFAULT_EXPORT_SETTINGS.format, 'mp4')
  assert.equal(DEFAULT_EXPORT_SETTINGS.resolution, '1080p')
  assert.equal(DEFAULT_EXPORT_SETTINGS.frameRate, 30)
  assert.equal(DEFAULT_EXPORT_SETTINGS.quality, 'standard')
  assert.equal(DEFAULT_EXPORT_SETTINGS.method, 'server')
  assert.equal(DEFAULT_EXPORT_SETTINGS.filename, 'export.mp4')
})

test('export download URL generation', () => {
  const url = exportApi.getDownloadURL('job_123_abc')
  assert.equal(url, '/api/exports/job_123_abc/download')
})

test('export store preflight validation checks', () => {
  const { checkPreflight } = useExportStore.getState()

  // Empty timeline
  const emptyCheck = checkPreflight(0, false)
  assert.equal(emptyCheck.valid, false)
  assert.match(emptyCheck.warning, /Timeline is empty/i)

  // Incomplete uploads
  const uploadingCheck = checkPreflight(5, true)
  assert.equal(uploadingCheck.valid, false)
  assert.match(uploadingCheck.warning, /uploading/i)

  // Valid timeline
  const validCheck = checkPreflight(3, false)
  assert.equal(validCheck.valid, true)
  assert.equal(validCheck.warning, null)
})

test('export settings state updates', () => {
  const store = useExportStore.getState()

  store.updateSettings({ resolution: '4k', frameRate: 60, quality: 'high' })
  const updated = useExportStore.getState().settings

  assert.equal(updated.resolution, '4k')
  assert.equal(updated.frameRate, 60)
  assert.equal(updated.quality, 'high')

  // Reset back to defaults
  store.updateSettings(DEFAULT_EXPORT_SETTINGS)
})
