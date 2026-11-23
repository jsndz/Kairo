import { useEffect } from 'react'

export function useNoticeDismiss(
  notice: string | null,
  setNotice: (value: string | null) => void,
  durationMs: number = 6000,
) {
  useEffect(() => {
    if (!notice) return
    const timer = setTimeout(() => {
      setNotice(null)
    }, durationMs)
    return () => clearTimeout(timer)
  }, [notice, setNotice, durationMs])
}
