export type FrontendSource = 'unknown' | 'bridge' | 'phoenix' | 'casl'
export type GuardStatus = 'unknown' | 'guarded' | 'disarmed'
export type ConnectionStatus = 'unknown' | 'online' | 'offline'
export type MonitoringStatus = 'unknown' | 'active' | 'blocked' | 'debug'
export type VisualSeverity = 'unknown' | 'normal' | 'info' | 'warning' | 'critical'

export type FrontendObjectSummary = {
  id: number
  source: FrontendSource
  nativeID: string
  displayNumber: string
  name: string
  address: string
  contractNumber: string
  phone: string
  statusCode: string
  statusText: string
  deviceType: string
  panelMark: string
  signalStrength: string
  sim1: string
  sim2: string
  lastTestTime: string
  lastMessageTime: string
  guardStatus: GuardStatus
  connectionStatus: ConnectionStatus
  monitoringStatus: MonitoringStatus
  hasAssignment: boolean
}

export type FrontendZone = {
  number: number
  name: string
  sensorType: string
  status: string
}

export type FrontendContact = {
  name: string
  position: string
  phone: string
  priority: number
}

export type FrontendEventItem = {
  id: number
  source: FrontendSource
  objectID: number
  objectNumber: string
  objectName: string
  time: string
  typeCode: string
  typeText: string
  zoneNumber: number
  details: string
  userName: string
  visualSeverity: VisualSeverity
}

export type FrontendAlarmItem = {
  id: number
  source: FrontendSource
  objectID: number
  objectNativeID: string
  objectNumber: string
  objectName: string
  address: string
  time: string
  typeCode: string
  typeText: string
  zoneNumber: number
  zoneName: string
  isProcessed: boolean
  processedBy: string
  processNote: string
  isInProgress: boolean
  inProgressBy: string
  isOwnedByMe: boolean
  canTakeOver: boolean
  canProcess: boolean
  details: string
  visualSeverity: VisualSeverity
}

export type FrontendAlarmGroup = {
  groupID: string
  source: FrontendSource
  objectID: number
  objectNativeID: string
  objectNumber: string
  objectName: string
  address: string
  alertLevel: number
  latestTime: string
  primary: FrontendAlarmItem
  items: FrontendAlarmItem[]
}

export type FrontendAlarmProcessingOption = {
  code: string
  label: string
}

export type FrontendAlarmProcessRequest = {
  user: string
  causeCode: string
  note: string
}

export type FrontendAlarmPickRequest = {
  user: string
}

export type FrontendObjectDetails = {
  summary: FrontendObjectSummary
  phones: string
  notes: string
  location: string
  launchDate: string
  zones: FrontendZone[]
  contacts: FrontendContact[]
  events: FrontendEventItem[]
}

export type FrontendEventPage = {
  items: FrontendEventItem[]
  totalCount: number
  hasMore: boolean
}

export type FrontendDBSettings = {
  firebirdEnabled: boolean
  firebirdUser: string
  firebirdPassword: string
  firebirdHost: string
  firebirdPort: string
  firebirdPath: string
  firebirdParams: string

  phoenixEnabled: boolean
  phoenixUser: string
  phoenixPassword: string
  phoenixHost: string
  phoenixPort: string
  phoenixInstance: string
  phoenixDatabase: string
  phoenixParams: string

  caslEnabled: boolean
  caslBaseURL: string
  caslToken: string
  caslEmail: string
  caslPass: string
  caslPultID: number

  mode: string
}
