import type { CSSProperties } from 'react'
import { reelFaces, reelStep } from '../config/scenes'

type PosterReelProps = {
  layer: 'backdrop' | 'cast'
}

export function PosterReel({ layer }: PosterReelProps) {
  return (
    <div className={`landing-art-window landing-art-window--${layer}`} aria-hidden="true">
      <div className="landing-camera">
        <div className="landing-camera-dolly" data-camera={layer}>
          <div className="landing-reel-axis">
            <div className="landing-reel">
              {reelFaces.map((scene, index) => (
                <div
                  className="landing-reel-face landing-scene"
                  key={`${scene.id}-${index}`}
                  data-scene={scene.id}
                  style={{ '--reel-face-angle': `${-index * reelStep}deg` } as CSSProperties}
                >
                  <img
                    className="landing-scene-art"
                    src={scene[layer]}
                    alt=""
                    loading="eager"
                    fetchPriority={index === 0 ? 'high' : 'low'}
                    decoding="async"
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
