import { useCallback, useEffect, useMemo, useState, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useShallow } from 'zustand/react/shallow'
import { resolveFrontendClient } from '../shared/api/client'
import type {
  FrontendDBSettings,
  FrontendAlarmProcessingOption,
  FrontendResponseGroup,
  FrontendSourceCapability,
} from '../shared/api/types'
import { useOperatorUIStore } from '../shared/state/ui-store'
import { useThemeStore } from '../shared/state/theme-store'
import { useJournalStream } from '../hooks/useJournalStream'
import { useObjectEventsFeed } from '../hooks/useObjectEventsFeed'
import { CardModal } from '../features/modal/CardModal'
import { EventModal } from '../features/modal/EventModal'
import { AlarmProcessingModal } from '../features/modal/AlarmProcessingModal'
import { DispatchGroupModal } from '../features/modal/DispatchGroupModal'
import { StandbyModal } from '../features/modal/StandbyModal'
import { SettingsModal } from '../features/modal/SettingsModal'
import { ObjectsTab } from '../features/objects/ObjectsTab'
import { SignalsTab } from '../features/signals/SignalsTab'
import { RECENT_JOURNAL_WINDOW_MS } from '../features/operator/constants'
import {
  buildUnprocessedRowMeta,
  flattenUnprocessedAlarmGroups,
  formatClock,
  mergeUnprocessedGroupsByObject,
  sortJournalRowsDesc,
  toAlarmRow,
  toArchiveRow,
  toObjectRow,
  toUnprocessedAlarmGroup,
} from '../features/operator/utils'
import './app.css'

const api = resolveFrontendClient()
const OPERATOR_NAME = 'Підлипний А.М'

export function App() {
  const themeMode = useThemeStore((state) => state.themeMode)
  const setThemeMode = useThemeStore((state) => state.setThemeMode)
  const {
    mainTab,
    setMainTab,
    innerTab,
    setInnerTab,
    bottomTab,
    setBottomTab,
    statusFilter,
    setStatusFilter,
    searchValue,
    setSearchValue,
    selectedObjectID,
    setSelectedObjectID,
    selectedSignalRowID,
    setSelectedSignalRowID,
    isCardModalOpen,
    setIsCardModalOpen,
    isEventModalOpen,
    setIsEventModalOpen,
    cardModalTab,
    setCardModalTab,
    eventModalTab,
    setEventModalTab,
    eventModalRowID,
    setEventModalRowID,
  } = useOperatorUIStore(
    useShallow((state) => ({
      mainTab: state.mainTab,
      setMainTab: state.setMainTab,
      innerTab: state.innerTab,
      setInnerTab: state.setInnerTab,
      bottomTab: state.bottomTab,
      setBottomTab: state.setBottomTab,
      statusFilter: state.statusFilter,
      setStatusFilter: state.setStatusFilter,
      searchValue: state.searchValue,
      setSearchValue: state.setSearchValue,
      selectedObjectID: state.selectedObjectID,
      setSelectedObjectID: state.setSelectedObjectID,
      selectedSignalRowID: state.selectedSignalRowID,
      setSelectedSignalRowID: state.setSelectedSignalRowID,
      isCardModalOpen: state.isCardModalOpen,
      setIsCardModalOpen: state.setIsCardModalOpen,
      isEventModalOpen: state.isEventModalOpen,
      setIsEventModalOpen: state.setIsEventModalOpen,
      cardModalTab: state.cardModalTab,
      setCardModalTab: state.setCardModalTab,
      eventModalTab: state.eventModalTab,
      setEventModalTab: state.setEventModalTab,
      eventModalRowID: state.eventModalRowID,
      setEventModalRowID: state.setEventModalRowID,
    })),
  )

  const [clockValue, setClockValue] = useState(formatClock(new Date()))
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false)
  const [settingsDraft, setSettingsDraft] = useState<FrontendDBSettings | null>(null)
  const [settingsBusy, setSettingsBusy] = useState(false)
  const [settingsError, setSettingsError] = useState('')
  const [settingsSuccess, setSettingsSuccess] = useState('')
  const [expandedUnprocessedGroups, setExpandedUnprocessedGroups] = useState<Record<string, boolean>>({})
  const [isAlarmProcessingModalOpen, setIsAlarmProcessingModalOpen] = useState(false)
  const [alarmProcessingOptions, setAlarmProcessingOptions] = useState<FrontendAlarmProcessingOption[]>([])
  const [alarmProcessingLoading, setAlarmProcessingLoading] = useState(false)
  const [alarmProcessingBusy, setAlarmProcessingBusy] = useState(false)
  const [alarmProcessingError, setAlarmProcessingError] = useState('')
  const [alarmWorkflowBusy, setAlarmWorkflowBusy] = useState(false)
  const [alarmWorkflowError, setAlarmWorkflowError] = useState('')
  const [showAllAlarms, setShowAllAlarms] = useState(false)
  const [groupDispatchedAlarmIDs, setGroupDispatchedAlarmIDs] = useState<Set<number>>(new Set())
  const [groupArrivedAlarmIDs, setGroupArrivedAlarmIDs] = useState<Set<number>>(new Set())
  const [isDispatchGroupModalOpen, setIsDispatchGroupModalOpen] = useState(false)
  const [dispatchGroupBusy, setDispatchGroupBusy] = useState(false)
  const [dispatchGroupError, setDispatchGroupError] = useState('')
  const [isStandbyModalOpen, setIsStandbyModalOpen] = useState(false)
  const [standbyBusy, setStandbyBusy] = useState(false)
  const [standbyError, setStandbyError] = useState('')
  const [responseGroups, setResponseGroups] = useState<FrontendResponseGroup[]>([])
  const [cachedProcessingOptions, setCachedProcessingOptions] = useState<FrontendAlarmProcessingOption[]>([])

  const objectsQuery = useQuery({
    queryKey: ['frontend', 'objects'],
    queryFn: () => api.listObjects(),
    refetchInterval: 30_000,
  })

  const capabilitiesQuery = useQuery({
    queryKey: ['frontend', 'capabilities'],
    queryFn: () => api.capabilities(),
    refetchInterval: 10_000,
  })

  useEffect(() => {
    void api.listResponseGroups().then(setResponseGroups).catch(() => {})
    void api.listAlarmProcessingOptionsCached().then(setCachedProcessingOptions).catch(() => {})
  }, [])

  const journalStream = useJournalStream(api)

  useEffect(() => {
    const id = window.setInterval(() => setClockValue(formatClock(new Date())), 1000)
    return () => window.clearInterval(id)
  }, [])

  useEffect(() => {
    document.documentElement.dataset.theme = themeMode
    document.documentElement.style.colorScheme = themeMode
  }, [themeMode])

  useEffect(() => {
    if (!isSettingsModalOpen) {
      return
    }

    let cancelled = false
    setSettingsBusy(true)
    setSettingsError('')
    setSettingsSuccess('')

    void api
      .getDBSettings()
      .then((settings) => {
        if (!cancelled) {
          setSettingsDraft(settings)
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setSettingsError(error instanceof Error ? error.message : String(error))
        }
      })
      .finally(() => {
        if (!cancelled) {
          setSettingsBusy(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [isSettingsModalOpen])

  const alarmIndex = useMemo(() => {
    const map = new Map<number, number>()
    for (const item of journalStream.alarmGroups) {
      map.set(item.objectID, (map.get(item.objectID) ?? 0) + 1)
    }
    return map
  }, [journalStream.alarmGroups])

  const objectRows = useMemo(
    () => (objectsQuery.data ?? []).map((item) => toObjectRow(item, alarmIndex.get(item.id) ?? 0)),
    [alarmIndex, objectsQuery.data],
  )

  const caslCapability = useMemo<FrontendSourceCapability | null>(
    () => capabilitiesQuery.data?.sources.find((item) => item.source === 'casl') ?? null,
    [capabilitiesQuery.data],
  )

  const caslHealthClass = useMemo(() => {
    switch (caslCapability?.healthStatus) {
      case 'offline':
        return 'source-health-banner source-health-banner--error'
      case 'degraded':
        return 'source-health-banner source-health-banner--warn'
      case 'unknown':
        return 'source-health-banner source-health-banner--info'
      default:
        return ''
    }
  }, [caslCapability?.healthStatus])

  const caslStatusbarClass = useMemo(() => {
    switch (caslCapability?.healthStatus) {
      case 'online':
        return 'status-link status-link--ok'
      case 'offline':
        return 'status-link status-link--error'
      case 'degraded':
        return 'status-link status-link--warn'
      default:
        return 'status-link'
    }
  }, [caslCapability?.healthStatus])

  const effectiveSelectedObjectID = selectedObjectID ?? objectRows[0]?.id ?? null

  const detailsQuery = useQuery({
    queryKey: ['frontend', 'object-details', effectiveSelectedObjectID],
    queryFn: () => api.getObjectDetails(effectiveSelectedObjectID ?? 0),
    enabled: effectiveSelectedObjectID != null,
    refetchInterval: 30_000,
  })

  const isObjectEventsTabOpen =
    (isCardModalOpen && cardModalTab === 'events_tab') || (isEventModalOpen && eventModalTab === 'events_tab')

  const objectEventsFeed = useObjectEventsFeed(api, effectiveSelectedObjectID, isObjectEventsTabOpen)

  const journalAlarmRows = useMemo(
    () => journalStream.alarmGroups.flatMap((group) => group.items.map(toAlarmRow)).sort(sortJournalRowsDesc),
    [journalStream.alarmGroups],
  )

  const unprocessedAlarmGroups = useMemo(() => {
    const filtered = showAllAlarms
      ? journalStream.alarmGroups
      : journalStream.alarmGroups.filter((g) => !g.primary.isInProgress || g.primary.isOwnedByMe || g.primary.canTakeOver)
    const raw = filtered.map(toUnprocessedAlarmGroup)
    return mergeUnprocessedGroupsByObject(raw)
  }, [journalStream.alarmGroups, showAllAlarms])

  useEffect(() => {
    setExpandedUnprocessedGroups((prev) => {
      const activeGroupIDs = new Set(unprocessedAlarmGroups.map((group) => group.groupID))
      const next: Record<string, boolean> = {}
      let changed = false

      for (const groupID of activeGroupIDs) {
        if (prev[groupID] !== undefined) {
          next[groupID] = prev[groupID]
        } else {
          // New group - collapsed by default
          next[groupID] = false
          changed = true
        }
      }
      
      // Also remove old groups
      for (const groupID of Object.keys(prev)) {
        if (!activeGroupIDs.has(groupID)) {
          changed = true
        }
      }

      return changed ? next : prev
    })
  }, [unprocessedAlarmGroups])

  const unprocessedFlatRows = useMemo(
    () => flattenUnprocessedAlarmGroups(unprocessedAlarmGroups, expandedUnprocessedGroups),
    [expandedUnprocessedGroups, unprocessedAlarmGroups],
  )

  const unprocessedRowMetaByID = useMemo(
    () => buildUnprocessedRowMeta(unprocessedAlarmGroups, expandedUnprocessedGroups),
    [expandedUnprocessedGroups, unprocessedAlarmGroups],
  )

  const unprocessedFlatRowsRef = useRef(unprocessedFlatRows)
  useEffect(() => {
    unprocessedFlatRowsRef.current = unprocessedFlatRows
  }, [unprocessedFlatRows])

  const [lastSeenAlarmTime, setLastSeenAlarmTime] = useState(0)

  useEffect(() => {
    let maxT = 0
    for (const g of journalStream.alarmGroups) {
      const hasCriticalItem = g.items.some(
        (item) => item.visualSeverity === 'critical' || (item.visualSeverity === 'unknown' && item.typeCode.toLowerCase() !== 'fault'),
      )
      if (!hasCriticalItem) continue
      const t = new Date(g.latestTime).getTime()
      if (t > maxT) maxT = t
    }
    if (maxT > lastSeenAlarmTime) {
      setLastSeenAlarmTime(maxT)
    }
  }, [journalStream.alarmGroups, lastSeenAlarmTime])

  useEffect(() => {
    if (lastSeenAlarmTime === 0) return

    if (mainTab === 'signals' && bottomTab === 'unproc') {
      return
    }

    const timerId = window.setTimeout(() => {
      setMainTab('signals')
      setBottomTab('unproc')

      const rows = unprocessedFlatRowsRef.current
      if (rows.length > 0) {
        setSelectedSignalRowID(rows[0].rowID)
        setSelectedObjectID(rows[0].objectID)
      }
    }, 30000)

    return () => window.clearTimeout(timerId)
  }, [lastSeenAlarmTime, mainTab, bottomTab, setMainTab, setBottomTab, setSelectedSignalRowID, setSelectedObjectID])

  const journalArchiveRows = useMemo(() => {
    const threshold = Date.now() - RECENT_JOURNAL_WINDOW_MS
    const recentEventRows = journalStream.events
      .map(toArchiveRow)
      .filter((row) => row.sortTimestampMs > 0 && row.sortTimestampMs >= threshold)
    return [...recentEventRows, ...journalAlarmRows].sort(sortJournalRowsDesc)
  }, [journalAlarmRows, journalStream.events])

  const effectiveSelectedSignalRowID =
    selectedSignalRowID ?? unprocessedFlatRows[0]?.rowID ?? journalArchiveRows[0]?.rowID ?? null

  const selectedSignalRow = useMemo(() => {
    const merged = [...journalAlarmRows, ...journalArchiveRows]
    return merged.find((item) => item.rowID === effectiveSelectedSignalRowID) ?? merged[0] ?? null
  }, [effectiveSelectedSignalRowID, journalAlarmRows, journalArchiveRows])

  const selectedObjectRow = useMemo(
    () => objectRows.find((item) => item.id === effectiveSelectedObjectID) ?? null,
    [effectiveSelectedObjectID, objectRows],
  )

  const [localPickedAlarmIDs, setLocalPickedAlarmIDs] = useState<Set<number>>(new Set())

  const activeAlarmRow = useMemo(() => {
    const row = selectedSignalRow
    if (row == null || !row.alarm) return null
    if (row.alarmID == null) {
      return row
    }
    const pickedByMe = localPickedAlarmIDs.has(row.alarmID)
    const groupDispatched = row.responseGroupDispatched || groupDispatchedAlarmIDs.has(row.alarmID)
    const groupArrived = row.responseGroupArrived || groupArrivedAlarmIDs.has(row.alarmID)
    if (!pickedByMe && !groupDispatched && !groupArrived) {
      return row
    }
    return {
      ...row,
      inProgressByMe: pickedByMe || row.inProgressByMe,
      canProcess: (pickedByMe || row.canProcess) && !groupDispatched,
      responseGroupDispatched: groupDispatched,
      responseGroupArrived: groupArrived,
    }
  }, [selectedSignalRow, localPickedAlarmIDs, groupDispatchedAlarmIDs, groupArrivedAlarmIDs])

  useEffect(() => {
    if (!isEventModalOpen) {
      setAlarmWorkflowBusy(false)
      setAlarmWorkflowError('')
      return
    }
    setAlarmWorkflowError('')
  }, [eventModalRowID, isEventModalOpen])

  const selectedObjectEvents = useMemo(
    () => objectEventsFeed.events.map(toArchiveRow).sort(sortJournalRowsDesc),
    [objectEventsFeed.events],
  )

  const liveObjectEvents = useMemo(
    () => journalArchiveRows.filter((row) => row.objectID === effectiveSelectedObjectID),
    [journalArchiveRows, effectiveSelectedObjectID],
  )

  const selectedObjectZones = detailsQuery.data?.zones ?? []
  const selectedObjectContacts = detailsQuery.data?.contacts ?? []

  const updateSettingsDraft = useCallback((patch: Partial<FrontendDBSettings>) => {
    setSettingsDraft((prev) => (prev == null ? prev : { ...prev, ...patch }))
  }, [])

  const handleSaveSettings = useCallback(async () => {
    if (settingsDraft == null) {
      return
    }

    setSettingsBusy(true)
    setSettingsError('')
    setSettingsSuccess('')
    try {
      await api.saveDBSettings(settingsDraft)
      setSettingsSuccess('Налаштування збережено та застосовано')
      await Promise.all([objectsQuery.refetch(), detailsQuery.refetch()])
    } catch (error: unknown) {
      setSettingsError(error instanceof Error ? error.message : String(error))
    } finally {
      setSettingsBusy(false)
    }
  }, [detailsQuery, objectsQuery, settingsDraft])

  const toggleUnprocessedGroup = useCallback((groupID: string) => {
    setExpandedUnprocessedGroups((prev) => ({ ...prev, [groupID]: !prev[groupID] }))
  }, [])

  const handleSelectObject = useCallback(
    (objectID: number) => {
      setSelectedObjectID(objectID)
      setIsCardModalOpen(true)
      setCardModalTab('kartochka')
    },
    [setCardModalTab, setIsCardModalOpen, setSelectedObjectID],
  )

  const handleSelectSignalRow = useCallback(
    (row: { rowID: string; objectID: number }) => {
      setSelectedSignalRowID(row.rowID)
      setSelectedObjectID(row.objectID)
    },
    [setSelectedObjectID, setSelectedSignalRowID],
  )

  const handleOpenEventModal = useCallback(
    (row: { rowID: string; objectID: number }) => {
      handleSelectSignalRow(row)
      setEventModalRowID(row.rowID)
      setIsEventModalOpen(true)
      setEventModalTab('kartochka')
    },
    [handleSelectSignalRow, setEventModalRowID, setEventModalTab, setIsEventModalOpen],
  )

  const handleOpenCardModal = useCallback(
    (row: { rowID: string; objectID: number }) => {
      handleSelectSignalRow(row)
      setIsCardModalOpen(true)
      setCardModalTab('kartochka')
    },
    [handleSelectSignalRow, setCardModalTab, setIsCardModalOpen],
  )

  const handleOpenAlarmProcessing = useCallback(() => {
    const alarmID = activeAlarmRow?.alarmID
    if (alarmID == null || alarmID <= 0) {
      setAlarmProcessingError('Для вибраного рядка недоступне відпрацювання тривоги.')
      setIsAlarmProcessingModalOpen(true)
      setAlarmProcessingOptions([])
      return
    }
    if (activeAlarmRow?.responseGroupDispatched) {
      setAlarmWorkflowError(
        activeAlarmRow.responseGroupArrived
          ? 'На тривогу вже прибула МГР. Спочатку зніміть групу, потім завершуйте тривогу.'
          : 'На тривогу вже вислана МГР. Спочатку відмітьте прибуття або зніміть групу.',
      )
      return
    }
    if (activeAlarmRow?.canProcess === false) {
      const assignee = activeAlarmRow.inProgressBy || 'інший оператор'
      setAlarmWorkflowError(`Тривога вже обробляється: ${assignee}. Спочатку перехопіть її.`)
      return
    }

    setIsAlarmProcessingModalOpen(true)
    setAlarmProcessingBusy(false)
    setAlarmProcessingError('')
    if (cachedProcessingOptions.length > 0) {
      setAlarmProcessingOptions(cachedProcessingOptions)
      setAlarmProcessingLoading(false)
    } else {
      setAlarmProcessingLoading(true)
      void api
        .getAlarmProcessingOptions(alarmID)
        .then((items) => {
          setAlarmProcessingOptions(items)
        })
        .catch((error: unknown) => {
          setAlarmProcessingOptions([])
          setAlarmProcessingError(error instanceof Error ? error.message : String(error))
        })
        .finally(() => {
          setAlarmProcessingLoading(false)
        })
    }
  }, [activeAlarmRow?.alarmID, activeAlarmRow?.canProcess, cachedProcessingOptions])

  const handleSubmitAlarmProcessing = useCallback(
    async ({ causeCode, note }: { causeCode: string; note: string }) => {
      if (!activeAlarmRow || !activeAlarmRow.alarmID || activeAlarmRow.alarmID <= 0) {
        setAlarmProcessingError('Некоректний ідентифікатор тривоги.')
        return
      }
      const alarmID = activeAlarmRow.alarmID

      setAlarmProcessingBusy(true)
      setAlarmProcessingError('')
      try {
        setAlarmWorkflowError('')
        
        const activeObjectID = activeAlarmRow.objectID
        const source = activeAlarmRow.source
        const alarmsCount = journalStream.alarmGroups.filter((g) => g.objectID === activeObjectID).length

        if (source !== 'casl' && alarmsCount > 1) {
          await api.groupProcessAlarm(alarmID, OPERATOR_NAME)
        } else {
          await api.processAlarm(alarmID, {
            user: OPERATOR_NAME,
            causeCode,
            note,
          })
        }
        
        setLocalPickedAlarmIDs((prev) => { const next = new Set(prev); next.delete(alarmID); return next })
        setIsAlarmProcessingModalOpen(false)
        setIsEventModalOpen(false)
        await Promise.all([objectsQuery.refetch(), detailsQuery.refetch()])
      } catch (error: unknown) {
        setAlarmProcessingError(error instanceof Error ? error.message : String(error))
      } finally {
        setAlarmProcessingBusy(false)
      }
    },
    [detailsQuery, activeAlarmRow, objectsQuery, setIsEventModalOpen, journalStream.alarmGroups],
  )

  const handlePickAlarm = useCallback(async () => {
    const alarmID = activeAlarmRow?.alarmID
    if (alarmID == null || alarmID <= 0) {
      setAlarmWorkflowError('Для вибраного рядка недоступне взяття тривоги в роботу.')
      return
    }
    if (activeAlarmRow?.responseGroupDispatched) {
      setAlarmWorkflowError('Тривога вже перейшла в етап МГР. Перехоплення через UI для цього стану недоступне.')
      return
    }
    if (activeAlarmRow?.inProgressByMe) {
      setAlarmWorkflowError('')
      return
    }

    setAlarmWorkflowBusy(true)
    setAlarmWorkflowError('')
    try {
      await api.pickAlarm(alarmID, { user: OPERATOR_NAME })
      setLocalPickedAlarmIDs((prev) => new Set([...prev, alarmID]))
      if (activeAlarmRow?.source === 'casl') {
        setAlarmWorkflowError('Команду перехоплення відправлено. Очікуємо оновлення CASL.')
      }
    } catch (error: unknown) {
      setAlarmWorkflowError(error instanceof Error ? error.message : String(error))
    } finally {
      setAlarmWorkflowBusy(false)
    }
  }, [activeAlarmRow?.alarmID, activeAlarmRow?.inProgressByMe, activeAlarmRow?.responseGroupDispatched, activeAlarmRow?.source])

  const handleStandby = useCallback(() => {
    const objectID = activeAlarmRow?.objectID
    if (objectID == null || objectID <= 0) return
    setStandbyError('')
    setIsStandbyModalOpen(true)
  }, [activeAlarmRow?.objectID])

  const handleConfirmStandby = useCallback(async (durationMinutes: number, reason: string) => {
    const objectID = activeAlarmRow?.objectID
    if (objectID == null || objectID <= 0) return
    setStandbyBusy(true)
    setStandbyError('')
    try {
      await api.standbyObject(objectID, durationMinutes, reason)
      setIsStandbyModalOpen(false)
    } catch (error: unknown) {
      setStandbyError(error instanceof Error ? error.message : String(error))
    } finally {
      setStandbyBusy(false)
    }
  }, [activeAlarmRow?.objectID])

  const handleDispatchGroup = useCallback(() => {
    const alarmID = activeAlarmRow?.alarmID
    if (alarmID == null || alarmID <= 0) return
    if (activeAlarmRow?.responseGroupDispatched) {
      setAlarmWorkflowError('МГР вже призначена для цієї тривоги.')
      return
    }
    setDispatchGroupError('')
    void api.listResponseGroups().then(setResponseGroups).catch(() => {})
    setIsDispatchGroupModalOpen(true)
  }, [activeAlarmRow?.alarmID, activeAlarmRow?.responseGroupDispatched])

  const handleConfirmDispatchGroup = useCallback(async (groupID: string) => {
    const alarmID = activeAlarmRow?.alarmID
    if (alarmID == null || alarmID <= 0) return
    setDispatchGroupBusy(true)
    setDispatchGroupError('')
    try {
      await api.assignResponseGroup(alarmID, { groupID })
      setGroupDispatchedAlarmIDs((prev) => new Set(prev).add(alarmID))
      setIsDispatchGroupModalOpen(false)
    } catch (error: unknown) {
      setDispatchGroupError(error instanceof Error ? error.message : String(error))
    } finally {
      setDispatchGroupBusy(false)
    }
  }, [activeAlarmRow?.alarmID])

  const handleGroupAction = useCallback(async () => {
    const alarmID = activeAlarmRow?.alarmID
    if (alarmID == null || alarmID <= 0) return
    const isArrived = groupArrivedAlarmIDs.has(alarmID)
    setAlarmWorkflowBusy(true)
    setAlarmWorkflowError('')
    try {
      if (isArrived) {
        await api.cancelResponseGroup(alarmID)
        setGroupArrivedAlarmIDs((prev) => { const next = new Set(prev); next.delete(alarmID); return next })
        setGroupDispatchedAlarmIDs((prev) => { const next = new Set(prev); next.delete(alarmID); return next })
      } else {
        await api.notifyGroupArrived(alarmID)
        setGroupArrivedAlarmIDs((prev) => new Set(prev).add(alarmID))
      }
    } catch (error: unknown) {
      setAlarmWorkflowError(error instanceof Error ? error.message : String(error))
    } finally {
      setAlarmWorkflowBusy(false)
    }
  }, [activeAlarmRow?.alarmID, groupArrivedAlarmIDs])

  const handleCancelAlarm = useCallback(async () => {
    if (!activeAlarmRow || !activeAlarmRow.alarmID || activeAlarmRow.alarmID <= 0) return
    const alarmID = activeAlarmRow.alarmID
    if (activeAlarmRow.responseGroupDispatched) {
      setAlarmWorkflowError('Поки активна МГР, завершення тривоги через UI недоступне. Спочатку зніміть групу.')
      return
    }
    setAlarmWorkflowBusy(true)
    setAlarmWorkflowError('')
    try {
      const activeObjectID = activeAlarmRow.objectID
      const source = activeAlarmRow.source
      const alarmsCount = journalStream.alarmGroups.filter((g) => g.objectID === activeObjectID).length

      if (source !== 'casl' && alarmsCount > 1) {
        await api.groupProcessAlarm(alarmID, OPERATOR_NAME)
      } else {
        await api.processAlarm(alarmID, { user: OPERATOR_NAME, causeCode: 'FALSE_ALARM', note: 'Скасовано оператором' })
      }
      
      setIsEventModalOpen(false)
      setLocalPickedAlarmIDs((prev) => {
        const next = new Set(prev)
        next.delete(alarmID)
        return next
      })
    } catch (error: unknown) {
      setAlarmWorkflowError(error instanceof Error ? error.message : String(error))
    } finally {
      setAlarmWorkflowBusy(false)
    }
  }, [activeAlarmRow, journalStream.alarmGroups])

  const handleToggleShowAll = useCallback(() => setShowAllAlarms((v) => !v), [])

  return (
    <div>
      <div className="shell">
        <aside className="sidebar">
          <div className="sb-top">
            <div style={{ height: 2 }} />
            {SIDEBAR_PRIMARY_BUTTONS.map(({ label, icon, active = false }) => (
              <button key={label} className={active ? 'sb-btn active' : 'sb-btn'}>
                {icon}
                <span className="sb-tip">{label}</span>
              </button>
            ))}
          </div>
          <div className="sb-bot">
            <button className="sb-btn">
              <PrintIcon />
              <span className="sb-tip">Друк</span>
            </button>
            <button className={isSettingsModalOpen ? 'sb-btn active' : 'sb-btn'} onClick={() => setIsSettingsModalOpen(true)}>
              <SettingsIcon />
              <span className="sb-tip">Налаштування</span>
            </button>
            <button className="sb-btn" style={{ background: 'rgba(14,158,120,.1)' }}>
              <LinkIcon />
              <span className="sb-tip">Звʼязок</span>
            </button>
          </div>
        </aside>

        <div className="main">
          <div className="main-tabs">
            <div className={mainTab === 'objects' ? 'mtab active' : 'mtab'} onClick={() => setMainTab('objects')}>
              Об'єкти
            </div>
            <div className={mainTab === 'signals' ? 'mtab active' : 'mtab'} onClick={() => setMainTab('signals')}>
              Прийом сигналів
            </div>
          </div>

          {caslCapability != null && caslCapability.healthStatus !== 'online' && (
            <div className={caslHealthClass}>{caslCapability.healthText}</div>
          )}

          <div className={mainTab === 'signals' ? 'tpane active' : 'tpane'}>
            <SignalsTab
              innerTab={innerTab}
              bottomTab={bottomTab}
              selectedSignalRow={selectedSignalRow}
              selectedObjectRow={selectedObjectRow}
              objectDetails={detailsQuery.data ?? null}
              unprocessedAlarmGroups={unprocessedAlarmGroups}
              journalArchiveRows={journalArchiveRows}
              unprocessedFlatRows={unprocessedFlatRows}
              unprocessedRowMetaByID={unprocessedRowMetaByID}
              expandedUnprocessedGroups={expandedUnprocessedGroups}
              showAllAlarms={showAllAlarms}
              selectedSignalRowID={effectiveSelectedSignalRowID}
              onSelectInnerTab={setInnerTab}
              onSelectBottomTab={setBottomTab}
              onToggleGroup={toggleUnprocessedGroup}
              onToggleShowAll={handleToggleShowAll}
              onSelectSignalRow={handleSelectSignalRow}
              onOpenEventModal={handleOpenEventModal}
              onOpenCardModal={handleOpenCardModal}
              isInWorkflow={activeAlarmRow?.inProgressByMe === true}
              groupDispatched={activeAlarmRow?.responseGroupDispatched === true}
              groupArrived={activeAlarmRow?.responseGroupArrived === true}
              workflowBusy={alarmWorkflowBusy || alarmProcessingBusy}
              onPickAlarm={() => void handlePickAlarm()}
              onStandby={() => void handleStandby()}
              onCancelAlarm={() => void handleCancelAlarm()}
              onDispatchGroup={handleDispatchGroup}
              onGroupAction={() => void handleGroupAction()}
              onOpenProcessAlarm={handleOpenAlarmProcessing}
            />
          </div>

          <div className={mainTab === 'objects' ? 'tpane active' : 'tpane'}>
            <ObjectsTab
              rows={objectRows}
              searchValue={searchValue}
              statusFilter={statusFilter}
              selectedObjectID={selectedObjectID}
              onSearchChange={setSearchValue}
              onStatusFilterChange={setStatusFilter}
              onRefresh={() => void objectsQuery.refetch()}
              onSelectObject={handleSelectObject}
            />
          </div>

          <div className="statusbar">
            <div className="sb-field">
              <div className="sb-dot" />
              <span>
                Оператор: <strong>{OPERATOR_NAME}</strong>
              </span>
            </div>
            <div className={`sb-field ${caslStatusbarClass}`} style={{ fontSize: 11 }}>
              {caslCapability != null ? `● ${caslCapability.healthText}` : '● CASL не використовується'}
            </div>
            <div className="sb-clock">{clockValue}</div>
          </div>
        </div>
      </div>

      <CardModal
        isOpen={isCardModalOpen}
        selectedObjectRow={selectedObjectRow}
        objectDetails={detailsQuery.data ?? null}
        selectedObjectZones={selectedObjectZones}
        selectedObjectContacts={selectedObjectContacts}
        selectedObjectEvents={selectedObjectEvents}
        objectEventsFeed={objectEventsFeed}
        tab={cardModalTab}
        onSelectTab={setCardModalTab}
        onClose={() => setIsCardModalOpen(false)}
      />

      <EventModal
        isOpen={isEventModalOpen}
        tab={eventModalTab}
        onSelectTab={setEventModalTab}
        onClose={() => setIsEventModalOpen(false)}
        eventModalRow={activeAlarmRow}
        selectedObjectRow={selectedObjectRow}
        objectDetails={detailsQuery.data ?? null}
        selectedObjectZones={selectedObjectZones}
        selectedObjectContacts={selectedObjectContacts}
        selectedObjectEvents={selectedObjectEvents}
        liveObjectEvents={liveObjectEvents}
        objectEventsFeed={objectEventsFeed}
        workflowBusy={alarmWorkflowBusy || alarmProcessingBusy}
        workflowError={alarmWorkflowError}
        isInWorkflow={activeAlarmRow?.inProgressByMe === true}
        groupDispatched={activeAlarmRow?.responseGroupDispatched === true}
        groupArrived={activeAlarmRow?.responseGroupArrived === true}
        onPickAlarm={() => void handlePickAlarm()}
        onStandby={() => void handleStandby()}
        onCancelAlarm={() => void handleCancelAlarm()}
        onDispatchGroup={handleDispatchGroup}
        onGroupAction={() => void handleGroupAction()}
        onOpenProcessAlarm={handleOpenAlarmProcessing}
      />

      <SettingsModal
        isOpen={isSettingsModalOpen}
        settingsDraft={settingsDraft}
        settingsBusy={settingsBusy}
        settingsError={settingsError}
        settingsSuccess={settingsSuccess}
        themeMode={themeMode}
        onClose={() => setIsSettingsModalOpen(false)}
        onSave={() => void handleSaveSettings()}
        onUpdateDraft={updateSettingsDraft}
        onThemeChange={setThemeMode}
      />

      <StandbyModal
        isOpen={isStandbyModalOpen}
        busy={standbyBusy}
        error={standbyError}
        onClose={() => setIsStandbyModalOpen(false)}
        onConfirm={(d, r) => void handleConfirmStandby(d, r)}
      />

      <DispatchGroupModal
        isOpen={isDispatchGroupModalOpen}
        groups={responseGroups}
        preferredGroupID={detailsQuery.data?.preferredResponseGroupID || ''}
        preferredGroupName={detailsQuery.data?.preferredResponseGroupName || ''}
        objectGroupHint={detailsQuery.data?.notes || selectedObjectRow?.note || ''}
        busy={dispatchGroupBusy}
        error={dispatchGroupError}
        onClose={() => setIsDispatchGroupModalOpen(false)}
        onConfirm={(groupID) => void handleConfirmDispatchGroup(groupID)}
      />

      <AlarmProcessingModal
        isOpen={isAlarmProcessingModalOpen}
        eventModalRow={activeAlarmRow}
        selectedObjectRow={selectedObjectRow}
        objectDetails={detailsQuery.data ?? null}
        selectedObjectZones={selectedObjectZones}
        selectedObjectContacts={selectedObjectContacts}
        selectedObjectEvents={selectedObjectEvents}
        liveObjectEvents={liveObjectEvents}
        objectEventsFeed={objectEventsFeed}
        options={alarmProcessingOptions}
        loading={alarmProcessingLoading}
        busy={alarmProcessingBusy}
        error={alarmProcessingError}
        onClose={() => setIsAlarmProcessingModalOpen(false)}
        onSubmit={(payload) => void handleSubmitAlarmProcessing(payload)}
      />
    </div>
  )
}

const SIDEBAR_PRIMARY_BUTTONS = [
  { label: 'Інформація', icon: <InfoIcon />, active: true },
  { label: 'Статус', icon: <StatusIcon /> },
  { label: 'Схема', icon: <GridIcon /> },
  { label: 'Фото', icon: <PhotoIcon /> },
  { label: 'Стенди', icon: <StandIcon /> },
  { label: 'Групи реагування', icon: <ResponseIcon /> },
  { label: 'Стоп-лист', icon: <StopIcon /> },
  { label: 'Журнал', icon: <JournalIcon /> },
  { label: 'Дистанційне керування', icon: <RemoteIcon /> },
]

function InfoIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="8" x2="12" y2="12" />
      <line x1="12" y1="16" x2="12.01" y2="16" />
    </svg>
  )
}

function StatusIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
    </svg>
  )
}

function GridIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <rect x="3" y="3" width="7" height="7" />
      <rect x="14" y="3" width="7" height="7" />
      <rect x="14" y="14" width="7" height="7" />
      <rect x="3" y="14" width="7" height="7" />
    </svg>
  )
}

function PhotoIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M23 19a2 2 0 01-2 2H3a2 2 0 01-2-2V8a2 2 0 012-2h4l2-3h6l2 3h4a2 2 0 012 2z" />
      <circle cx="12" cy="13" r="4" />
    </svg>
  )
}

function StandIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <rect x="2" y="3" width="20" height="14" rx="2" />
      <line x1="8" y1="21" x2="16" y2="21" />
      <line x1="12" y1="17" x2="12" y2="21" />
    </svg>
  )
}

function ResponseIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 00-3-3.87" />
      <path d="M16 3.13a4 4 0 010 7.75" />
    </svg>
  )
}

function StopIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <circle cx="12" cy="12" r="10" />
      <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
    </svg>
  )
}

function JournalIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
      <polyline points="14 2 14 8 20 8" />
    </svg>
  )
}

function RemoteIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
    </svg>
  )
}

function PrintIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <polyline points="6 9 6 2 18 2 18 9" />
      <path d="M6 18H4a2 2 0 01-2-2v-5a2 2 0 012-2h16a2 2 0 012 2v5a2 2 0 01-2 2h-2" />
      <rect x="6" y="14" width="12" height="8" />
    </svg>
  )
}

function SettingsIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
    </svg>
  )
}

function LinkIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" style={{ color: 'var(--ac)' }}>
      <polyline points="23 6 13.5 15.5 8.5 10.5 1 18" />
      <polyline points="17 6 23 6 23 12" />
    </svg>
  )
}
