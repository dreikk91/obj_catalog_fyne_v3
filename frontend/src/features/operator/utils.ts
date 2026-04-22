import type { FrontendAlarmGroup, FrontendAlarmItem, FrontendEventItem, FrontendObjectSummary, VisualSeverity } from '../../shared/api/types'
import { sourceLabel } from '../../shared/ui/source'
import type { JournalRow, ObjectRow, UnprocessedAlarmGroup, UnprocessedRowMeta } from './types'

export function toObjectRow(item: FrontendObjectSummary, unprocessedAlarmCount: number): ObjectRow {
  const status = resolveStatus(item, unprocessedAlarmCount)
  return {
    id: item.id,
    number: item.displayNumber || String(item.id),
    group: resolveObjectGroup(item),
    contract: item.contractNumber.trim() || '—',
    name: item.name || '—',
    address: item.address || '—',
    statusLabel: status.label,
    statusKey: status.key,
    statusClass: status.className,
    lastEventAt: toDateTimeText(item.lastMessageTime),
    lastTestAt: toDateTimeText(item.lastTestTime),
    phone: item.phone || '—',
    note: item.statusText || '—',
  }
}

export function toArchiveRow(item: FrontendEventItem): JournalRow {
  const date = parseDate(item.time)
  const sortTimestampMs = date.getTime()
  const typeText = item.typeText || 'Подія'
  const isAlarm = item.visualSeverity === 'critical' || isCriticalCode(item.typeCode)
  const severity = resolveJournalSeverity(item.visualSeverity, item.typeCode)
  const ts = Number.isFinite(sortTimestampMs) ? sortTimestampMs : 0
  return {
    rowID: `event-${item.source}-${item.id}-${ts}`,
    alarmID: null,
    source: item.source,
    sortTimestampMs: Number.isFinite(sortTimestampMs) ? sortTimestampMs : 0,
    line: sourceLabel(item.source),
    objectID: item.objectID,
    objectNumber: item.objectNumber || String(item.objectID),
    code: item.typeCode || '—',
    typeText,
    date: formatDate(date),
    time: formatTime(date),
    group: resolveJournalGroup(item.details, item.zoneNumber),
    zone: item.zoneNumber > 0 ? String(item.zoneNumber) : '—',
    objectName: item.objectName || '—',
    state: isAlarm ? 'Оброблено' : 'Архів',
    details: item.details || '—',
    alarm: isAlarm,
    processed: true,
    inProgress: false,
    inProgressBy: '',
    inProgressByMe: false,
    canTakeOver: false,
    canProcess: false,
    responseGroupID: '',
    responseGroupDispatched: false,
    responseGroupArrived: false,
    severity,
  }
}

export function toAlarmRow(item: FrontendAlarmItem): JournalRow {
  const date = parseDate(item.time)
  const sortTimestampMs = date.getTime()
  const typeText = item.typeText || 'Тривога'
  const severity = resolveJournalSeverity(item.visualSeverity, item.typeCode)
  const zone = item.zoneName.trim() || (item.zoneNumber > 0 ? String(item.zoneNumber) : '—')
  return {
    rowID: `alarm-${item.source}-${item.id}-${sortTimestampMs}-${item.typeCode}-${item.zoneNumber}`,
    alarmID: item.id,
    source: item.source,
    sortTimestampMs: Number.isFinite(sortTimestampMs) ? sortTimestampMs : 0,
    line: sourceLabel(item.source),
    objectID: item.objectID,
    objectNumber: item.objectNumber || String(item.objectID),
    code: item.typeCode || '—',
    typeText,
    date: formatDate(date),
    time: formatTime(date),
    group: resolveJournalGroup(item.details, item.zoneNumber),
    zone,
    objectName: item.objectName || '—',
    state: item.isProcessed
      ? 'Оброблено'
      : item.isResponseGroupArrived
        ? 'МГР прибула'
        : item.isResponseGroupDispatched
          ? 'МГР вислана'
          : item.isInProgress
            ? 'Прийнято'
            : 'Нова',
    details: buildAlarmDetailsText(item),
    alarm: true,
    processed: item.isProcessed,
    inProgress: item.isInProgress,
    inProgressBy: item.inProgressBy,
    inProgressByMe: item.isOwnedByMe,
    canTakeOver: item.canTakeOver,
    canProcess: item.canProcess,
    responseGroupID: item.responseGroupID,
    responseGroupDispatched: item.isResponseGroupDispatched,
    responseGroupArrived: item.isResponseGroupArrived,
    severity,
  }
}

export function sliceRecentEvents(items: FrontendEventItem[], limit: number): FrontendEventItem[] {
  if (items.length <= limit) {
    return [...items]
  }
  return [...items]
    .sort((left, right) => compareEventItemsDesc(left, right))
    .slice(0, limit)
}

export function mergeRecentEvents(current: FrontendEventItem[], incoming: FrontendEventItem[]): FrontendEventItem[] {
  if (incoming.length === 0) {
    return current
  }

  const merged = new Map<string, FrontendEventItem>()
  for (const item of current) {
    merged.set(getEventIdentity(item), item)
  }
  for (const item of incoming) {
    merged.set(getEventIdentity(item), item)
  }

  return [...merged.values()].sort((left, right) => compareEventItemsDesc(left, right))
}

export function compareEventItemsDesc(left: FrontendEventItem, right: FrontendEventItem): number {
  const leftTime = parseDate(left.time).getTime()
  const rightTime = parseDate(right.time).getTime()
  if (leftTime === rightTime) {
    return right.id - left.id
  }
  return rightTime - leftTime
}

export function sortJournalRowsDesc(left: JournalRow, right: JournalRow): number {
  if (left.sortTimestampMs === right.sortTimestampMs) {
    // Використовуємо числовий порівняльник для рядків, щоб '10' було більше ніж '2'
    return right.rowID.localeCompare(left.rowID, undefined, { numeric: true })
  }
  return right.sortTimestampMs - left.sortTimestampMs
}

export function toUnprocessedAlarmGroup(group: FrontendAlarmGroup): UnprocessedAlarmGroup {
  const rows = group.items.map(toAlarmRow).sort(sortJournalRowsDesc)
  const anchorRow = toAlarmRow(group.primary)
  return {
    groupID: group.groupID,
    objectID: group.objectID,
    alertLevel: group.alertLevel,
    anchorRow,
    rows,
    latestSortTimestampMs: parseDate(group.latestTime).getTime(),
  }
}

export function getSeverityPriority(severity: string): number {
  switch (severity) {
    case 'critical':
      return 100
    case 'fault':
      return 80
    case 'warning':
      return 60
    case 'info':
      return 40
    case 'normal':
      return 20
    default:
      return 0
  }
}

export function isPanicAlarmRow(row: JournalRow): boolean {
  if (!row) return false
  return row.code === 'panic' || row.typeText === 'PANIC_ALARM' || row.code === 'PANIC_ALARM'
}

export function mergeUnprocessedGroupsByObject(groups: UnprocessedAlarmGroup[]): UnprocessedAlarmGroup[] {
  const byKey = new Map<string, UnprocessedAlarmGroup>()
  
  for (const group of groups) {
    const shouldMergeAsTree = supportsTreeGroupedUnprocessedGroup(group)
    const key = shouldMergeAsTree ? `${group.anchorRow.source}:${group.objectID}` : group.groupID
    const stableGroupID = shouldMergeAsTree ? `group-${group.anchorRow.source}-obj-${group.objectID}` : group.groupID
    
    const existing = byKey.get(key)
    if (existing == null) {
      byKey.set(key, { ...group, groupID: stableGroupID, rows: [...group.rows] })
      continue
    }
    
    const rowMap = new Map<string, JournalRow>()
    for (const row of [...existing.rows, ...group.rows]) {
      rowMap.set(row.rowID, row)
    }
    const allRows = Array.from(rowMap.values()).sort(sortJournalRowsDesc)
    const existingPrio = getSeverityPriority(existing.anchorRow.severity)
    const groupPrio = getSeverityPriority(group.anchorRow.severity)
    
    const groupAnchorIsBetter =
      groupPrio > existingPrio ||
      (groupPrio === existingPrio && group.latestSortTimestampMs > existing.latestSortTimestampMs)
      
    const anchor = groupAnchorIsBetter ? group.anchorRow : existing.anchorRow

    byKey.set(key, {
      groupID: stableGroupID,
      objectID: existing.objectID,
      alertLevel: Math.max(existing.alertLevel, group.alertLevel),
      anchorRow: anchor,
      rows: allRows,
      latestSortTimestampMs: Math.max(existing.latestSortTimestampMs, group.latestSortTimestampMs),
    })
  }
  
  return Array.from(byKey.values()).sort((left, right) => {
    const prioLeft = getSeverityPriority(left.anchorRow.severity)
    const prioRight = getSeverityPriority(right.anchorRow.severity)
    if (prioLeft !== prioRight) {
      return prioRight - prioLeft
    }
    return right.latestSortTimestampMs - left.latestSortTimestampMs
  })
}

export function flattenUnprocessedAlarmGroups(
  groups: UnprocessedAlarmGroup[],
  expandedGroups: Record<string, boolean>,
): JournalRow[] {
  const rows: JournalRow[] = []
  for (const group of groups) {
    rows.push(group.anchorRow)
    if (!isTreeGroupedUnprocessedGroup(group) || !expandedGroups[group.groupID]) {
      continue
    }

    const expandedRows = [...group.rows].sort(sortJournalRowsDesc).filter((row) => row.rowID !== group.anchorRow.rowID)
    rows.push(...expandedRows)
  }
  return rows
}

export function buildUnprocessedRowMeta(
  groups: UnprocessedAlarmGroup[],
  expandedGroups: Record<string, boolean>,
): Map<string, UnprocessedRowMeta> {
  const map = new Map<string, UnprocessedRowMeta>()
  for (const group of groups) {
    if (!isTreeGroupedUnprocessedGroup(group)) {
      continue
    }

    const memberRowIDs = group.rows.map((row) => row.rowID)
    map.set(group.anchorRow.rowID, {
      groupID: group.groupID,
      isParent: true,
      isChild: false,
      groupSize: group.rows.length,
      memberRowIDs,
    })

    if (!expandedGroups[group.groupID]) {
      continue
    }

    const childRows = group.rows
      .filter((childRow) => childRow.rowID !== group.anchorRow.rowID)
      .sort(sortJournalRowsDesc)

    childRows.forEach((childRow, index) => {
      if (childRow.rowID === group.anchorRow.rowID) {
        return
      }
      map.set(childRow.rowID, {
        groupID: group.groupID,
        isParent: false,
        isChild: true,
        groupSize: group.rows.length,
        memberRowIDs: [],
        childIndex: index,
        isLastChild: index === childRows.length - 1,
      })
    })
  }
  return map
}

export function isTreeGroupedUnprocessedGroup(group: UnprocessedAlarmGroup): boolean {
  return supportsTreeGroupedUnprocessedGroup(group) && group.rows.length > 1
}

function supportsTreeGroupedUnprocessedGroup(group: UnprocessedAlarmGroup): boolean {
  return group.anchorRow.source === 'casl' || group.anchorRow.source === 'phoenix'
}

export function resolveJournalRowClass(row: JournalRow, isSelected: boolean, forceAlarm = false): string {
  const classes: string[] = []
  if (isSelected) {
    classes.push('selected')
  }

  if (!row.inProgress) {
    if (row.severity === 'fault') {
      classes.push('evt-fault')
    } else if (row.severity === 'warning') {
      classes.push('evt-warning')
    } else if (forceAlarm || row.alarm || row.severity === 'critical') {
      classes.push('alarm')
    } else if (row.severity === 'info') {
      classes.push('evt-info')
    }
  }

  return classes.join(' ')
}

export function resolveJournalTypeClass(row: JournalRow): string {
  if (row.alarm || row.severity === 'critical') {
    return 'red'
  }
  if (row.severity === 'fault') {
    return 'evt-type-fault'
  }
  if (row.severity === 'warning') {
    return 'evt-type-warning'
  }
  if (row.severity === 'info') {
    return 'evt-type-info'
  }
  if (row.severity === 'normal') {
    return 'evt-type-normal'
  }
  return ''
}

export function resolveJournalStateChipClass(row: JournalRow): string {
  if (row.responseGroupArrived) {
    return 'chip-info'
  }
  if (row.responseGroupDispatched) {
    return 'chip-orange'
  }
  if (row.inProgress) {
    return row.inProgressByMe ? 'chip-info' : 'chip-orange'
  }
  if (row.alarm || row.severity === 'critical') {
    return 'chip-red'
  }
  if (row.severity === 'fault') {
    return 'chip-yellow'
  }
  if (row.severity === 'warning') {
    return 'chip-orange'
  }
  if (row.severity === 'info') {
    return 'chip-info'
  }
  if (row.severity === 'normal') {
    return 'chip-green'
  }
  return 'chip-gray'
}

export function resolveJournalIndicatorColor(row: JournalRow): string {
  if (row.alarm || row.severity === 'critical') {
    return 'var(--ac5)'
  }
  if (row.severity === 'fault') {
    return '#facc15'
  }
  if (row.severity === 'warning') {
    return 'var(--ac4)'
  }
  if (row.severity === 'info') {
    return 'var(--ac2)'
  }
  if (row.severity === 'normal') {
    return 'var(--ac)'
  }
  return 'var(--tx2)'
}

export function resolveObjectIndicatorColor(status: ObjectRow['statusKey']): string {
  switch (status) {
    case 'guarded':
      return 'var(--ac)'
    case 'alarm':
      return 'var(--ac5)'
    case 'call':
      return 'var(--ac4)'
    default:
      return 'var(--tx2)'
  }
}

export function parseDate(raw: string): Date {
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return new Date(0)
  }
  return parsed
}

export function toDateTimeText(raw: string | undefined): string {
  if (!raw || raw.trim() === '') {
    return '—'
  }
  const date = parseDate(raw)
  return `${formatDate(date)} ${formatTime(date)}`
}

export function formatDate(date: Date): string {
  return `${pad2(date.getUTCDate())}.${pad2(date.getUTCMonth() + 1)}.${date.getUTCFullYear()}`
}

export function formatTime(date: Date): string {
  return `${pad2(date.getUTCHours())}:${pad2(date.getUTCMinutes())}:${pad2(date.getUTCSeconds())}`
}

export function formatClock(date: Date): string {
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}`
}

export function pad2(value: number): string {
  return String(value).padStart(2, '0')
}

function resolveObjectGroup(item: FrontendObjectSummary): string {
  const panelMark = item.panelMark.trim()
  if (panelMark !== '') {
    return panelMark
  }
  return '—'
}

function resolveStatus(
  item: FrontendObjectSummary,
  unprocessedAlarmCount: number,
): { label: string; key: ObjectRow['statusKey']; className: ObjectRow['statusClass'] } {
  if (unprocessedAlarmCount > 0) {
    return { label: 'ТРИВОГА', key: 'alarm', className: 'chip-red' }
  }
  if (item.guardStatus === 'disarmed') {
    return { label: 'БЕЗ ОХОРОНИ', key: 'unguarded', className: 'chip-gray' }
  }
  if (item.connectionStatus === 'offline' || item.monitoringStatus === 'blocked') {
    return { label: 'НА ПРОЗВОНІ', key: 'call', className: 'chip-orange' }
  }
  return { label: 'ПІД ОХОРОНОЮ', key: 'guarded', className: 'chip-green' }
}

function resolveJournalGroup(details: string, fallbackZoneNumber: number): string {
  const match = details.match(/груп[аеи]\s*([0-9]+)/iu)
  if (match?.[1]) {
    return match[1]
  }
  if (fallbackZoneNumber > 0) {
    return String(fallbackZoneNumber)
  }
  return '—'
}

function isCriticalCode(code: string): boolean {
  const c = code.toLowerCase()
  return (
    c === 'fire' ||
    c === 'burglary' ||
    c === 'panic' ||
    c === 'medical' ||
    c === 'gas' ||
    c === 'tamper' ||
    c === 'fault' ||
    c === 'offline' ||
    c === 'alarm_notification' ||
    c === 'device_blocked'
  )
}

function resolveJournalSeverity(
  visualSeverity: VisualSeverity,
  typeCode: string,
): VisualSeverity {
  if (visualSeverity !== 'unknown') {
    return visualSeverity
  }
  
  if (isCriticalCode(typeCode)) {
    // Some critical codes might be visually represented as faults instead of red alarms
    if (typeCode.toLowerCase() === 'fault') {
      return 'fault'
    }
    return 'critical'
  }

  const code = typeCode.toLowerCase()
  if (
    code === 'power_fail' ||
    code === 'batt_low' ||
    code === 'manager_assigned' ||
    code === 'manager_canceled' ||
    code === 'service'
  ) {
    return 'warning'
  }

  if (
    code === 'test' ||
    code === 'notification' ||
    code === 'operator_action' ||
    code === 'manager_arrived' ||
    code === 'alarm_finished' ||
    code === 'device_unblocked' ||
    code === 'system'
  ) {
    return 'info'
  }

  if (
    code === 'restore' ||
    code === 'arm' ||
    code === 'disarm' ||
    code === 'power_ok' ||
    code === 'online'
  ) {
    return 'normal'
  }

  return 'warning'
}


function buildAlarmDetailsText(item: FrontendAlarmItem): string {
  const details = item.details || '—'
  if (!item.isInProgress) {
    return details
  }

  if (item.isOwnedByMe) {
    return `В роботі у вас. ${details}`
  }

  const assignee = item.inProgressBy.trim() || 'інший оператор'
  return `В роботі: ${assignee}. ${details}`
}

function getEventIdentity(item: FrontendEventItem): string {
  return `${item.id}|${item.objectID}|${item.time}|${item.typeCode}`
}
