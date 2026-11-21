import { memo, useEffect, useMemo, useState } from 'react'
import { Player, type CallbackListener } from '@remotion/player'
import { AbsoluteFill, Html5Audio, Html5Video, Sequence, Img } from 'remotion'
import { Maximize, Pause, Play, SkipBack, SkipForward, Volume2, VolumeX } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { duration, FPS, timecode, type Clip, type Media, type Track } from '../state/model'
import { usePlaybackFrame, type Playback } from '../state/playback'

type PreviewInput = {
  clips: Clip[]
  tracks: Track[]
  media: Media[]
  onMediaError: (src: string) => void
}

function Composition({ clips, tracks, media, onMediaError }: PreviewInput) {
  const ordered = useMemo(() => {
    const assets = new Map(media.map(asset => [asset.id, asset]))
    const order = new Map(tracks.map((track, index) => [track.id, index]))
    return [...clips]
      .sort((a, b) => order.get(a.trackId)! - order.get(b.trackId)!)
      .flatMap(clip => {
        const asset = assets.get(clip.mediaId)
        return asset ? [{ clip, asset }] : []
      })
  }, [clips, tracks, media])

  return (
    <AbsoluteFill style={{ backgroundColor: '#050607' }}>
      {ordered.map(({ clip, asset }) => (
        <Sequence key={clip.id} from={clip.at} durationInFrames={clip.length} premountFor={FPS}>
          {asset.kind === 'video' ? (
            <Html5Video
              key={asset.src}
              src={asset.src}
              trimBefore={clip.offset}
              volume={clip.volume}
              pauseWhenBuffering
              onError={() => onMediaError(asset.src)}
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          ) : asset.kind === 'audio' ? (
            <Html5Audio
              key={asset.src}
              src={asset.src}
              trimBefore={clip.offset}
              volume={clip.volume}
              pauseWhenBuffering
              onError={() => onMediaError(asset.src)}
            />
          ) : (
            <Img
              key={asset.src}
              src={asset.src}
              onError={() => onMediaError(asset.src)}
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          )}
        </Sequence>
      ))}
    </AbsoluteFill>
  )
}

function TimeDisplay({ player, frames }: { player: Playback; frames: number }) {
  const frame = usePlaybackFrame(player)
  return (
    <div className="time-display">
      <strong>{timecode(frame)}</strong>
      <span> / {timecode(frames)}</span>
    </div>
  )
}

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
