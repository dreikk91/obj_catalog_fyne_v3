import type { FrontendClient } from './frontend-client'
import type {
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
  const listEvents = frontendBridge.ListEvents
  const listAlarms = frontendBridge.ListAlarms
  const listAlarmGroups = frontendBridge.ListAlarmGroups
  const getAlarmProcessingOptions = frontendBridge.GetAlarmProcessingOptions
  const pickAlarm = frontendBridge.PickAlarm
  const processAlarm = frontendBridge.ProcessAlarm
  const listObjectEvents = frontendBridge.ListObjectEvents
  const getObjectDetails = frontendBridge.GetObjectDetails

  if (!listObjects || !listEvents || !listAlarms || !listAlarmGroups || !getAlarmProcessingOptions || !pickAlarm || !processAlarm || !listObjectEvents || !getObjectDetails) {
    return null
  }

  return {
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
