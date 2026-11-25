import { scenes } from '../config/scenes'

export function SceneTrack() {
  return (
    <div className="landing-scene-window" aria-hidden="true">
      {scenes.map((scene, index) => (
        <div className="landing-scene" data-slide={scene.id} key={scene.id}>
          <img
            className="landing-scene-backdrop"
            src={scene.backdrop}
            alt=""
            fetchPriority={index === 0 ? 'high' : 'low'}
            decoding="async"
          />
          <img
            className="landing-scene-cast"
            src={scene.cast}
            alt=""
            fetchPriority={index === 0 ? 'high' : 'low'}
            decoding="async"
          />
        </div>
      ))}
    </div>
  )
}
