import { MAX_FRAMES, type Clip } from './model.ts'

export type Edit =
  | { type: 'add'; clip: Clip }
  | { type: 'move'; clipId: string; trackId: string; at: number }
  | { type: 'trim'; clipId: string; at: number; offset: number; length: number }
  | { type: 'split'; clipId: string; frame: number; rightId: string }
  | { type: 'delete'; clipId: string }
  | { type: 'volume'; clipId: string; volume: number }
  | { type: 'add-track'; trackId: string; kind?: 'video' | 'audio' }
  | { type: 'delete-track'; trackId: string }
  | { type: 'rename'; name: string }
  | { type: 'reset' | 'undo' | 'redo' }
  | { type: 'relink'; mediaId: string; replacementId: string }

export function edgeTrim(clip: Clip, sourceFrames: number, edge: 'left' | 'right', delta: number) {
  if (edge === 'left') {
    const shift = Math.max(-Math.min(clip.at, clip.offset), Math.min(clip.length - 1, delta))
    return { at: clip.at + shift, offset: clip.offset + shift, length: clip.length - shift }
  }
  return {
    at: clip.at,
    offset: clip.offset,
    length: Math.max(1, Math.min(sourceFrames - clip.offset, MAX_FRAMES - clip.at, clip.length + delta)),
  }
}
