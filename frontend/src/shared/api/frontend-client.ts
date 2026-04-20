import type {
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
} from './types'

export interface FrontendClient {
  listObjects(): Promise<FrontendObjectSummary[]>
  listEvents(): Promise<FrontendEventItem[]>
  listAlarms(): Promise<FrontendAlarmItem[]>
  listAlarmGroups(): Promise<FrontendAlarmGroup[]>
  getAlarmProcessingOptions(alarmID: number): Promise<FrontendAlarmProcessingOption[]>
  pickAlarm(alarmID: number, request: FrontendAlarmPickRequest): Promise<void>
  processAlarm(alarmID: number, request: FrontendAlarmProcessRequest): Promise<void>
  listObjectEvents(objectID: number, offset: number, limit: number): Promise<FrontendEventPage>
  getObjectDetails(objectID: number): Promise<FrontendObjectDetails>
  getDBSettings(): Promise<FrontendDBSettings>
  saveDBSettings(settings: FrontendDBSettings): Promise<void>
}
