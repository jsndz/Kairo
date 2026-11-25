import type { CSSProperties } from 'react'

const assetPath = `${import.meta.env.BASE_URL}landing/`

type SceneTheme = CSSProperties & {
  '--landing-text': string
  '--landing-nav-text': string
  '--landing-accent': string
  '--landing-ink': string
  '--landing-backdrop': string
  '--landing-light': string
}

type Scene = {
  id: string
  word: string
  backdrop: string
  cast: string
  alt: string
  theme: SceneTheme
}

export const scenes: readonly Scene[] = [
  {
    id: 'bros',
    word: 'Bros',
    backdrop: `${assetPath}bros-backdrop.webp`,
    cast: `${assetPath}bros-cast.webp`,
    alt: 'Two friends lock hands in an amber-lit forest clearing.',
    theme: {
      '--landing-text': '#fff0d8',
      '--landing-nav-text': '#f4e3ca',
      '--landing-accent': '#ffca86',
      '--landing-ink': '#28180e',
      '--landing-backdrop': '#180f0a',
      '--landing-light': '#532c18',
    },
  },
  {
    id: 'lover',
    word: 'Lover',
    backdrop: `${assetPath}lover-backdrop.webp`,
    cast: `${assetPath}lover-cast.webp`,
    alt: 'A couple touch foreheads and laugh on a rooftop at dusk.',
    theme: {
      '--landing-text': '#ffede9',
      '--landing-nav-text': '#f7e0da',
      '--landing-accent': '#f3c4bb',
      '--landing-ink': '#321926',
      '--landing-backdrop': '#231320',
      '--landing-light': '#644050',
    },
  },
  {
    id: 'boss',
    word: 'Boss',
    backdrop: `${assetPath}boss-backdrop.webp`,
    cast: `${assetPath}boss-cast.webp`,
    alt: 'An employee and their boss exchange a look over a laptop.',
    theme: {
      '--landing-text': '#e9eee9',
      '--landing-nav-text': '#dce3df',
      '--landing-accent': '#c9d8d2',
      '--landing-ink': '#172522',
      '--landing-backdrop': '#101919',
      '--landing-light': '#3b4c49',
    },
  },
]

// Two copies give each scene an identical pair of neighbors around a closed reel.
export const reelFaces = [...scenes, ...scenes]
export const reelStep = 360 / reelFaces.length
export const reelRadiusRatio = 0.5 / Math.tan(reelStep * Math.PI / 360)
