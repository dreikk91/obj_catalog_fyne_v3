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

export interface FrontendClient {
  capabilities(): Promise<FrontendCapabilities>
  listObjects(): Promise<FrontendObjectSummary[]>
  listEvents(): Promise<FrontendEventItem[]>
  listAlarms(): Promise<FrontendAlarmItem[]>
  listAlarmGroups(): Promise<FrontendAlarmGroup[]>
  listAlarmProcessingOptionsCached(): Promise<FrontendAlarmProcessingOption[]>
  listResponseGroups(): Promise<FrontendResponseGroup[]>
  getAlarmProcessingOptions(alarmID: number): Promise<FrontendAlarmProcessingOption[]>
  pickAlarm(alarmID: number, request: FrontendAlarmPickRequest): Promise<void>
  processAlarm(alarmID: number, request: FrontendAlarmProcessRequest): Promise<void>
  groupProcessAlarm(alarmID: number, user: string): Promise<void>
  assignResponseGroup(alarmID: number, request: FrontendAlarmGroupActionRequest): Promise<void>
  notifyGroupArrived(alarmID: number): Promise<void>
  cancelResponseGroup(alarmID: number): Promise<void>
  standbyObject(objectID: number, durationMinutes: number, reason: string): Promise<void>
  listObjectEvents(objectID: number, offset: number, limit: number): Promise<FrontendEventPage>
  getObjectDetails(objectID: number): Promise<FrontendObjectDetails>
  getDBSettings(): Promise<FrontendDBSettings>
  saveDBSettings(settings: FrontendDBSettings): Promise<void>
}
