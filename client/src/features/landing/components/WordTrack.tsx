import type { CSSProperties } from 'react'
import { reelFaces, reelStep } from '../config/scenes'

export function WordTrack() {
  return (
    <span className="landing-word-window" aria-hidden="true">
      <span className="landing-reel-axis">
        <span className="landing-reel">
          {reelFaces.map((scene, index) => (
            <span
              className="landing-reel-face"
              key={`${scene.id}-${index}`}
              style={{ '--reel-face-angle': `${-index * reelStep}deg` } as CSSProperties}
            >
              <span className="landing-title" data-scene={scene.id}>{scene.word}</span>
            </span>
          ))}
        </span>
      </span>
    </span>
  )
}
