import { useRef, useState } from 'react'
import { useGSAP } from '@gsap/react'
import gsap from 'gsap'
import { Observer } from 'gsap/Observer'
import { scenes } from '../config/scenes'

gsap.registerPlugin(useGSAP, Observer)

type Direction = -1 | 1

export function useHeroAnimation() {
  const heroRef = useRef<HTMLElement>(null)
  const moveRef = useRef<((direction: Direction) => void) | null>(null)
  const [activeIndex, setActiveIndex] = useState(0)

  useGSAP(() => {
    const hero = heroRef.current
    if (!hero) return

    const slides = scenes.map(scene => ({
      panels: hero.querySelectorAll<HTMLElement>(`[data-slide="${scene.id}"]`),
      backdrop: hero.querySelector<HTMLElement>(`[data-slide="${scene.id}"] .landing-scene-backdrop`),
    }))
    const wrap = gsap.utils.wrap(0, scenes.length)
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    let currentIndex = 0
    let transition: gsap.core.Timeline | null = null
    let wheelConsumed = false
    let swipeStartedDuringTransition = false

    const move = (direction: Direction) => {
      if (transition) return

      const nextIndex = wrap(currentIndex + direction)
      const outgoing = slides[currentIndex]
      const incoming = slides[nextIndex]
      const accent = { '--landing-accent': scenes[nextIndex].accent }
      const updateScene = () => {
        currentIndex = nextIndex
        setActiveIndex(nextIndex)
      }

      if (reducedMotion.matches) {
        gsap.set(outgoing.panels, { visibility: 'hidden' })
        gsap.set(incoming.panels, { visibility: 'visible', yPercent: 0 })
        gsap.set(incoming.backdrop, { yPercent: 0 })
        gsap.set(hero, accent)
        updateScene()
        return
      }

      const duration = 0.8
      const distance = 100 * direction
      // Counter the panel travel slightly so the background moves more slowly than the cast.
      const parallax = 7 * direction
      gsap.set(incoming.panels, { visibility: 'visible', yPercent: distance })
      gsap.set(incoming.backdrop, { yPercent: -parallax })

      transition = gsap.timeline({
        defaults: { duration, ease: 'power3.inOut' },
        onComplete: () => { transition = null },
      })
        .to(outgoing.panels, { yPercent: -distance }, 0)
        .to(incoming.panels, { yPercent: 0 }, 0)
        .to(outgoing.backdrop, { yPercent: parallax }, 0)
        .to(incoming.backdrop, { yPercent: 0 }, 0)
        .to(hero, accent, 0)
        .call(updateScene, [], duration / 2)
        .set(outgoing.panels, { visibility: 'hidden' }, duration)
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
