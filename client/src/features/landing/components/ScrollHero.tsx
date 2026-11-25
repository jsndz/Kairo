import type { CSSProperties } from 'react'
import { reelRadiusRatio, scenes } from '../config/scenes'
import { useHeroAnimation } from '../hooks/useHeroAnimation'
import { Headline } from './Headline'
import { Navbar } from './Navbar'
import { PosterReel } from './PosterReel'
import { SceneControls } from './SceneControls'

const initialStyle = {
  ...scenes[0].theme,
  '--landing-reel-radius-ratio': reelRadiusRatio,
} as CSSProperties

export function ScrollHero() {
  const { heroRef, activeIndex, moveScene } = useHeroAnimation()

  return (
    <main
      className="landing-page landing-hero"
      ref={heroRef}
      style={initialStyle}
    >
      <PosterReel layer="backdrop" />
      <Headline activeWord={scenes[activeIndex].word} />
      <PosterReel layer="cast" />
      <Navbar />
      <SceneControls activeIndex={activeIndex} moveScene={moveScene} />
    </main>
  )
}
