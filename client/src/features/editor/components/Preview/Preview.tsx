import { memo, useEffect, useMemo, useState } from 'react'
import { Player, type CallbackListener } from '@remotion/player'
import { Maximize, Pause, Play, SkipBack, SkipForward, Volume2, VolumeX } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { duration, FPS } from '../../state/model.ts'
import type { Playback } from '../../state/playback.ts'
import { Composition, type PreviewInput } from './Composition.tsx'
import { TimeDisplay } from './TimeDisplay.tsx'

export const Preview = memo(function Preview({
  clips,
  tracks,
  media,
  player,
  onMediaError,
}: PreviewInput & { player: Playback }) {
  const [playing, setPlaying] = useState(false)
  const [muted, setMuted] = useState(false)
  const frames = duration(clips)
  const inputProps = useMemo(() => ({ clips, tracks, media, onMediaError }), [clips, tracks, media, onMediaError])

  useEffect(() => {
    const current = player.current
    if (!current) return
    const play = () => setPlaying(true)
    const pause = () => setPlaying(false)
    const mute: CallbackListener<'mutechange'> = event => setMuted(event.detail.isMuted)
    current.addEventListener('play', play)
    current.addEventListener('pause', pause)
    current.addEventListener('ended', pause)
    current.addEventListener('mutechange', mute)
    return () => {
      current.removeEventListener('play', play)
      current.removeEventListener('pause', pause)
      current.removeEventListener('ended', pause)
      current.removeEventListener('mutechange', mute)
    }
  }, [player])

  useEffect(() => {
    const current = player.current
    if (current && current.getCurrentFrame() >= frames) current.seekTo(Math.max(0, frames - 1))
  }, [frames, player])

  return (
    <section className="preview-panel" aria-label="Preview">
      <div className="panel-heading">
        <h2>Preview</h2>
        <span className="technical">16:9 · 30 FPS</span>
      </div>
      <div className="preview-stage">
        <div className="preview-canvas">
          <Player
            ref={player}
            component={Composition}
            inputProps={inputProps}
            durationInFrames={frames}
            compositionWidth={1280}
            compositionHeight={720}
            fps={FPS}
            style={{ width: '100%', height: '100%' }}
            moveToBeginningWhenEnded={false}
            errorFallback={() => <div className="preview-message">This media could not be played.</div>}
          />
          {!clips.length && <div className="preview-message">Add media to start editing</div>}
        </div>
      </div>
      <div className="transport">
        <TimeDisplay player={player} frames={frames} />
        <div className="transport-controls">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Go to start"
            title="Go to start"
            onClick={() => player.current?.seekTo(0)}
          >
            <SkipBack />
          </Button>
          <Button
            className="play-button"
            size="icon"
            disabled={!clips.length}
            aria-label={playing ? 'Pause' : 'Play'}
            title="Play / pause (Space)"
            onClick={event => player.current?.toggle(event)}
          >
            {playing ? <Pause /> : <Play />}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Go to end"
            title="Go to end"
            onClick={() => player.current?.seekTo(Math.max(0, frames - 1))}
          >
            <SkipForward />
          </Button>
        </div>
        <div className="transport-end">
          <Button
            variant="ghost"
            size="icon"
            aria-label={muted ? 'Unmute preview' : 'Mute preview'}
            title={muted ? 'Unmute preview' : 'Mute preview'}
            onClick={() => (muted ? player.current?.unmute() : player.current?.mute())}
          >
            {muted ? <VolumeX /> : <Volume2 />}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Fullscreen"
            title="Fullscreen"
            onClick={() => player.current?.requestFullscreen()}
          >
            <Maximize />
          </Button>
        </div>
      </div>
    </section>
  )
})
