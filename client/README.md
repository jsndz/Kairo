# Kairo editor client

Start Go and MinIO using the [repository setup](../README.md), then run `npm ci` and `npm run dev` here.

Import video, audio, or images. Click an asset to append it to its track. Imported images start at five seconds and use video tracks. Select a clip to change its track, trim it, or adjust its volume. Drag clips to move them and use the ruler to seek. Use **S** to split, **Delete** to remove, **Ctrl/Cmd+Z** to undo, and **Ctrl/Cmd+Shift+Z** to redo. Higher video tracks appear in front. Clip edges may touch on the same track.

Files preview from the browser cache while uploads run in the background. Use **Retry upload** after an interrupted transfer. A clip without a playable source remains on the timeline; select it to relink the media. Download project exports a JSON description of the current edit. Video rendering is outside this stage.
