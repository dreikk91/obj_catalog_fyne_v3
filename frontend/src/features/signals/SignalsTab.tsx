import { useEffect, useRef, useState } from 'react'
import type { PointerEvent as ReactPointerEvent } from 'react'
import type { FrontendObjectDetails } from '../../shared/api/types'
import type { BottomTab, InnerTab } from '../../shared/state/ui-store'
import type { JournalRow, ObjectRow, UnprocessedAlarmGroup, UnprocessedRowMeta } from '../operator/types'
import { ContactsSidebar } from './ContactsSidebar'
import { BottomEventTables } from './BottomEventTables'
import { InnerTabs } from './InnerTabs'
import { ObjectInfoBar } from './ObjectInfoBar'

const DEFAULT_BOTTOM_HEIGHT = 260
const MIN_BOTTOM_HEIGHT = 160
const MIN_MID_HEIGHT = 180

type SignalsTabProps = {
  innerTab: InnerTab
  bottomTab: BottomTab
  selectedSignalRow: JournalRow | null
  selectedObjectRow: ObjectRow | null
  objectDetails: FrontendObjectDetails | null
  unprocessedAlarmGroups: UnprocessedAlarmGroup[]
  journalArchiveRows: JournalRow[]
  unprocessedFlatRows: JournalRow[]
  unprocessedRowMetaByID: Map<string, UnprocessedRowMeta>
  expandedUnprocessedGroups: Record<string, boolean>
  showAllAlarms: boolean
  selectedSignalRowID: string | null
  onSelectInnerTab: (tab: InnerTab) => void
  onSelectBottomTab: (tab: BottomTab) => void
  onToggleGroup: (groupID: string) => void
  onToggleShowAll: () => void
  onSelectSignalRow: (row: JournalRow) => void
  onOpenEventModal: (row: JournalRow) => void
  onOpenCardModal: (row: JournalRow) => void
  isInWorkflow: boolean
  groupDispatched: boolean
  groupArrived: boolean
  workflowBusy: boolean
  onPickAlarm: () => void
  onStandby: () => void
  onCancelAlarm: () => void
  onDispatchGroup: () => void
  onGroupAction: () => void
  onOpenProcessAlarm: () => void
}

export function SignalsTab(props: SignalsTabProps) {
  const layoutRef = useRef<HTMLDivElement | null>(null)
  const resizeStateRef = useRef<{ startY: number; startHeight: number } | null>(null)
  const [bottomHeight, setBottomHeight] = useState(DEFAULT_BOTTOM_HEIGHT)
  const [isResizing, setIsResizing] = useState(false)

  useEffect(() => {
    if (!isResizing) {
      return
    }

    const handlePointerMove = (event: PointerEvent) => {
      const layout = layoutRef.current
      const resizeState = resizeStateRef.current
      if (layout == null || resizeState == null) {
        return
      }

      const delta = resizeState.startY - event.clientY
      const nextHeight = resizeState.startHeight + delta
      const maxHeight = Math.max(MIN_BOTTOM_HEIGHT, layout.clientHeight - MIN_MID_HEIGHT)
      setBottomHeight(Math.min(maxHeight, Math.max(MIN_BOTTOM_HEIGHT, nextHeight)))
    }

    const handlePointerUp = () => {
      resizeStateRef.current = null
      setIsResizing(false)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    window.addEventListener('pointercancel', handlePointerUp)

    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
      window.removeEventListener('pointercancel', handlePointerUp)
    }
  }, [isResizing])

  useEffect(() => {
    return () => {
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
  }, [])

  const handleResizeStart = (event: ReactPointerEvent<HTMLDivElement>) => {
    resizeStateRef.current = {
      startY: event.clientY,
      startHeight: bottomHeight,
    }
    setIsResizing(true)
    document.body.style.cursor = 'row-resize'
    document.body.style.userSelect = 'none'
  }

  return (
    <div className="ps-layout" ref={layoutRef}>
      <ObjectInfoBar
        selectedSignalRow={props.selectedSignalRow}
        selectedObjectRow={props.selectedObjectRow}
        objectDetails={props.objectDetails}
      />

      <div className="ps-mid">
        <ContactsSidebar objectDetails={props.objectDetails} />
        <InnerTabs
          innerTab={props.innerTab}
          onSelectTab={props.onSelectInnerTab}
          selectedObjectRow={props.selectedObjectRow}
          objectDetails={props.objectDetails}
        />
      </div>

      <div
        className={isResizing ? 'signals-splitter is-resizing' : 'signals-splitter'}
        onPointerDown={handleResizeStart}
        role="separator"
        aria-orientation="horizontal"
        aria-label="Зміна висоти журналу подій"
      />

      <BottomEventTables
        height={bottomHeight}
        isResizing={isResizing}
        bottomTab={props.bottomTab}
        onSelectBottomTab={props.onSelectBottomTab}
        unprocessedAlarmGroups={props.unprocessedAlarmGroups}
        journalArchiveRows={props.journalArchiveRows}
        unprocessedFlatRows={props.unprocessedFlatRows}
        unprocessedRowMetaByID={props.unprocessedRowMetaByID}
        expandedUnprocessedGroups={props.expandedUnprocessedGroups}
        showAllAlarms={props.showAllAlarms}
        onToggleGroup={props.onToggleGroup}
        onToggleShowAll={props.onToggleShowAll}
        selectedSignalRowID={props.selectedSignalRowID}
        onSelectSignalRow={props.onSelectSignalRow}
        onOpenEventModal={props.onOpenEventModal}
        onOpenCardModal={props.onOpenCardModal}
        isInWorkflow={props.isInWorkflow}
        groupDispatched={props.groupDispatched}
        groupArrived={props.groupArrived}
        workflowBusy={props.workflowBusy}
        onPickAlarm={props.onPickAlarm}
        onStandby={props.onStandby}
        onCancelAlarm={props.onCancelAlarm}
        onDispatchGroup={props.onDispatchGroup}
        onGroupAction={props.onGroupAction}
        onOpenProcessAlarm={props.onOpenProcessAlarm}
      />
    </div>
  )
}
