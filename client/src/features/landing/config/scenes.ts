export const assetPath = `${import.meta.env.BASE_URL}landing/`

type Scene = {
  id: string
  word: string
  backdrop: string
  cast: string
  alt: string
  accent: string
}

export const scenes: readonly Scene[] = [
  {
    id: 'bros',
    word: 'Bros',
    backdrop: `${assetPath}bros-backdrop.webp`,
    cast: `${assetPath}bros-cast.webp`,
    alt: 'Two friends lock hands in an amber-lit forest clearing.',
    accent: '#ffca86',
  },
  {
    id: 'lover',
    word: 'Lover',
    backdrop: `${assetPath}lover-backdrop.webp`,
    cast: `${assetPath}lover-cast.webp`,
    alt: 'A couple touch foreheads and laugh on a rooftop at dusk.',
    accent: '#f3c4bb',
  },
  {
    id: 'boss',
    word: 'Boss',
    backdrop: `${assetPath}boss-backdrop.webp`,
    cast: `${assetPath}boss-cast.webp`,
    alt: 'An employee and their boss exchange a look over a laptop.',
    accent: '#c9d8d2',
  },
]
