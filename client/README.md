# Kairo editor client

Run `npm ci` and `npm run dev` from this directory. Use `npm run build` to type-check and create the production bundle.

The starter project includes two clips and an audio bed. Import video or audio, then click an asset to append it to its track. Drag clips on the timeline and click or drag the ruler to seek. Drag either edge of a selected clip to trim it. Use the inspector to edit its start time, source offset, duration, or volume. Use **S** to split, **Delete** to remove a clip, and **Ctrl/Cmd+Z** to undo. Press **Space** to play or pause when focus is outside an input or button. For keyboard dragging, focus a clip and use **Space**, the left or right arrow keys, and **Space** again to drop.

- `src/features/editor/model.ts` defines the frame-based project model.
- `store.ts` manages timeline edits, selection, history, and local project persistence with Zustand.
- `media.ts` stores imported files in IndexedDB. TanStack Query manages media library reads and imports.
- `Preview.tsx` renders the timeline with Remotion Player. `Timeline.tsx` uses dnd-kit and `Waveform.tsx` uses WaveSurfer.js for audio visualization.
- `MediaBin.tsx` and `Inspector.tsx` handle media selection and clip properties. The app uses the existing shadcn/ui button and Tailwind setup.

The current project and its imported files are stored in this browser. Download project exports a JSON edit manifest. Video rendering and server collaboration belong to the backend milestones.

Sample footage by Nguyễn Thành Nhơn on Pexels. [Coastal drift](https://www.pexels.com/video/scenic-aerial-view-of-rocky-coastline-35603903/) and [Breaking tide](https://www.pexels.com/video/aerial-view-of-rocky-coastline-and-ocean-waves-35732026/). The sample audio bed is synthesized.
