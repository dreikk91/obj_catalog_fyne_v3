import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'off'

const _orig = {
  debug: console.debug.bind(console),
  log:   console.log.bind(console),
  info:  console.info.bind(console),
  warn:  console.warn.bind(console),
  error: console.error.bind(console),
}

const LEVELS: Record<LogLevel, number> = { debug: 0, info: 1, warn: 2, error: 3, off: 4 }
const noop = (..._args: unknown[]) => {}

export function applyLogLevel(level: LogLevel) {
  const min = LEVELS[level]
  console.debug = min <= 0 ? _orig.debug : noop
  console.log   = min <= 1 ? _orig.log   : noop
  console.info  = min <= 1 ? _orig.info  : noop
  console.warn  = min <= 2 ? _orig.warn  : noop
  console.error = min <= 3 ? _orig.error : noop
}

type LogStore = {
  logLevel: LogLevel
  setLogLevel: (level: LogLevel) => void
}

export const useLogStore = create<LogStore>()(
  persist(
    (set) => ({
      logLevel: 'warn',
      setLogLevel: (logLevel) => set({ logLevel }),
    }),
    { name: 'frontend-log-level' },
  ),
)
