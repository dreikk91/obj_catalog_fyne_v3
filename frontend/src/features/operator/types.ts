import type { RefObject, UIEvent } from 'react'
import type { FrontendSource, VisualSeverity } from '../../shared/api/types'

export type ObjectRow = {
  id: number
  number: string
  group: string
  contract: string
  name: string
  address: string
  statusLabel: string
  statusKey: 'guarded' | 'unguarded' | 'alarm' | 'call'
  statusClass: 'chip-green' | 'chip-gray' | 'chip-red' | 'chip-orange'
  lastEventAt: string
  lastTestAt: string
  phone: string
  note: string
}

export type JournalRow = {
  rowID: string
  alarmID: number | null
  source: FrontendSource
  sortTimestampMs: number
  line: string
  objectID: number
  objectNumber: string
  code: string
  typeText: string
  date: string
  time: string
  group: string
  zone: string
  objectName: string
  state: string
  details: string
  alarm: boolean
  processed: boolean
  inProgress: boolean
  inProgressBy: string
  inProgressByMe: boolean
  canTakeOver: boolean
  canProcess: boolean
  responseGroupID: string
  responseGroupDispatched: boolean
  responseGroupArrived: boolean
  severity: VisualSeverity
}

export type TableColumnMeta = {
  fluid?: boolean
  minWidth?: number
}

export type UnprocessedAlarmGroup = {
  groupID: string
  objectID: number
  alertLevel: number
  anchorRow: JournalRow
  rows: JournalRow[]
  latestSortTimestampMs: number
}

export type UnprocessedRowMeta = {
  groupID: string
  isParent: boolean
  isChild: boolean
  groupSize: number
  memberRowIDs: string[]
  childIndex?: number
  isLastChild?: boolean
}

export type VirtualRowsOptions = {
  rowHeight: number
  initialCount?: number
  step?: number
  overscanRows?: number
}

export type VirtualRowsSlice<T> = {
  containerRef: RefObject<HTMLDivElement | null>
  onScroll: (event: UIEvent<HTMLDivElement>) => void
  visibleRows: T[]
  startIndex: number
  topPaddingPx: number
  bottomPaddingPx: number
  loadedCount: number
  totalCount: number
}
