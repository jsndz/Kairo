import type { CSSProperties } from 'react'
import { assetPath, scenes } from '../config/scenes'
import { useHeroAnimation } from '../hooks/useHeroAnimation'
import { Headline } from './Headline'
import { Navbar } from './Navbar'
import { SceneControls } from './SceneControls'
import { SceneTrack } from './SceneTrack'

const initialStyle = {
  '--landing-accent': scenes[0].accent,
} as CSSProperties

export function ScrollHero() {
  const { heroRef, activeIndex, moveScene } = useHeroAnimation()

  return (
    <main
      className="landing-page landing-hero"
      ref={heroRef}
      style={initialStyle}
    >
      <SceneTrack />
      <Navbar />
      <div className="landing-product">
        <Headline activeWord={scenes[activeIndex].word} />
        <p className="landing-description">
          A real-time collaborative video editor for two. Share a timeline and make every cut together.
        </p>
        <img
          className="landing-editor-preview"
          src={`${assetPath}editor.webp`}
          alt="Kairo editor showing a video preview, clip controls and a shared editing timeline"
          width={1440}
          height={900}
        />
      </div>
      <SceneControls activeIndex={activeIndex} moveScene={moveScene} />
    </main>
  )
}
