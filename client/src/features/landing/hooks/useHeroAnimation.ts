import { useRef, useState } from 'react'
import { useGSAP } from '@gsap/react'
import gsap from 'gsap'
import { Observer } from 'gsap/Observer'
import { reelFaces, reelStep, scenes } from '../config/scenes'

gsap.registerPlugin(useGSAP, Observer)

type Direction = -1 | 1

export function useHeroAnimation() {
  const heroRef = useRef<HTMLElement>(null)
  const moveRef = useRef<((direction: Direction) => void) | null>(null)
  const [activeIndex, setActiveIndex] = useState(0)

  useGSAP(() => {
    const hero = heroRef.current
    if (!hero) return

    const backdrop = hero.querySelector('[data-camera="backdrop"]')
    const cast = hero.querySelector('[data-camera="cast"]')
    if (!backdrop || !cast) return

    const reels = hero.querySelectorAll('.landing-reel')
    const backdropFrames = backdrop.querySelectorAll('.landing-scene-art')
    const castFrames = cast.querySelectorAll('.landing-scene-art')
    const wrapFace = gsap.utils.wrap(0, reelFaces.length)
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    let turn = 0
    let transition: gsap.core.Timeline | null = null
    let wheelConsumed = false
    let swipeStartedDuringTransition = false

    const move = (direction: Direction) => {
      if (transition) return

      const nextTurn = turn + direction
      const nextFace = wrapFace(nextTurn)
      const next = nextFace % scenes.length
      // Keep the rotation continuous across both ends of the scene list.
      const angle = nextTurn * reelStep
      const theme = scenes[next].theme

      if (reducedMotion.matches) {
        gsap.set(reels, { rotationX: angle })
        gsap.set(hero, theme)
        turn = nextTurn
        setActiveIndex(next)
        return
      }

      const backdropDepth = -hero.clientHeight * 0.67
      const castDepth = backdropDepth * 0.88
      const currentFace = wrapFace(turn)
      const rollingArt = [
        backdropFrames[currentFace], backdropFrames[nextFace],
        castFrames[currentFace], castFrames[nextFace],
      ]
      transition = gsap.timeline({
        onComplete: () => {
          turn = nextTurn
          transition = null
        },
      })

      transition
        .to(backdrop, { z: backdropDepth, duration: 0.32, ease: 'power2.inOut' }, 0)
        .to(cast, { z: castDepth, yPercent: direction * 1.5, duration: 0.32, ease: 'power2.inOut' }, 0)
        .addLabel('roll', '+=0.03')
        .to(reels, { rotationX: angle + direction * 4.6, duration: 0.52, ease: 'power3.inOut' }, 'roll')
        .to(cast, { yPercent: -direction * 1.5, duration: 0.52, ease: 'power3.inOut' }, 'roll')
        .to(hero, { ...theme, duration: 0.36, ease: 'sine.inOut' }, 'roll+=0.09')
        .to(rollingArt, { filter: 'blur(3px)', duration: 0.16, ease: 'power2.in' }, 'roll+=0.09')
        .to(rollingArt, { filter: 'blur(0px)', duration: 0.19, ease: 'power2.out' }, 'roll+=0.25')
        .set(rollingArt, { clearProps: 'filter' }, 'roll+=0.44')
        .addLabel('catch', 'roll+=0.52')
        .to(reels, { rotationX: angle - direction * 1.2, duration: 0.075, ease: 'power2.in' }, 'catch')
        .to(reels, { rotationX: angle, duration: 0.095, ease: 'power2.out' }, 'catch+=0.075')
        .to(backdrop, { z: backdropDepth * 0.94, duration: 0.075, ease: 'power2.out' }, 'catch')
        .to(backdrop, { z: backdropDepth, duration: 0.095, ease: 'power2.inOut' }, 'catch+=0.075')
        .addLabel('push', 'catch+=0.17')
        .call(() => setActiveIndex(next), [], 'push')
        .to([backdrop, cast], { z: 0, yPercent: 0, duration: 0.44, ease: 'power3.inOut' }, 'push')
    }

    moveRef.current = move

    const wheel = Observer.create({
      target: hero,
      type: 'wheel',
      tolerance: 14,
      onStopDelay: 0.28,
      ignoreCheck: event => {
        const { ctrlKey, deltaX, deltaY } = event as WheelEvent
        return ctrlKey || Math.abs(deltaX) > Math.abs(deltaY)
      },
      onStop: () => { wheelConsumed = false },
      onChangeY: ({ deltaY }) => {
        if (wheelConsumed) return
        // A trackpad can keep sending wheel events after the turn has finished.
        wheelConsumed = true
        move(deltaY > 0 ? 1 : -1)
      },
    })

    const touch = Observer.create({
      target: hero,
      type: 'touch',
      dragMinimum: 10,
      ignoreCheck: event =>
        ('touches' in event && (event as TouchEvent).touches.length > 1) ||
        (/^(touchstart|pointerdown)$/.test(event.type) &&
          event.target instanceof Element && Boolean(event.target.closest('a, button'))),
      onPress: () => { swipeStartedDuringTransition = transition !== null },
      onDragEnd: ({ event, startX, startY, x, y }) => {
        if (startX === undefined || startY === undefined || x === undefined || y === undefined) return
        const travel = y - startY
        if (swipeStartedDuringTransition || event.type.endsWith('cancel') ||
            Math.abs(travel) < 32 || Math.abs(travel) <= Math.abs(x - startX)) return
        move(travel < 0 ? 1 : -1)
      },
    })

    function handleKey(event: KeyboardEvent) {
      if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || event.repeat) return
      const direction = ['ArrowDown', 'PageDown'].includes(event.key) ? 1
        : ['ArrowUp', 'PageUp'].includes(event.key) ? -1 : null
      if (direction === null) return
      event.preventDefault()
      move(direction)
    }

    function handleMotionPreference() {
      if (reducedMotion.matches) transition?.progress(1)
    }

    window.addEventListener('keydown', handleKey)
    reducedMotion.addEventListener('change', handleMotionPreference)

    return () => {
      moveRef.current = null
      wheel.kill()
      touch.kill()
      transition?.kill()
      window.removeEventListener('keydown', handleKey)
      reducedMotion.removeEventListener('change', handleMotionPreference)
    }
  }, { scope: heroRef })

  function moveScene(direction: Direction) {
    moveRef.current?.(direction)
  }

  return { heroRef, activeIndex, moveScene }
}
