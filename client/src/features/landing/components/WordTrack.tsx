import { scenes } from '../config/scenes'

export function WordTrack() {
  return (
    <span className="landing-word-window" aria-hidden="true">
      {scenes.map(scene => (
        <span className="landing-title" data-slide={scene.id} key={scene.id}>
          {scene.word}
        </span>
      ))}
    </span>
  )
}
