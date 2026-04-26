import type { FrontendClient } from './frontend-client'
import type {
  FrontendCapabilities,
  FrontendAlarmGroupActionRequest,
  FrontendAlarmItem,
  FrontendAlarmGroup,
  FrontendAlarmPickRequest,
  FrontendAlarmProcessRequest,
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
  normalizeDBSettings,
  normalizeEventPage,
  normalizeEventItem,
  normalizeObjectDetails,
  normalizeObjectSummary,
  normalizeResponseGroup,
} from './normalize'

type WailsMethod<T extends unknown[] = unknown[], R = unknown> = (...args: T) => Promise<R>

type WailsFrontendBridge = {
  Capabilities?: WailsMethod<[], FrontendCapabilities>
  ListObjects?: WailsMethod<[], FrontendObjectSummary[]>
  ListEvents?: WailsMethod<[], FrontendEventItem[]>
  ListAlarms?: WailsMethod<[], FrontendAlarmItem[]>
  ListAlarmGroups?: WailsMethod<[], FrontendAlarmGroup[]>
  ListAlarmProcessingOptionsCached?: WailsMethod<[], FrontendAlarmProcessingOption[]>
  ListResponseGroups?: WailsMethod<[], FrontendResponseGroup[]>
  GetAlarmProcessingOptions?: WailsMethod<[number], FrontendAlarmProcessingOption[]>
  PickAlarm?: WailsMethod<[number, FrontendAlarmPickRequest], void>
  ProcessAlarm?: WailsMethod<[number, FrontendAlarmProcessRequest], void>
  GroupProcessAlarm?: WailsMethod<[number, string], void>
  AssignResponseGroup?: WailsMethod<[number, FrontendAlarmGroupActionRequest], void>
  NotifyGroupArrived?: WailsMethod<[number], void>
  CancelResponseGroup?: WailsMethod<[number], void>
  StandbyObject?: WailsMethod<[number, number, string], void>
  ListObjectEvents?: WailsMethod<[number, number, number], FrontendEventPage>
  GetObjectDetails?: WailsMethod<[number], FrontendObjectDetails>
}

type WailsOperatorSettingsBridge = {
  GetDBSettings?: WailsMethod<[], FrontendDBSettings>
  SaveDBSettings?: WailsMethod<[FrontendDBSettings], void>
}

type WailsNamespace = {
  wailsbridge?: {
    FrontendV1Service?: WailsFrontendBridge
    OperatorSettingsService?: WailsOperatorSettingsBridge
  }
  main?: {
    FrontendV1Service?: WailsFrontendBridge
    OperatorSettingsService?: WailsOperatorSettingsBridge
  }
}

declare global {
  interface Window {
    go?: WailsNamespace
  }
}

export function createWailsFrontendClient(): FrontendClient | null {
  const frontendBridge = resolveFrontendBridge()
  if (frontendBridge == null) {
    return null
  }
  const settingsBridge = resolveSettingsBridge()

  const listObjects = frontendBridge.ListObjects
  const capabilities = frontendBridge.Capabilities
  const listEvents = frontendBridge.ListEvents
  const listAlarms = frontendBridge.ListAlarms
  const listAlarmGroups = frontendBridge.ListAlarmGroups
  const getAlarmProcessingOptions = frontendBridge.GetAlarmProcessingOptions
  const pickAlarm = frontendBridge.PickAlarm
  const processAlarm = frontendBridge.ProcessAlarm
  const listObjectEvents = frontendBridge.ListObjectEvents
  const getObjectDetails = frontendBridge.GetObjectDetails

  if (!capabilities || !listObjects || !listEvents || !listAlarms || !listAlarmGroups || !getAlarmProcessingOptions || !pickAlarm || !processAlarm || !listObjectEvents || !getObjectDetails) {
    return null
  }

  return {
    async capabilities() {
      const result = await capabilities()
      return normalizeCapabilities(result)
    },
    async listObjects() {
      const items = await listObjects()
      return items.map(normalizeObjectSummary)
    },
    async listEvents() {
      const items = await listEvents()
      return items.map(normalizeEventItem)
    },
    async listAlarms() {
      const items = await listAlarms()
      return items.map(normalizeAlarmItem)
    },
    async listAlarmGroups() {
      const items = await listAlarmGroups()
      return items.map(normalizeAlarmGroup)
    },
    async listAlarmProcessingOptionsCached() {
      const fn = frontendBridge.ListAlarmProcessingOptionsCached
      if (!fn) return []
      const items = await fn()
      return items.map(normalizeAlarmProcessingOption)
    },
    async listResponseGroups() {
      const fn = frontendBridge.ListResponseGroups
      if (!fn) return []
      const items = await fn()
      return items.map(normalizeResponseGroup)
    },
    async getAlarmProcessingOptions(alarmID) {
      const items = await getAlarmProcessingOptions(alarmID)
      return items.map(normalizeAlarmProcessingOption)
    },
    async pickAlarm(alarmID, request) {
      await pickAlarm(alarmID, request)
    },
    async processAlarm(alarmID, request) {
      await processAlarm(alarmID, request)
    },
    async groupProcessAlarm(alarmID, user) {
      const fn = frontendBridge.GroupProcessAlarm
      if (!fn) throw new Error('GroupProcessAlarm не підтримується')
      await fn(alarmID, user)
    },
    async assignResponseGroup(alarmID, request) {
      const fn = frontendBridge.AssignResponseGroup
      if (!fn) throw new Error('AssignResponseGroup не підтримується')
      await fn(alarmID, request)
    },
    async notifyGroupArrived(alarmID) {
      const fn = frontendBridge.NotifyGroupArrived
      if (!fn) throw new Error('NotifyGroupArrived не підтримується')
      await fn(alarmID)
    },
    async cancelResponseGroup(alarmID) {
      const fn = frontendBridge.CancelResponseGroup
      if (!fn) throw new Error('CancelResponseGroup не підтримується')
      await fn(alarmID)
    },
    async standbyObject(objectID, durationMinutes, reason) {
      const fn = frontendBridge.StandbyObject
      if (!fn) throw new Error('StandbyObject не підтримується')
      await fn(objectID, durationMinutes, reason)
    },
    async listObjectEvents(objectID, offset, limit) {
      const page = await listObjectEvents(objectID, offset, limit)
      return normalizeEventPage(page)
    },
    async getObjectDetails(objectID) {
      const details = await getObjectDetails(objectID)
      return normalizeObjectDetails(details)
    },
    async getDBSettings() {
      const getDBSettings = settingsBridge?.GetDBSettings
      if (!getDBSettings) {
        throw new Error('Сервіс налаштувань Wails недоступний')
      }
      const settings = await getDBSettings()
      return normalizeDBSettings(settings)
    },
    async saveDBSettings(settings) {
      const saveDBSettings = settingsBridge?.SaveDBSettings
      if (!saveDBSettings) {
        throw new Error('Сервіс налаштувань Wails недоступний')
      }
      await saveDBSettings(settings)
    },
    async dialPhone(phone: string) {
      const base = import.meta.env.VITE_FRONTEND_API_BASE ?? '/api/frontend/v1'
      const res = await fetch(`${base}/dial`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone }),
      })
      if (!res.ok) throw new Error(`dial failed: HTTP ${res.status}`)
      return res.json() as Promise<{ callID: string }>
    },
    async hangupCall(callID: string) {
      const base = import.meta.env.VITE_FRONTEND_API_BASE ?? '/api/frontend/v1'
      await fetch(`${base}/dial/${encodeURIComponent(callID)}`, { method: 'DELETE' })
    },
    async getAMISettings() {
      const getSettings = settingsBridge?.GetDBSettings
      if (!getSettings) throw new Error('Сервіс налаштувань Wails недоступний')
      const s = await getSettings()
      const v = s as Record<string, unknown>
      return {
        enabled: Boolean(v.AMIEnabled),
        host:      String(v.AMIHost      || '127.0.0.1'),
        port:      Number(v.AMIPort      || 5038),
        username:  String(v.AMIUsername  || 'admin'),
        secret:    String(v.AMISecret    ?? ''),
        extension: String(v.AMIExtension || '100'),
        context:   String(v.AMIContext   || 'from-internal'),
      }
    },
    async saveAMISettings(settings) {
      const getSettings = settingsBridge?.GetDBSettings
      const saveSettings = settingsBridge?.SaveDBSettings
      if (!getSettings || !saveSettings) throw new Error('Сервіс налаштувань Wails недоступний')
      const current = await getSettings()
      await saveSettings({
        ...current,
        AMIEnabled:   settings.enabled,
        AMIHost:      settings.host,
        AMIPort:      settings.port,
        AMIUsername:  settings.username,
        AMISecret:    settings.secret,
        AMIExtension: settings.extension,
        AMIContext:   settings.context,
      } as typeof current)
    },
    async getAMIStatus() {
      const base = import.meta.env.VITE_FRONTEND_API_BASE ?? '/api/frontend/v1'
      try {
        const res = await fetch(`${base}/ami-status`)
        if (!res.ok) return { connected: false, enabled: false }
        return res.json() as Promise<{ connected: boolean; enabled: boolean }>
      } catch {
        return { connected: false, enabled: false }
      }
    },
  }
}

function resolveFrontendBridge(): WailsFrontendBridge | null {
  if (typeof window === 'undefined' || window.go == null) {
    return null
  }

  if (window.go.wailsbridge?.FrontendV1Service != null) {
    return window.go.wailsbridge.FrontendV1Service
  }

  if (window.go.main?.FrontendV1Service != null) {
    return window.go.main.FrontendV1Service
  }

  return null
}

function resolveSettingsBridge(): WailsOperatorSettingsBridge | null {
  if (typeof window === 'undefined' || window.go == null) {
    return null
  }

  if (window.go.main?.OperatorSettingsService != null) {
    return window.go.main.OperatorSettingsService
  }

  if (window.go.wailsbridge?.OperatorSettingsService != null) {
    return window.go.wailsbridge.OperatorSettingsService
  }

  return null
}
