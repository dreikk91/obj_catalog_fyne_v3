import type { FrontendClient } from './frontend-client'
import type {
  FrontendCapabilities,
  FrontendAlarmItem,
  FrontendAlarmGroup,
  FrontendAlarmPickRequest,
  FrontendAlarmProcessingOption,
  FrontendDBSettings,
  FrontendEventPage,
  FrontendEventItem,
  FrontendObjectDetails,
  FrontendObjectSummary,
  FrontendResponseGroup,
} from './types'
import {
  normalizeCapabilities,
  normalizeAlarmItem,
  normalizeAlarmGroup,
  normalizeAlarmProcessingOption,
  normalizeEventItem,
  normalizeEventPage,
  normalizeObjectDetails,
  normalizeObjectSummary,
  normalizeResponseGroup,
} from './normalize'

const FRONTEND_API_BASE = import.meta.env.VITE_FRONTEND_API_BASE ?? '/api/frontend/v1'

type ObjectListResponse = { items: FrontendObjectSummary[] }
type CapabilitiesResponse = FrontendCapabilities
type AlarmListResponse = { items: FrontendAlarmItem[] }
type AlarmGroupListResponse = { items: FrontendAlarmGroup[] }
type AlarmProcessingOptionsResponse = { items: FrontendAlarmProcessingOption[] }
type ResponseGroupListResponse = { items: FrontendResponseGroup[] }
type EventListResponse = { items: FrontendEventItem[] }
type EventPageResponse = FrontendEventPage

export function createHTTPFrontendClient(): FrontendClient {
  return {
    async capabilities() {
      const body = await fetchJSON<CapabilitiesResponse>(`${FRONTEND_API_BASE}/capabilities`)
      return normalizeCapabilities(body)
    },
    async listObjects() {
      const body = await fetchJSON<ObjectListResponse>(`${FRONTEND_API_BASE}/objects`)
      return body.items.map(normalizeObjectSummary)
    },
    async listEvents() {
      const body = await fetchJSON<EventListResponse>(`${FRONTEND_API_BASE}/events`)
      return body.items.map(normalizeEventItem)
    },
    async listAlarms() {
      const body = await fetchJSON<AlarmListResponse>(`${FRONTEND_API_BASE}/alarms`)
      return body.items.map(normalizeAlarmItem)
    },
    async listAlarmGroups() {
      const body = await fetchJSON<AlarmGroupListResponse>(`${FRONTEND_API_BASE}/alarm-groups`)
      return body.items.map(normalizeAlarmGroup)
    },
    async listAlarmProcessingOptionsCached() {
      const body = await fetchJSON<AlarmProcessingOptionsResponse>(`${FRONTEND_API_BASE}/alarm-processing-options`)
      return body.items.map(normalizeAlarmProcessingOption)
    },
    async listResponseGroups() {
      const body = await fetchJSON<ResponseGroupListResponse>(`${FRONTEND_API_BASE}/response-groups`)
      return body.items.map(normalizeResponseGroup)
    },
    async getAlarmProcessingOptions(alarmID) {
      const body = await fetchJSON<AlarmProcessingOptionsResponse>(`${FRONTEND_API_BASE}/alarms/${alarmID}/processing-options`)
      return body.items.map(normalizeAlarmProcessingOption)
    },
    async pickAlarm(alarmID, request: FrontendAlarmPickRequest) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/alarms/${alarmID}/pick`, {
        method: 'POST',
        body: JSON.stringify(request),
      })
    },
    async processAlarm(alarmID, request) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/alarms/${alarmID}/process`, {
        method: 'POST',
        body: JSON.stringify(request),
      })
    },
    async groupProcessAlarm(alarmID, user) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/alarms/${alarmID}/group-process`, {
        method: 'POST',
        body: JSON.stringify({ User: user }),
      })
    },
    async assignResponseGroup(alarmID, request) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/alarms/${alarmID}/assign-group`, {
        method: 'POST',
        body: JSON.stringify(request),
      })
    },
    async notifyGroupArrived(alarmID) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/alarms/${alarmID}/group-arrived`, {
        method: 'POST',
      })
    },
    async cancelResponseGroup(alarmID) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/alarms/${alarmID}/cancel-group`, {
        method: 'POST',
      })
    },
    async standbyObject(objectID, durationMinutes, reason) {
      await fetchJSON<void>(`${FRONTEND_API_BASE}/objects/${objectID}/standby`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ durationMinutes, reason }),
      })
    },
    async listObjectEvents(objectID, offset, limit) {
      const params = new URLSearchParams({
        offset: String(offset),
        limit: String(limit),
      })
      const body = await fetchJSON<EventPageResponse>(`${FRONTEND_API_BASE}/objects/${objectID}/events?${params.toString()}`)
      return normalizeEventPage(body)
    },
    async getObjectDetails(objectID) {
      const body = await fetchJSON<FrontendObjectDetails>(`${FRONTEND_API_BASE}/objects/${objectID}`)
      return normalizeObjectDetails(body)
    },
    async getDBSettings() {
      throw new Error('Налаштування доступні лише у Wails застосунку')
    },
    async saveDBSettings(_: FrontendDBSettings) {
      throw new Error('Налаштування доступні лише у Wails застосунку')
    },
  }
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status} while requesting ${url}`)
  }
  if (response.status === 204) {
    return undefined as T
  }
  return (await response.json()) as T
}
