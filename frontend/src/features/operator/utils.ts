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
  const isAlarm = item.visualSeverity === 'critical' || isAlarmTypeText(typeText)
  const severity = resolveJournalSeverity(item.visualSeverity, isAlarm, typeText)
  return {
    rowID: `event-${item.id}-${item.time}`,
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
    severity,
  }
}

export function toAlarmRow(item: FrontendAlarmItem): JournalRow {
  const date = parseDate(item.time)
  const sortTimestampMs = date.getTime()
  const typeText = item.typeText || 'Тривога'
  const severity = resolveJournalSeverity(item.visualSeverity, true, typeText)
  const zone = item.zoneName.trim() || (item.zoneNumber > 0 ? String(item.zoneNumber) : '—')
  return {
    rowID: `alarm-${item.id}-${item.time}`,
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
    state: item.isProcessed ? 'Оброблено' : item.isInProgress ? 'В роботі' : 'Нова',
    details: buildAlarmDetailsText(item),
    alarm: true,
    processed: item.isProcessed,
    inProgress: item.isInProgress,
    inProgressBy: item.inProgressBy,
    inProgressByMe: item.isOwnedByMe,
    canTakeOver: item.canTakeOver,
    canProcess: item.canProcess,
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
    return left.rowID < right.rowID ? 1 : -1
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

export function mergeUnprocessedGroupsByObject(groups: UnprocessedAlarmGroup[]): UnprocessedAlarmGroup[] {
  const byObject = new Map<number, UnprocessedAlarmGroup>()
  for (const group of groups) {
    const existing = byObject.get(group.objectID)
    if (existing == null) {
      byObject.set(group.objectID, { ...group, rows: [...group.rows] })
      continue
    }
    const allRows = [...existing.rows, ...group.rows]
    const anchor = group.alertLevel > existing.alertLevel ? group.anchorRow : existing.anchorRow
    byObject.set(group.objectID, {
      groupID: existing.groupID,
      objectID: existing.objectID,
      alertLevel: Math.max(existing.alertLevel, group.alertLevel),
      anchorRow: anchor,
      rows: allRows,
      latestSortTimestampMs: Math.max(existing.latestSortTimestampMs, group.latestSortTimestampMs),
    })
  }
  return Array.from(byObject.values())
}

export function flattenUnprocessedAlarmGroups(
  groups: UnprocessedAlarmGroup[],
  expandedGroups: Record<string, boolean>,
): JournalRow[] {
  const rows: JournalRow[] = []
  for (const group of groups) {
    rows.push(group.anchorRow)
    if (!expandedGroups[group.groupID] || group.rows.length <= 1) {
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
    const memberRowIDs = group.rows.map((row) => row.rowID)
    map.set(group.anchorRow.rowID, {
      groupID: group.groupID,
      isParent: true,
      isChild: false,
      groupSize: group.rows.length,
      memberRowIDs,
    })

    if (!expandedGroups[group.groupID] || group.rows.length <= 1) {
      continue
    }

    for (const childRow of group.rows) {
      if (childRow.rowID === group.anchorRow.rowID) {
        continue
      }
      map.set(childRow.rowID, {
        groupID: group.groupID,
        isParent: false,
        isChild: true,
        groupSize: group.rows.length,
        memberRowIDs: [],
      })
    }
  }
  return map
}

export function resolveJournalRowClass(row: JournalRow, isSelected: boolean, forceAlarm = false): string {
  const classes: string[] = []
  if (isSelected) {
    classes.push('selected')
  }

  if (forceAlarm || row.alarm || row.severity === 'critical') {
    classes.push('alarm')
  } else if (row.severity === 'warning') {
    classes.push('evt-warning')
  } else if (row.severity === 'info') {
    classes.push('evt-info')
  }

  return classes.join(' ')
}

export function resolveJournalTypeClass(row: JournalRow): string {
  if (row.alarm || row.severity === 'critical') {
    return 'red'
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
  if (row.inProgress) {
    return row.inProgressByMe ? 'chip-info' : 'chip-orange'
  }
  if (row.alarm || row.severity === 'critical') {
    return 'chip-red'
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

function isAlarmTypeText(value: string): boolean {
  const text = value.toLowerCase()
  return text.includes('трив') || text.includes('пожеж') || text.includes('alarm') || text.includes('fire')
}

function resolveJournalSeverity(
  visualSeverity: VisualSeverity,
  alarm: boolean,
  typeText: string,
): VisualSeverity {
  if (visualSeverity !== 'unknown') {
    return visualSeverity
  }
  if (alarm || isAlarmTypeText(typeText)) {
    return 'critical'
  }

  const text = typeText.toLowerCase()
  if (text.includes('несправ') || text.includes('fault') || text.includes('помил')) {
    return 'warning'
  }
  if (text.includes('інф') || text.includes('info') || text.includes('повідом')) {
    return 'info'
  }

  return 'unknown'
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
