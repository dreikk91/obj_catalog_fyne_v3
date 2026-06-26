import { useEffect, useState } from 'react'

let _now = Date.now()
const _listeners = new Set<() => void>()
let _timer: ReturnType<typeof setInterval> | null = null

function tick() {
  _now = Date.now()
  _listeners.forEach((fn) => fn())
}

function subscribe(fn: () => void) {
  _listeners.add(fn)
  if (_timer === null) {
    _timer = setInterval(tick, 1000)
  }
}

function unsubscribe(fn: () => void) {
  _listeners.delete(fn)
  if (_listeners.size === 0 && _timer !== null) {
    clearInterval(_timer)
    _timer = null
  }
}

export function useNow(): number {
  const [now, setNow] = useState(_now)
  useEffect(() => {
    const handler = () => setNow(_now)
    subscribe(handler)
    return () => unsubscribe(handler)
  }, [])
  return now
}

export function formatAge(nowMs: number, timestampMs: number): string {
  if (!timestampMs) return '—'
  const s = Math.max(0, Math.floor((nowMs - timestampMs) / 1000))
  if (s < 60) return `${s}с`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}хв`
  const h = Math.floor(m / 60)
  return `${h}г ${m % 60}хв`
}

export function ageUrgencyClass(nowMs: number, timestampMs: number): string {
  if (!timestampMs) return ''
  const minutes = (nowMs - timestampMs) / 60000
  if (minutes > 15) return 'age-critical'
  if (minutes > 10) return 'age-high'
  if (minutes > 5) return 'age-medium'
  return ''
}
