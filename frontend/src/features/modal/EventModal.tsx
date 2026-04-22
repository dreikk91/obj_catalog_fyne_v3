import { useCallback, useMemo } from 'react'
import type { UIEvent } from 'react'
import type {
  FrontendContact,
  FrontendObjectDetails,
  FrontendObjectSummary,
  FrontendSource,
  FrontendZone,
  ConnectionStatus,
  GuardStatus,
  MonitoringStatus,
} from '../../shared/api/types'
import type { ModalTab } from '../../shared/state/ui-store'
import { formatEventTime } from '../../shared/ui/time'
import { useVirtualRows } from '../../hooks/useVirtualRows'
import { useColumnVisibility } from '../../hooks/useColumnVisibility'
import { ColumnVisibilityButton } from '../../shared/ui/ColumnVisibilityButton'
import { BASE_GROUP_NAMES, BASE_KEY_OWNERS, MODAL_TABS } from '../operator/constants'
import type { JournalRow, ObjectRow } from '../operator/types'
import { pad2, resolveJournalTypeClass } from '../operator/utils'

type GroupedRowHeader = { type: 'header'; groupName: string; groupNumber: number; groupStateText: string; groupID: string; id: string }
type GroupedRowItem<T> = { type: 'item'; item: T; id: string }
type GroupedRow<T> = GroupedRowHeader | GroupedRowItem<T>

function useGroupedRows<T extends { groupName: string; groupNumber: number; groupStateText: string; groupID: string }>(items: T[]): GroupedRow<T>[] {
  return useMemo(() => {
    const groups = new Map<string, T[]>()
    for (const item of items) {
      const gkey = `${item.groupNumber}:${item.groupName || 'Без групи'}`
      const arr = groups.get(gkey) || []
      arr.push(item)
      groups.set(gkey, arr)
    }
    const grouped: GroupedRow<T>[] = []
    for (const [key, groupItems] of groups.entries()) {
      if (groupItems.length === 0) continue
      const ref = groupItems[0]
      const gname = ref.groupName || 'Без групи'
      grouped.push({ type: 'header', groupName: gname, groupNumber: ref.groupNumber, groupStateText: ref.groupStateText, groupID: ref.groupID, id: `header-${key}` })
      for (let i = 0; i < groupItems.length; i++) {
        grouped.push({ type: 'item', item: groupItems[i], id: `item-${key}-${i}` })
      }
    }
    return grouped
  }, [items])
}

type EventModalProps = {
  isOpen: boolean
  tab: ModalTab
  onSelectTab: (tab: ModalTab) => void
  onClose: () => void
  eventModalRow: JournalRow | null
  selectedObjectRow: ObjectRow | null
  objectDetails?: FrontendObjectDetails | null
  selectedObjectZones: FrontendZone[]
  selectedObjectContacts: FrontendContact[]
  selectedObjectEvents: JournalRow[]
  liveObjectEvents: JournalRow[]
  objectEventsFeed: {
    totalCount: number
    hasMore: boolean
    isInitialLoading: boolean
    isLoadingMore: boolean
    loadMore: () => void
  }
  workflowBusy: boolean
  workflowError: string
  isInWorkflow: boolean
  groupDispatched: boolean
  groupArrived: boolean
  onPickAlarm: () => void
  onStandby: () => void
  onCancelAlarm: () => void
  onDispatchGroup: () => void
  onGroupAction: () => void
  onOpenProcessAlarm: () => void
}

export function EventModal({
  isOpen,
  tab,
  onSelectTab,
  onClose,
  eventModalRow,
  selectedObjectRow,
  objectDetails,
  selectedObjectZones,
  selectedObjectContacts,
  selectedObjectEvents,
  liveObjectEvents,
  objectEventsFeed,
  workflowBusy,
  workflowError,
  isInWorkflow,
  groupDispatched,
  groupArrived,
  onPickAlarm,
  onStandby,
  onCancelAlarm,
  onDispatchGroup,
  onGroupAction,
  onOpenProcessAlarm,
}: EventModalProps) {
  const groupedZones = useGroupedRows(selectedObjectZones)
  const zonesVirtual = useVirtualRows(groupedZones, { rowHeight: 28, initialCount: 120, step: 120 })
  const responseVirtual = useVirtualRows(BASE_GROUP_NAMES, { rowHeight: 28, initialCount: 80, step: 80 })
  const keysVirtual = useVirtualRows(BASE_KEY_OWNERS, { rowHeight: 28, initialCount: 80, step: 80 })
  const groupedContacts = useGroupedRows(selectedObjectContacts)
  const contactsVirtual = useVirtualRows(groupedContacts, { rowHeight: 28, initialCount: 120, step: 120 })
  const eventsVirtual = useVirtualRows(selectedObjectEvents, { rowHeight: 28, initialCount: 160, step: 160 })
  const workflowHint = useMemo(() => {
    if (eventModalRow == null) return ''
    if (groupDispatched) {
      return groupArrived
        ? 'МГР вже прибула на обʼєкт. Спочатку зніміть групу, після цього завершуйте тривогу.'
        : 'МГР вже вислана на обʼєкт. Можна підтвердити прибуття або зняти групу.'
    }
    if (eventModalRow.inProgress && !isInWorkflow && eventModalRow.inProgressBy !== '') {
      return `Тривога вже у роботі: ${eventModalRow.inProgressBy}. За потреби її можна перехопити.`
    }
    if (isInWorkflow) {
      return 'Тривога закріплена за вами. Доступні дії оператора та керування МГР.'
    }
    return ''
  }, [eventModalRow, groupArrived, groupDispatched, isInWorkflow])

  const handleEventsScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      eventsVirtual.onScroll(event)
      const container = event.currentTarget
      if (objectEventsFeed.hasMore && !objectEventsFeed.isLoadingMore && container.scrollTop + container.clientHeight >= container.scrollHeight - 160) {
        objectEventsFeed.loadMore()
      }
    },
    [eventsVirtual, objectEventsFeed],
  )

  return (
    <div className={isOpen ? 'modal-overlay open' : 'modal-overlay'}>
      <div className="modal wide">
        <div className="modal-tb">
          <div className="modal-tb-icon" style={{ background: 'var(--ac4)' }}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="3">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
            </svg>
          </div>
          <span className="modal-tb-title">Обробка події —</span>
          <span className="modal-tb-obj">
            {eventModalRow?.typeText ?? 'Тривога'} — {eventModalRow?.objectName ?? '—'} ({eventModalRow?.objectNumber ?? '—'})
          </span>
          <div className="modal-tb-close" onClick={onClose}>✕</div>
        </div>

        <div className="modal-toolbar">
          {MODAL_TABS.map((item) => (
            <button key={item.id} className={tab === item.id ? 'mtb-btn active' : 'mtb-btn'} onClick={() => onSelectTab(item.id)}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                {item.icon}
              </svg>
              {item.label}
            </button>
          ))}
        </div>

        <div className="modal-obj-hdr">
          <div className="obj-num-badge" style={{ background: 'var(--ac4)' }}>
            {eventModalRow?.objectNumber ?? '—'}
          </div>
          <HeaderField label="Назва:" value={eventModalRow?.objectName ?? '—'} />
          <HeaderField label="Подія:" value={eventModalRow?.typeText ?? '—'} alarm />
          <div className="obj-hdr-status obj-hdr-status--alarm">
            <div className="chip-dot" />
            <span>ТРИВОГА</span>
          </div>
        </div>

        <div className="modal-content">
          {tab === 'kartochka' && <CardSummaryPane selectedObjectRow={selectedObjectRow} objectDetails={objectDetails} />}
          {tab === 'devices' && <DevicesPane summary={objectDetails?.summary} />}
          {tab === 'zones' && <ZonesPane virtualRows={zonesVirtual} rows={groupedZones} contacts={selectedObjectContacts} emptyText="Дані відсутні" compact />}
          {tab === 'response' && <ResponsePane virtualRows={responseVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'keys' && <KeysPane virtualRows={keysVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'resp' && <ContactsPane virtualRows={contactsVirtual} rows={groupedContacts} />}
          {tab === 'photo' && <PhotoPane />}
          {tab === 'events_tab' && (
            <EventsPane virtualRows={eventsVirtual} rows={selectedObjectEvents} feed={objectEventsFeed} onScroll={handleEventsScroll} />
          )}
        </div>

        <div className="proc-box">
          <div className="proc-hdr-row">
            <span className="proc-hdr-label">Обробка події</span>
            {workflowError !== '' && <span className="proc-error-inline">{workflowError}</span>}
            <div style={{ flex: 1 }} />
            <button
              className="proc-wbtn proc-wbtn--violet"
              onClick={onPickAlarm}
              disabled={workflowBusy || eventModalRow?.alarmID == null || isInWorkflow || groupDispatched}
            >
              {eventModalRow?.inProgress && !isInWorkflow ? 'Перехопити тривогу' : isInWorkflow ? '● У вас в роботі' : 'Взяти в роботу'}
            </button>
            {groupDispatched && (
              <button className="proc-wbtn proc-wbtn--gray" onClick={onGroupAction} disabled={workflowBusy}>
                {groupArrived ? 'Зняти групу' : 'Підтвердити прибуття'}
              </button>
            )}
            <button
              className="proc-wbtn proc-wbtn--green"
              onClick={onOpenProcessAlarm}
              disabled={workflowBusy || eventModalRow?.alarmID == null || !isInWorkflow || groupDispatched}
            >
              Завершити із причиною
            </button>
          </div>
          {workflowHint !== '' && (
            <div className="proc-hint-line" style={{ marginBottom: 8, color: 'var(--tx2)', fontSize: 12 }}>
              {workflowHint}
            </div>
          )}
          <div className="proc-action-row">
            <button className="proc-lbtn proc-lbtn--green" onClick={onDispatchGroup} disabled={workflowBusy || !isInWorkflow || groupDispatched}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round">
                <path d="M1 3h15v13H1z" /><path d="M16 8h4l3 3v5h-7V8z" />
                <circle cx="5.5" cy="18.5" r="2.5" /><circle cx="18.5" cy="18.5" r="2.5" />
              </svg>
              {groupDispatched ? 'Групу призначено' : 'Вислати групу'}
            </button>
            <button className="proc-lbtn proc-lbtn--blue" onClick={onStandby} disabled={workflowBusy || !isInWorkflow}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7">
                <rect x="2" y="3" width="20" height="14" rx="2" />
                <line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" />
              </svg>
              До стендів
            </button>
            <button className="proc-lbtn proc-lbtn--red" onClick={onCancelAlarm} disabled={workflowBusy || !isInWorkflow || groupDispatched}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7">
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" /><line x1="9" y1="9" x2="15" y2="15" />
              </svg>
              Скасування тривоги
            </button>
          </div>
          <div className="proc-live-events">
            <div className="proc-live-hdr">
              <span>Необроблені події на об'єкті</span>
              {liveObjectEvents.length > 0 && <span className="proc-live-count">{liveObjectEvents.length}</span>}
            </div>
            <div className="proc-live-table-wrap">
              <table className="mtable">
                <thead>
                  <tr>
                    <th style={{ width: 60 }}>Об'єкт</th>
                    <th style={{ width: 88 }}>Дата</th>
                    <th style={{ width: 80 }}>Час</th>
                    <th style={{ width: 38 }}>Гр.</th>
                    <th style={{ width: 130 }}>Тип коду</th>
                    <th style={{ width: 44 }}>Шл.</th>
                    <th style={{ width: 50 }}>Лінія</th>
                    <th style={{ width: 60 }}>Код</th>
                    <th>Опис події</th>
                  </tr>
                </thead>
                <tbody>
                  {liveObjectEvents.length === 0 ? (
                    <tr><td colSpan={9} style={{ textAlign: 'center', color: 'var(--tx2)' }}>Подій не знайдено</td></tr>
                  ) : (
                    liveObjectEvents.slice(0, 20).map((row) => (
                      <tr key={`live-${row.rowID}`}>
                        <td className="bright">{row.objectNumber}</td>
                        <td>{row.date}</td>
                        <td>{row.time}</td>
                        <td className="dim">{row.group || '—'}</td>
                        <td className={resolveJournalTypeClass(row)}>{row.typeText}</td>
                        <td className="mono dim">{row.zone || '—'}</td>
                        <td className="mono dim">{row.line || '—'}</td>
                        <td className="mono dim">{row.code || '—'}</td>
                        <td className="dim">{row.details || '—'}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div className="modal-footer">
          <div style={{ flex: 1 }} />
          <button className="btn btn-red" style={{ width: 140, height: 28 }} onClick={onClose}>ЗАКРИТИ</button>
        </div>
      </div>
    </div>
  )
}

// ── Panes ─────────────────────────────────────────────────────────────────────

function CardSummaryPane({
  selectedObjectRow,
  objectDetails,
}: {
  selectedObjectRow: ObjectRow | null
  objectDetails?: FrontendObjectDetails | null
}) {
  const s = objectDetails?.summary
  const source = s?.source ?? 'unknown'
  const noteLabel = source === 'phoenix' ? 'Примітка' : source === 'bridge' ? 'Опис' : 'Опис / ГМР'

  return (
    <div className="modal-pane active">
      <div className="igrid">
        <div className="isection">
          <div className="isect-title">Загальні відомості</div>
          <InputRow label="Номер об'єкта" value={s?.displayNumber || selectedObjectRow?.number || '—'} mono />
          <InputRow label="Назва" value={s?.name || selectedObjectRow?.name || '—'} />
          <InputRow label="Адреса" value={s?.address || selectedObjectRow?.address || '—'} />
          <InputRow label="Договір" value={s?.contractNumber || selectedObjectRow?.contract || '—'} />
          <InputRow label="Телефони" value={objectDetails?.phones || s?.phone || selectedObjectRow?.phone || '—'} />
          <InputRow label={noteLabel} value={objectDetails?.notes || selectedObjectRow?.note || '—'} />
          {objectDetails?.location ? <InputRow label="Розташування" value={objectDetails.location} /> : null}
          {objectDetails?.launchDate ? <InputRow label="Дата запуску" value={objectDetails.launchDate} /> : null}
          {s?.statusText ? <InputRow label="Статус" value={s.statusText} /> : null}
        </div>
        <div className="isection">
          <div className="isect-title">Прилад та стан</div>
          <div className="irow">
            <label>Охорона</label>
            <GuardChip status={s?.guardStatus ?? 'unknown'} />
          </div>
          <div className="irow">
            <label>Зв'язок</label>
            <ConnectionChip status={s?.connectionStatus ?? 'unknown'} />
          </div>
          <div className="irow">
            <label>Моніторинг</label>
            <MonitoringChip status={s?.monitoringStatus ?? 'unknown'} />
          </div>
          <InputRow label={deviceTypeLabel(source)} value={s?.deviceType || '—'} />
          <InputRow label={panelMarkLabel(source)} value={s?.panelMark || '—'} mono />
          {s?.sim1 ? <InputRow label={sim1Label(source)} value={s.sim1} mono /> : null}
          {s?.sim2 ? <InputRow label={sim2Label(source)} value={s.sim2} mono /> : null}
          {s?.signalStrength ? <InputRow label="Рівень сигналу" value={s.signalStrength} /> : null}
          <InputRow label="Останній тест" value={s?.lastTestTime ? formatEventTime(s.lastTestTime) : selectedObjectRow?.lastTestAt || '—'} />
          <InputRow label="Остання подія" value={s?.lastMessageTime ? formatEventTime(s.lastMessageTime) : selectedObjectRow?.lastEventAt || '—'} />
        </div>
      </div>
    </div>
  )
}

function DevicesPane({ summary }: { summary?: FrontendObjectSummary }) {
  const source = summary?.source ?? 'unknown'
  return (
    <div className="modal-pane active">
      <div className="igrid">
        <div className="isection">
          <div className="isect-title">Встановлені прилади</div>
          <InputRow label={deviceTypeLabel(source)} value={summary?.deviceType || '—'} />
          <InputRow label={panelMarkLabel(source)} value={summary?.panelMark || '—'} mono />
          {summary?.sim1 ? <InputRow label={sim1Label(source)} value={summary.sim1} mono /> : null}
          {summary?.sim2 ? <InputRow label={sim2Label(source)} value={summary.sim2} mono /> : null}
        </div>
        <div className="isection">
          <div className="isect-title">Стан зв'язку</div>
          <div className="irow">
            <label>Зв'язок</label>
            <ConnectionChip status={summary?.connectionStatus ?? 'unknown'} />
          </div>
          {summary?.signalStrength ? <InputRow label="Рівень сигналу" value={summary.signalStrength} /> : null}
          <InputRow label="Останній тест" value={summary?.lastTestTime ? formatEventTime(summary.lastTestTime) : '—'} />
          <InputRow label="Остання подія" value={summary?.lastMessageTime ? formatEventTime(summary.lastMessageTime) : '—'} />
        </div>
      </div>
    </div>
  )
}

function ZonesPane({
  virtualRows,
  rows,
  contacts,
  emptyText,
  compact = false,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<GroupedRow<FrontendZone>>>
  rows: GroupedRow<FrontendZone>[]
  contacts: FrontendContact[]
  emptyText: string
  compact?: boolean
}) {
  const colSpan = compact ? 4 : 6
  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            {compact ? (
              <tr>
                <th style={{ width: 44 }}>№</th>
                <th>Назва зони</th>
                <th>Тип датчика</th>
                <th>Стан</th>
              </tr>
            ) : (
              <tr>
                <th style={{ width: 56 }}>№</th>
                <th style={{ width: 40 }}>Прил.</th>
                <th>Назва зони</th>
                <th style={{ width: 130 }}>Тип датчика</th>
                <th style={{ width: 110 }}>Стан</th>
                <th>Обхід</th>
              </tr>
            )}
          </thead>
          <tbody>
            <SpacerRow colSpan={colSpan} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((row) => {
              if (row.type === 'header') {
                const header = row as GroupedRowHeader
                const groupContacts = contacts.filter((c) => c.groupID === header.groupID)
                return (
                  <tr key={header.id}>
                    <td colSpan={colSpan} className="bright" style={{ background: 'var(--bg3)', padding: '6px 8px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <span style={{ color: 'var(--tx3)', fontSize: 13 }}>▾</span>
                          <span>{header.groupNumber > 0 ? `Група ${header.groupNumber}: ` : ''}{header.groupName}</span>
                        </div>
                        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                          {groupContacts.length > 0 && (
                            <div style={{ display: 'flex', gap: 6, fontSize: 11, color: 'var(--tx2)' }}>
                              <span style={{ opacity: 0.7 }}>Відп:</span>
                              {groupContacts.map((c, idx) => (
                                <span key={c.name} style={{ color: 'var(--ac2)' }}>
                                  {c.name}{idx < groupContacts.length - 1 ? ',' : ''}
                                </span>
                              ))}
                            </div>
                          )}
                          {header.groupStateText && (
                            <span className={header.groupStateText.toLowerCase().includes('трив') || header.groupStateText.toLowerCase().includes('відключ') ? 'chip chip-red' : 'bright'}>
                              {header.groupStateText}
                            </span>
                          )}
                        </div>
                      </div>
                    </td>
                  </tr>
                )
              }
              const zone = row.item
              return (
                <tr key={row.id}>
                  <td className={compact ? undefined : 'mono bright'}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, paddingLeft: 10 }}>
                      <span style={{ color: 'var(--bd)' }}>└</span>
                      <span>{zone.number}</span>
                    </div>
                  </td>
                  {!compact && <td className="dim" style={{ textAlign: 'center' }}>1</td>}
                  <td>{zone.name || '—'}</td>
                  <td>{zone.sensorType || '—'}</td>
                  <td>
                    <span className={
                      zone.status.toLowerCase().includes('трив') ? 'chip chip-red'
                      : zone.status.toLowerCase().includes('несправ') ? 'chip chip-orange'
                      : 'chip chip-green'
                    }>
                      {zone.status || 'НОРМА'}
                    </span>
                  </td>
                  {!compact && <td className="dim">—</td>}
                </tr>
              )
            })}
            <SpacerRow colSpan={colSpan} height={virtualRows.bottomPaddingPx} />
            {rows.length === 0 && <tr><td colSpan={colSpan}>{emptyText}</td></tr>}
          </tbody>
        </table>
        <LoadStatus loadedCount={virtualRows.loadedCount} totalCount={virtualRows.totalCount} />
      </div>
    </div>
  )
}

function ResponsePane({ virtualRows, phone }: { virtualRows: ReturnType<typeof useVirtualRows<string>>; phone: string }) {
  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th>Пріор.</th>
              <th>Назва групи</th>
              <th>Позивний</th>
              <th>Телефон</th>
              <th>Тип виїзду</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={5} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((groupName, idx) => (
              <tr key={groupName}>
                <td className="bright">{virtualRows.startIndex + idx + 1}</td>
                <td>{groupName}</td>
                <td className="dim">{groupName}-01</td>
                <td className="mono dim">{phone}</td>
                <td className="dim">Виїзд</td>
              </tr>
            ))}
            <SpacerRow colSpan={5} height={virtualRows.bottomPaddingPx} />
          </tbody>
        </table>
        <LoadStatus loadedCount={virtualRows.loadedCount} totalCount={virtualRows.totalCount} />
      </div>
    </div>
  )
}

function KeysPane({ virtualRows, phone }: { virtualRows: ReturnType<typeof useVirtualRows<string>>; phone: string }) {
  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th>Код</th>
              <th>Власник</th>
              <th>Телефон</th>
              <th>Тип доступу</th>
              <th>Стан</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={5} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((name, idx) => (
              <tr key={name}>
                <td className="mono bright">{`10${pad2(virtualRows.startIndex + idx + 1)}`}</td>
                <td>{name}</td>
                <td className="mono dim">{phone}</td>
                <td className="dim">Взяття/здача</td>
                <td><span className="chip chip-green">НОРМА</span></td>
              </tr>
            ))}
            <SpacerRow colSpan={5} height={virtualRows.bottomPaddingPx} />
          </tbody>
        </table>
        <LoadStatus loadedCount={virtualRows.loadedCount} totalCount={virtualRows.totalCount} />
      </div>
    </div>
  )
}

function ContactsPane({
  virtualRows,
  rows,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<GroupedRow<FrontendContact>>>
  rows: GroupedRow<FrontendContact>[]
}) {
  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th style={{ width: 64 }}>Пріор.</th>
              <th style={{ width: 260 }}>ПІБ</th>
              <th style={{ width: 140 }}>Телефон</th>
              <th>Посада</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={4} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((row) => {
              if (row.type === 'header') {
                return (
                  <tr key={row.id}>
                    <td colSpan={4} className="bright" style={{ background: 'var(--bg3)' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingLeft: 4 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <span style={{ color: 'var(--tx3)', fontSize: 13 }}>▾</span>
                          <span>{row.groupNumber > 0 ? `Група ${row.groupNumber}: ` : ''}{row.groupName}</span>
                        </div>
                        {row.groupStateText && (
                          <span className={row.groupStateText.toLowerCase().includes('трив') || row.groupStateText.toLowerCase().includes('відключ') ? 'dim' : 'bright'}>
                            [{row.groupStateText}]
                          </span>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              }
              const contact = row.item
              return (
                <tr key={row.id}>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, paddingLeft: 10 }}>
                      <span style={{ color: 'var(--border)' }}>└</span>
                      <span>{contact.priority}</span>
                    </div>
                  </td>
                  <td>{contact.name}</td>
                  <td>{contact.phone || '—'}</td>
                  <td>{contact.position || '—'}</td>
                </tr>
              )
            })}
            <SpacerRow colSpan={4} height={virtualRows.bottomPaddingPx} />
            {rows.length === 0 && <tr><td colSpan={4}>Дані відсутні</td></tr>}
          </tbody>
        </table>
        <LoadStatus loadedCount={virtualRows.loadedCount} totalCount={virtualRows.totalCount} />
      </div>
    </div>
  )
}

function EventsPane({
  virtualRows,
  rows,
  feed,
  onScroll,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<JournalRow>>
  rows: JournalRow[]
  feed: EventModalProps['objectEventsFeed']
  onScroll: (event: UIEvent<HTMLDivElement>) => void
}) {
  const { columnVisibility, toggleColumn, resetAll } = useColumnVisibility('obj-events')

  const allColumns = useMemo(() => [
    { id: 'date', label: 'Дата' },
    { id: 'time', label: 'Час' },
    { id: 'typeText', label: 'Тип події' },
    { id: 'line', label: 'Лінія' },
    { id: 'code', label: 'Код' },
    { id: 'details', label: 'Опис' },
  ], [])

  const toggleableColumns = useMemo(
    () => allColumns.map((col) => ({ ...col, isVisible: columnVisibility[col.id] !== false })),
    [allColumns, columnVisibility],
  )

  const visibleSet = useMemo(() => {
    const set = new Set<string>()
    for (const col of allColumns) {
      if (columnVisibility[col.id] !== false) set.add(col.id)
    }
    return set
  }, [allColumns, columnVisibility])

  const visibleColCount = visibleSet.size

  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              {visibleSet.has('date') && (
                <th style={{ width: 88 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <ColumnVisibilityButton columns={toggleableColumns} onToggle={toggleColumn} onReset={resetAll} />
                    Дата
                  </div>
                </th>
              )}
              {visibleSet.has('time') && (
                <th style={{ width: 80 }}>
                  {!visibleSet.has('date') && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <ColumnVisibilityButton columns={toggleableColumns} onToggle={toggleColumn} onReset={resetAll} />
                      Час
                    </div>
                  )}
                  {visibleSet.has('date') && 'Час'}
                </th>
              )}
              {visibleSet.has('typeText') && (
                <th style={{ width: 160 }}>
                  {!visibleSet.has('date') && !visibleSet.has('time') && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <ColumnVisibilityButton columns={toggleableColumns} onToggle={toggleColumn} onReset={resetAll} />
                      Тип події
                    </div>
                  )}
                  {(visibleSet.has('date') || visibleSet.has('time')) && 'Тип події'}
                </th>
              )}
              {visibleSet.has('line') && <th style={{ width: 52 }}>Лінія</th>}
              {visibleSet.has('code') && <th style={{ width: 68 }}>Код</th>}
              {visibleSet.has('details') && <th>Опис</th>}
            </tr>
          </thead>
          <tbody>
            {rows.map((item) => (
              <tr key={`event-modal-${item.rowID}`}>
                {visibleSet.has('date') && <td>{item.date}</td>}
                {visibleSet.has('time') && <td>{item.time}</td>}
                {visibleSet.has('typeText') && <td className={resolveJournalTypeClass(item)}>{item.typeText}</td>}
                {visibleSet.has('line') && <td className="mono dim">{item.line}</td>}
                {visibleSet.has('code') && <td className="mono dim">{item.code}</td>}
                {visibleSet.has('details') && <td className="dim">{item.details}</td>}
              </tr>
            ))}
            {feed.isInitialLoading && virtualRows.totalCount === 0 && (
              <tr><td colSpan={visibleColCount}>Завантаження подій...</td></tr>
            )}
            {!feed.isInitialLoading && rows.length === 0 && (
              <tr><td colSpan={visibleColCount}>Подій для об'єкта не знайдено</td></tr>
            )}
          </tbody>
        </table>
        {(virtualRows.loadedCount < virtualRows.totalCount || feed.hasMore || feed.isLoadingMore) && (
          <div className="table-load-status">
            {virtualRows.loadedCount < virtualRows.totalCount
              ? `Показано ${virtualRows.loadedCount} з ${virtualRows.totalCount}. Прокрутіть вниз для підвантаження.`
              : feed.isLoadingMore
                ? 'Завантаження наступної сторінки подій...'
                : `Показано ${rows.length} з ${Math.max(feed.totalCount, rows.length)}. Прокрутіть вниз для підвантаження.`}
          </div>
        )}
      </div>
    </div>
  )
}

function PhotoPane() {
  return (
    <div className="modal-pane active">
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: 12, color: 'var(--tx2)' }}>
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" opacity=".3">
          <path d="M23 19a2 2 0 01-2 2H3a2 2 0 01-2-2V8a2 2 0 012-2h4l2-3h6l2 3h4a2 2 0 012 2z" />
          <circle cx="12" cy="13" r="4" />
        </svg>
        <span style={{ fontSize: 12 }}>Фото відсутні</span>
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>+ Додати</button>
      </div>
    </div>
  )
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function InputRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="irow">
      <label>{label}</label>
      <input value={value} readOnly className={mono ? 'mono' : undefined} />
    </div>
  )
}

function GuardChip({ status }: { status: GuardStatus }) {
  if (status === 'guarded') return <span className="chip chip-green">ОХОРОНЯЄТЬСЯ</span>
  if (status === 'disarmed') return <span className="chip chip-gray">ЗНЯТО</span>
  return <span className="chip chip-gray">—</span>
}

function ConnectionChip({ status }: { status: ConnectionStatus }) {
  if (status === 'online') return <span className="chip chip-green">ONLINE</span>
  if (status === 'offline') return <span className="chip chip-red">OFFLINE</span>
  return <span className="chip chip-gray">—</span>
}

function MonitoringChip({ status }: { status: MonitoringStatus }) {
  if (status === 'active') return <span className="chip chip-green">АКТИВНИЙ</span>
  if (status === 'blocked') return <span className="chip chip-orange">ЗАБЛОКОВАНИЙ</span>
  if (status === 'debug') return <span className="chip chip-orange">СТЕНДИ</span>
  return <span className="chip chip-gray">—</span>
}

function deviceTypeLabel(source: FrontendSource): string {
  return source === 'bridge' ? 'ППК' : 'Тип приладу'
}

function panelMarkLabel(source: FrontendSource): string {
  if (source === 'casl') return 'Номер ППК'
  if (source === 'bridge') return 'Прилад'
  return 'Ідентифікатор'
}

function sim1Label(source: FrontendSource): string {
  return source === 'phoenix' ? 'Телефон SIM1' : 'SIM 1'
}

function sim2Label(source: FrontendSource): string {
  return source === 'phoenix' ? 'Телефон SIM2' : 'SIM 2'
}

function HeaderField({ label, value, alarm = false }: { label: string; value: string; alarm?: boolean }) {
  return (
    <div className="hdr-field">
      <label>{label}</label>
      <span className="v" style={alarm ? { color: 'var(--ac4)' } : undefined}>{value}</span>
    </div>
  )
}

function SpacerRow({ colSpan, height }: { colSpan: number; height: number }) {
  if (height <= 0) return null
  return <tr className="vt-spacer" aria-hidden><td colSpan={colSpan} style={{ height }} /></tr>
}

function LoadStatus({ loadedCount, totalCount }: { loadedCount: number; totalCount: number }) {
  if (loadedCount >= totalCount) return null
  return (
    <div className="table-load-status">
      Показано {loadedCount} з {totalCount}. Прокрутіть вниз для підвантаження.
    </div>
  )
}
