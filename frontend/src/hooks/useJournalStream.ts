import { useEffect, useState } from 'react'
import type { FrontendClient } from '../shared/api/frontend-client'
import { normalizeAlarmGroup, normalizeEventItem } from '../shared/api/normalize'
import type { FrontendAlarmGroup, FrontendEventItem } from '../shared/api/types'
import { JOURNAL_BOOTSTRAP_EVENTS_LIMIT, JOURNAL_WS_URL } from '../features/operator/constants'
import { mergeRecentEvents, sliceRecentEvents } from '../features/operator/utils'

export function useJournalStream(api: FrontendClient): { events: FrontendEventItem[]; alarmGroups: FrontendAlarmGroup[]; connected: boolean } {
  const [events, setEvents] = useState<FrontendEventItem[]>([])
  const [alarmGroups, setAlarmGroups] = useState<FrontendAlarmGroup[]>([])
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    let disposed = false
    let ws: WebSocket | null = null
    let reconnectTimer: number | null = null
    let fallbackPollTimer: number | null = null

    const clearReconnectTimer = () => {
      if (reconnectTimer != null) {
        window.clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
    }

    const clearFallbackPollTimer = () => {
      if (fallbackPollTimer != null) {
        window.clearInterval(fallbackPollTimer)
        fallbackPollTimer = null
      }
    }

    const pullFallbackSnapshot = async () => {
      try {
        const [freshEvents, freshAlarmGroups] = await Promise.all([api.listEvents(), api.listAlarmGroups()])
        if (disposed) {
          return
        }
        setEvents(sliceRecentEvents(freshEvents, JOURNAL_BOOTSTRAP_EVENTS_LIMIT))
        setAlarmGroups(freshAlarmGroups)
      } catch {
        // fallback polling best-effort, errors are ignored until websocket reconnects
      }
    }

    const startFallbackPolling = () => {
      if (fallbackPollTimer != null || disposed) {
        return
      }
      void pullFallbackSnapshot()
      fallbackPollTimer = window.setInterval(() => {
        void pullFallbackSnapshot()
      }, 20_000)
    }

    const stopFallbackPolling = () => {
      clearFallbackPollTimer()
    }

    const connect = () => {
      if (disposed) {
        return
      }

      ws = new WebSocket(JOURNAL_WS_URL)
      ws.onopen = () => {
        if (disposed) {
          return
        }
        setConnected(true)
        stopFallbackPolling()
      }
      ws.onmessage = (event) => {
        if (disposed || typeof event.data !== 'string') {
          return
        }
        try {
          const payload = JSON.parse(event.data) as {
            kind?: unknown
            events?: unknown[]
            alarmGroups?: unknown[]
            alarmGroupsChanged?: unknown
          }
          const kind = typeof payload.kind === 'string' ? payload.kind : ''
          if (Array.isArray(payload.events)) {
            const incomingEvents = payload.events.map(normalizeEventItem)
            if (kind === 'bootstrap') {
              setEvents(sliceRecentEvents(incomingEvents, JOURNAL_BOOTSTRAP_EVENTS_LIMIT))
            } else {
              setEvents((prev) => mergeRecentEvents(prev, incomingEvents))
            }
          }
          const alarmGroupsChanged = payload.alarmGroupsChanged === true
          if (Array.isArray(payload.alarmGroups)) {
            const incomingAlarmGroups = payload.alarmGroups.map(normalizeAlarmGroup)
            if (kind === 'bootstrap' || alarmGroupsChanged) {
              setAlarmGroups(incomingAlarmGroups)
            }
          } else if (alarmGroupsChanged) {
            setAlarmGroups([])
          }
        } catch {
          // ignore malformed websocket payloads
        }
      }
      ws.onerror = () => {
        ws?.close()
      }
      ws.onclose = () => {
        if (disposed) {
          return
        }
        setConnected(false)
        startFallbackPolling()
        clearReconnectTimer()
        reconnectTimer = window.setTimeout(connect, 2_000)
      }
    }

    startFallbackPolling()
    connect()

    return () => {
      disposed = true
      clearReconnectTimer()
      clearFallbackPollTimer()
      if (ws != null) {
        ws.close()
      }
    }
  }, [api])

  return { events, alarmGroups, connected }
}
