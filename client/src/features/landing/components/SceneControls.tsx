import { ArrowDown, ArrowUp } from 'lucide-react'
import { scenes } from '../config/scenes'

type SceneControlsProps = {
  activeIndex: number
  moveScene: (direction: -1 | 1) => void
}

export function SceneControls({ activeIndex, moveScene }: SceneControlsProps) {
  return (
    <div className="landing-scene-controls" role="group" aria-label="Scenes">
      <button
        type="button"
        aria-label="Previous scene"
        onClick={() => moveScene(-1)}
      >
        <ArrowUp size={18} aria-hidden="true" />
      </button>
      <div className="landing-scene-status">
        <span className="landing-scene-hint">Scroll or swipe</span>
        <span className="landing-scene-progress" aria-hidden="true">
          {scenes.map((scene, index) => <span key={scene.id} data-current={index === activeIndex} />)}
        </span>
        <span className="sr-only" role="status">
          {scenes[activeIndex].word}, scene {activeIndex + 1} of {scenes.length}. {scenes[activeIndex].alt}
        </span>
      </div>
      <button
        type="button"
        aria-label="Next scene"
        onClick={() => moveScene(1)}
      >
        <ArrowDown size={18} aria-hidden="true" />
      </button>
    </div>
  )
}
