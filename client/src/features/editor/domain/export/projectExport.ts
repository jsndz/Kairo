import type { Project } from '../../state/model.ts'

/**
 * Serializes project state into formatted JSON for backup or export.
 */
export function serializeProject(project: Project): string {
  return JSON.stringify({ version: 2, project }, null, 2)
}

/**
 * Triggers a browser file download using a temporary Object URL.
 */
export function downloadFile(content: string, filename: string, mimeType: string = 'application/json'): void {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/**
 * Exports project as a downloadable `.kairo.json` file.
 */
export function exportProjectToJSON(project: Project): void {
  const content = serializeProject(project)
  downloadFile(content, `${project.name}.kairo.json`, 'application/json')
}
