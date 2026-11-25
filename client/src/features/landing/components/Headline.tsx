import { WordTrack } from './WordTrack'

export function Headline({ activeWord }: { activeWord: string }) {
  return (
    <h1 className="landing-headline">
      <span className="landing-headline-anchor">Edit videos with your</span>
      <span className="sr-only"> {activeWord}</span>
      <WordTrack />
    </h1>
  )
}
