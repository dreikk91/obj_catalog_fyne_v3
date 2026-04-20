import type {
  ConnectionStatus,
  FrontendAlarmItem,
  FrontendAlarmGroup,
  FrontendAlarmProcessRequest,
  FrontendAlarmPickRequest,
  FrontendAlarmProcessingOption,
  FrontendContact,
  FrontendDBSettings,
  FrontendEventPage,
  FrontendEventItem,
  FrontendObjectDetails,
  FrontendObjectSummary,
  FrontendResponseGroup,
  FrontendSource,
  FrontendZone,
  GuardStatus,
  MonitoringStatus,
  VisualSeverity,
} from './types'

export function normalizeObjectSummary(input: unknown): FrontendObjectSummary {
  const value = asRecord(input)
  return {
    id: asNumber(value.id ?? value.ID),
    source: asSource(value.source ?? value.Source),
    nativeID: asString(value.nativeID ?? value.NativeID),
    displayNumber: asString(value.displayNumber ?? value.DisplayNumber),
    name: asString(value.name ?? value.Name),
    address: asString(value.address ?? value.Address),
    contractNumber: asString(value.contractNumber ?? value.ContractNumber),
    phone: asString(value.phone ?? value.Phone),
    statusCode: asString(value.statusCode ?? value.StatusCode),
    statusText: asString(value.statusText ?? value.StatusText),
    deviceType: asString(value.deviceType ?? value.DeviceType),
    panelMark: asString(value.panelMark ?? value.PanelMark),
    signalStrength: asString(value.signalStrength ?? value.SignalStrength),
    sim1: asString(value.sim1 ?? value.SIM1),
    sim2: asString(value.sim2 ?? value.SIM2),
    lastTestTime: asDateText(value.lastTestTime ?? value.LastTestTime),
    lastMessageTime: asDateText(value.lastMessageTime ?? value.LastMessageTime),
    guardStatus: asGuardStatus(value.guardStatus ?? value.GuardStatus),
    connectionStatus: asConnectionStatus(value.connectionStatus ?? value.ConnectionStatus),
    monitoringStatus: asMonitoringStatus(value.monitoringStatus ?? value.MonitoringStatus),
    hasAssignment: asBoolean(value.hasAssignment ?? value.HasAssignment),
  }
}

export function normalizeAlarmItem(input: unknown): FrontendAlarmItem {
  const value = asRecord(input)
  return {
    id: asNumber(value.id ?? value.ID),
    source: asSource(value.source ?? value.Source),
    objectID: asNumber(value.objectID ?? value.ObjectID),
    objectNativeID: asString(value.objectNativeID ?? value.ObjectNativeID),
    objectNumber: asString(value.objectNumber ?? value.ObjectNumber),
    objectName: asString(value.objectName ?? value.ObjectName),
    address: asString(value.address ?? value.Address),
    time: asDateText(value.time ?? value.Time),
    typeCode: asString(value.typeCode ?? value.TypeCode),
    typeText: asString(value.typeText ?? value.TypeText),
    zoneNumber: asNumber(value.zoneNumber ?? value.ZoneNumber),
    zoneName: asString(value.zoneName ?? value.ZoneName),
    isProcessed: asBoolean(value.isProcessed ?? value.IsProcessed),
    processedBy: asString(value.processedBy ?? value.ProcessedBy),
    processNote: asString(value.processNote ?? value.ProcessNote),
    isInProgress: asBoolean(value.isInProgress ?? value.IsInProgress),
    inProgressBy: asString(value.inProgressBy ?? value.InProgressBy),
    isOwnedByMe: asBoolean(value.isOwnedByMe ?? value.IsOwnedByMe),
    canTakeOver: asBoolean(value.canTakeOver ?? value.CanTakeOver),
    canProcess: asBoolean(value.canProcess ?? value.CanProcess),
    details: asString(value.details ?? value.Details),
    visualSeverity: asVisualSeverity(value.visualSeverity ?? value.VisualSeverity),
  }
}

export function normalizeEventItem(input: unknown): FrontendEventItem {
  const value = asRecord(input)
  return {
    id: asNumber(value.id ?? value.ID),
    source: asSource(value.source ?? value.Source),
    objectID: asNumber(value.objectID ?? value.ObjectID),
    objectNumber: asString(value.objectNumber ?? value.ObjectNumber),
    objectName: asString(value.objectName ?? value.ObjectName),
    time: asDateText(value.time ?? value.Time),
    typeCode: asString(value.typeCode ?? value.TypeCode),
    typeText: asString(value.typeText ?? value.TypeText),
    zoneNumber: asNumber(value.zoneNumber ?? value.ZoneNumber),
    details: asString(value.details ?? value.Details),
    userName: asString(value.userName ?? value.UserName),
    visualSeverity: asVisualSeverity(value.visualSeverity ?? value.VisualSeverity),
  }
}

export function normalizeAlarmGroup(input: unknown): FrontendAlarmGroup {
  const value = asRecord(input)
  return {
    groupID: asString(value.groupID ?? value.GroupID),
    source: asSource(value.source ?? value.Source),
    objectID: asNumber(value.objectID ?? value.ObjectID),
    objectNativeID: asString(value.objectNativeID ?? value.ObjectNativeID),
    objectNumber: asString(value.objectNumber ?? value.ObjectNumber),
    objectName: asString(value.objectName ?? value.ObjectName),
    address: asString(value.address ?? value.Address),
    alertLevel: asNumber(value.alertLevel ?? value.AlertLevel),
    latestTime: asDateText(value.latestTime ?? value.LatestTime),
    primary: normalizeAlarmItem(value.primary ?? value.Primary),
    items: asArray(value.items ?? value.Items).map(normalizeAlarmItem),
  }
}

export function normalizeAlarmProcessingOption(input: unknown): FrontendAlarmProcessingOption {
  const value = asRecord(input)
  return {
    code: asString(value.code ?? value.Code),
    label: asString(value.label ?? value.Label),
  }
}

export function normalizeAlarmProcessRequest(input: unknown): FrontendAlarmProcessRequest {
  const value = asRecord(input)
  return {
    user: asString(value.user ?? value.User),
    causeCode: asString(value.causeCode ?? value.CauseCode),
    note: asString(value.note ?? value.Note),
  }
}

export function normalizeAlarmPickRequest(input: unknown): FrontendAlarmPickRequest {
  const value = asRecord(input)
  return {
    user: asString(value.user ?? value.User),
  }
}

export function normalizeEventPage(input: unknown): FrontendEventPage {
  const value = asRecord(input)
  return {
    items: asArray(value.items ?? value.Items).map(normalizeEventItem),
    totalCount: asNumber(value.totalCount ?? value.TotalCount),
    hasMore: asBoolean(value.hasMore ?? value.HasMore),
  }
}

export function normalizeObjectDetails(input: unknown): FrontendObjectDetails {
  const value = asRecord(input)
  const zones = asArray(value.zones ?? value.Zones).map(normalizeZone)
  const contacts = asArray(value.contacts ?? value.Contacts).map(normalizeContact)
  const events = asArray(value.events ?? value.Events).map(normalizeEventItem)

  return {
    summary: normalizeObjectSummary(value.summary ?? value.Summary),
    phones: asString(value.phones ?? value.Phones),
    notes: asString(value.notes ?? value.Notes),
    location: asString(value.location ?? value.Location),
    launchDate: asString(value.launchDate ?? value.LaunchDate),
    zones,
    contacts,
    events,
  }
}

export function normalizeDBSettings(input: unknown): FrontendDBSettings {
  const value = asRecord(input)
  return {
    firebirdEnabled: asBoolean(value.firebirdEnabled ?? value.FirebirdEnabled),
    firebirdUser: asString(value.firebirdUser ?? value.FirebirdUser),
    firebirdPassword: asString(value.firebirdPassword ?? value.FirebirdPassword),
    firebirdHost: asString(value.firebirdHost ?? value.FirebirdHost),
    firebirdPort: asString(value.firebirdPort ?? value.FirebirdPort),
    firebirdPath: asString(value.firebirdPath ?? value.FirebirdPath),
    firebirdParams: asString(value.firebirdParams ?? value.FirebirdParams),

    phoenixEnabled: asBoolean(value.phoenixEnabled ?? value.PhoenixEnabled),
    phoenixUser: asString(value.phoenixUser ?? value.PhoenixUser),
    phoenixPassword: asString(value.phoenixPassword ?? value.PhoenixPassword),
    phoenixHost: asString(value.phoenixHost ?? value.PhoenixHost),
    phoenixPort: asString(value.phoenixPort ?? value.PhoenixPort),
    phoenixInstance: asString(value.phoenixInstance ?? value.PhoenixInstance),
    phoenixDatabase: asString(value.phoenixDatabase ?? value.PhoenixDatabase),
    phoenixParams: asString(value.phoenixParams ?? value.PhoenixParams),

    caslEnabled: asBoolean(value.caslEnabled ?? value.CASLEnabled),
    caslBaseURL: asString(value.caslBaseURL ?? value.CASLBaseURL),
    caslToken: asString(value.caslToken ?? value.CASLToken),
    caslEmail: asString(value.caslEmail ?? value.CASLEmail),
    caslPass: asString(value.caslPass ?? value.CASLPass),
    caslPultID: asNumber(value.caslPultID ?? value.CASLPultID),

    mode: asString(value.mode ?? value.Mode),
  }
}

export function normalizeResponseGroup(input: unknown): FrontendResponseGroup {
  const value = asRecord(input)
  return {
    id: asString(value.id ?? value.ID),
    name: asString(value.name ?? value.Name),
    callsign: asString(value.callsign ?? value.Callsign),
    phone: asString(value.phone ?? value.Phone),
  }
}

function normalizeZone(input: unknown): FrontendZone {
  const value = asRecord(input)
  return {
    number: asNumber(value.number ?? value.Number),
    name: asString(value.name ?? value.Name),
    sensorType: asString(value.sensorType ?? value.SensorType),
    status: asString(value.status ?? value.Status),
    groupID: asString(value.groupID ?? value.GroupID),
    groupNumber: asNumber(value.groupNumber ?? value.GroupNumber),
    groupName: asString(value.groupName ?? value.GroupName),
    groupStateText: asString(value.groupStateText ?? value.GroupStateText),
  }
}

function normalizeContact(input: unknown): FrontendContact {
  const value = asRecord(input)
  return {
    name: asString(value.name ?? value.Name),
    position: asString(value.position ?? value.Position),
    phone: asString(value.phone ?? value.Phone),
    priority: asNumber(value.priority ?? value.Priority),
    groupID: asString(value.groupID ?? value.GroupID),
    groupNumber: asNumber(value.groupNumber ?? value.GroupNumber),
    groupName: asString(value.groupName ?? value.GroupName),
    groupStateText: asString(value.groupStateText ?? value.GroupStateText),
  }
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value == null || typeof value !== 'object') {
    return {}
  }
  return value as Record<string, unknown>
}

function asArray(value: unknown): unknown[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value
}

function asNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string') {
    const num = Number(value)
    if (Number.isFinite(num)) {
      return num
    }
  }
  return 0
}

function asBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    return normalized === 'true' || normalized === '1' || normalized === 'yes'
  }
  if (typeof value === 'number') {
    return value === 1
  }
  return false
}

function asString(value: unknown): string {
  if (typeof value === 'string') {
    return value
  }
  if (value == null) {
    return ''
  }
  return String(value)
}

function asDateText(value: unknown): string {
  if (value instanceof Date) {
    return value.toISOString()
  }
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return new Date(value).toISOString()
  }
  return ''
}

function asSource(value: unknown): FrontendSource {
  const source = asString(value).toLowerCase()
  if (source === 'bridge' || source === 'phoenix' || source === 'casl') {
    return source
  }
  return 'unknown'
}

function asGuardStatus(value: unknown): GuardStatus {
  const status = asString(value).toLowerCase()
  if (status === 'guarded' || status === 'disarmed') {
    return status
  }
  return 'unknown'
}

function asConnectionStatus(value: unknown): ConnectionStatus {
  const status = asString(value).toLowerCase()
  if (status === 'online' || status === 'offline') {
    return status
  }
  return 'unknown'
}

function asMonitoringStatus(value: unknown): MonitoringStatus {
  const status = asString(value).toLowerCase()
  if (status === 'active' || status === 'blocked' || status === 'debug') {
    return status
  }
  return 'unknown'
}

function asVisualSeverity(value: unknown): VisualSeverity {
  const level = asString(value).toLowerCase()
  if (level === 'normal' || level === 'info' || level === 'warning' || level === 'critical') {
    return level
  }
  return 'unknown'
}
