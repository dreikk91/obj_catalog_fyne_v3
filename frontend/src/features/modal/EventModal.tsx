import { useCallback, useMemo } from 'react'
import type { UIEvent } from 'react'
import type { FrontendContact, FrontendZone } from '../../shared/api/types'
import type { ModalTab } from '../../shared/state/ui-store'
import { useVirtualRows } from '../../hooks/useVirtualRows'
import { useColumnVisibility } from '../../hooks/useColumnVisibility'
import { ColumnVisibilityButton } from '../../shared/ui/ColumnVisibilityButton'
import { BASE_GROUP_NAMES, BASE_KEY_OWNERS, CARD_DEVICE_ROWS, MODAL_TABS } from '../operator/constants'
import type { JournalRow, ObjectRow } from '../operator/types'
import { pad2, resolveJournalTypeClass } from '../operator/utils'

type EventModalProps = {
  isOpen: boolean
  tab: ModalTab
  onSelectTab: (tab: ModalTab) => void
  onClose: () => void
  eventModalRow: JournalRow | null
  selectedObjectRow: ObjectRow | null
  selectedObjectZones: FrontendZone[]
  selectedObjectContacts: FrontendContact[]
  selectedObjectEvents: JournalRow[]
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
  selectedObjectZones,
  selectedObjectContacts,
  selectedObjectEvents,
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
  const eventDeviceRows = useMemo(
    () => [
      {
        type: 'ГПО',
        model: selectedObjectRow?.note || 'Орлан-GPRS v3',
        serial: 'ORL-2023-00451',
        version: '3.14.2',
        channel: 'GPRS',
        status: 'НОРМА',
      },
      CARD_DEVICE_ROWS[1],
    ],
    [selectedObjectRow?.note],
  )

  const devicesVirtual = useVirtualRows(eventDeviceRows, { rowHeight: 28, initialCount: 80, step: 80 })
  const zonesVirtual = useVirtualRows(selectedObjectZones, { rowHeight: 28, initialCount: 120, step: 120 })
  const responseVirtual = useVirtualRows(BASE_GROUP_NAMES, { rowHeight: 28, initialCount: 80, step: 80 })
  const keysVirtual = useVirtualRows(BASE_KEY_OWNERS, { rowHeight: 28, initialCount: 80, step: 80 })
  const contactsVirtual = useVirtualRows(selectedObjectContacts, { rowHeight: 28, initialCount: 120, step: 120 })
  const eventsVirtual = useVirtualRows(selectedObjectEvents, { rowHeight: 28, initialCount: 160, step: 160 })

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
          <div className="modal-tb-close" onClick={onClose}>
            ✕
          </div>
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
          {tab === 'kartochka' && <EventSummaryPane eventModalRow={eventModalRow} />}
          {tab === 'devices' && <DevicesPane virtualRows={devicesVirtual} />}
          {tab === 'zones' && <ZonesPane virtualRows={zonesVirtual} rows={selectedObjectZones} />}
          {tab === 'response' && <ResponsePane virtualRows={responseVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'keys' && <KeysPane virtualRows={keysVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'resp' && <ContactsPane virtualRows={contactsVirtual} rows={selectedObjectContacts} />}
          {tab === 'photo' && <PhotoPane />}
          {tab === 'events_tab' && (
            <EventsPane virtualRows={eventsVirtual} rows={selectedObjectEvents} feed={objectEventsFeed} onScroll={handleEventsScroll} />
          )}
        </div>
        <div className="proc-box">
          <div className="proc-hdr">Обробка події</div>
          {workflowError !== '' && <div className="proc-error">{workflowError}</div>}
          <div className="proc-btns">
            <button
              className="btn btn-violet"
              style={{ height: 30 }}
              onClick={onPickAlarm}
              disabled={workflowBusy || eventModalRow?.alarmID == null || isInWorkflow}
            >
              {eventModalRow?.inProgress && !isInWorkflow ? 'Перехопити тривогу' : isInWorkflow ? 'У вас в роботі' : 'Взяти в роботу'}
            </button>
            <button
              className="btn btn-gray"
              style={{ height: 30 }}
              onClick={onStandby}
              disabled={workflowBusy || !isInWorkflow}
            >
              В стенди
            </button>
            <button
              className="btn btn-gray"
              style={{ height: 30 }}
              onClick={onCancelAlarm}
              disabled={workflowBusy || !isInWorkflow}
            >
              Відміна тривоги
            </button>
            <button
              className="btn btn-green"
              style={{ height: 30 }}
              onClick={onDispatchGroup}
              disabled={workflowBusy || !isInWorkflow}
            >
              Вислати групу
            </button>
            <button
              className="btn btn-gray"
              style={{ height: 30 }}
              onClick={onGroupAction}
              disabled={workflowBusy || !groupDispatched}
            >
              {groupArrived ? 'Зняти групу з тривоги' : 'Група прибула'}
            </button>
            <button
              className="btn btn-green"
              style={{ height: 30 }}
              onClick={onOpenProcessAlarm}
              disabled={workflowBusy || eventModalRow?.alarmID == null || !isInWorkflow}
            >
              Завершити тривогу
            </button>
          </div>
        </div>
        <div className="modal-footer">
          <div style={{ marginLeft: 'auto' }} />
          <button className="btn btn-red" style={{ width: 140, height: 28 }} onClick={onClose}>
            ЗАКРИТИ
          </button>
        </div>
      </div>
    </div>
  )
}

function EventSummaryPane({ eventModalRow }: { eventModalRow: JournalRow | null }) {
  return (
    <div className="modal-pane active">
      <div className="igrid">
        <div className="isection">
          <div className="isect-title">Подія</div>
          <InputRow label="Лінія" value={eventModalRow?.line ?? '—'} />
          <InputRow label="Код" value={eventModalRow?.code ?? '—'} />
          <InputRow label="Дата" value={eventModalRow?.date ?? '—'} />
          <InputRow label="Час" value={eventModalRow?.time ?? '—'} />
        </div>
        <div className="isection">
          <div className="isect-title">Деталі</div>
          <InputRow label="Опис" value={eventModalRow?.details ?? '—'} />
          <InputRow label="Група" value={eventModalRow?.group ?? '—'} />
          <InputRow label="Зона" value={eventModalRow?.zone ?? '—'} />
          <InputRow label="Стан" value={eventModalRow?.state ?? '—'} />
          <InputRow label="Обробляє" value={eventModalRow?.inProgress ? eventModalRow.inProgressBy || (eventModalRow.inProgressByMe ? 'Ви' : 'Інший оператор') : '—'} />
        </div>
      </div>
    </div>
  )
}

function DevicesPane({
  virtualRows,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<(typeof CARD_DEVICE_ROWS)[number]>>
}) {
  return (
    <div className="modal-pane active">
      <div className="mtable-toolbar">
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>+ Додати</button>
      </div>
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th>Тип</th>
              <th>Модель</th>
              <th>Серійний №</th>
              <th>Версія ПЗ</th>
              <th>Канал</th>
              <th>Стан</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={6} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((item) => (
              <tr key={`${item.type}-${item.serial}`}>
                <td className="bright">{item.type}</td>
                <td>{item.model}</td>
                <td className="mono dim">{item.serial}</td>
                <td className="mono dim">{item.version}</td>
                <td>{item.channel}</td>
                <td><span className="chip chip-green">{item.status}</span></td>
              </tr>
            ))}
            <SpacerRow colSpan={6} height={virtualRows.bottomPaddingPx} />
          </tbody>
        </table>
        <LoadStatus loadedCount={virtualRows.loadedCount} totalCount={virtualRows.totalCount} />
      </div>
    </div>
  )
}

function ZonesPane({ virtualRows, rows }: { virtualRows: ReturnType<typeof useVirtualRows<FrontendZone>>; rows: FrontendZone[] }) {
  return (
    <div className="modal-pane active">
      <div className="mtable-toolbar">
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>+ Додати</button>
      </div>
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th style={{ width: 36 }}>№</th>
              <th style={{ width: 40 }}>Прил.</th>
              <th>Назва зони</th>
              <th style={{ width: 130 }}>Тип датчика</th>
              <th style={{ width: 110 }}>Стан</th>
              <th style={{ width: 70 }}>Обхід</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={6} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((zone) => (
              <tr key={`event-zone-${zone.number}-${zone.name}`}>
                <td className="mono bright" style={{ textAlign: 'center' }}>{zone.number}</td>
                <td className="dim" style={{ textAlign: 'center' }}>1</td>
                <td>{zone.name || '—'}</td>
                <td className="dim">{zone.sensorType || '—'}</td>
                <td>
                  <span
                    className={
                      zone.status.toLowerCase().includes('трив')
                        ? 'chip chip-red'
                        : zone.status.toLowerCase().includes('несправ')
                          ? 'chip chip-orange'
                          : 'chip chip-green'
                    }
                  >
                    {zone.status || 'НОРМА'}
                  </span>
                </td>
                <td className="dim">—</td>
              </tr>
            ))}
            <SpacerRow colSpan={6} height={virtualRows.bottomPaddingPx} />
            {rows.length === 0 && (
              <tr>
                <td colSpan={6}>Зони відсутні</td>
              </tr>
            )}
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
      <div className="mtable-toolbar">
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>+ Додати</button>
      </div>
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
            {virtualRows.visibleRows.map((name, idx) => (
              <tr key={name}>
                <td className="bright">{virtualRows.startIndex + idx + 1}</td>
                <td>{name}</td>
                <td className="dim">{name}-01</td>
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
      <div className="mtable-toolbar">
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>+ Додати</button>
      </div>
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

function ContactsPane({ virtualRows, rows }: { virtualRows: ReturnType<typeof useVirtualRows<FrontendContact>>; rows: FrontendContact[] }) {
  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th>Пріор.</th>
              <th>ПІБ</th>
              <th>Телефон</th>
              <th>Посада</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={4} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((contact) => (
              <tr key={`event-${contact.name}-${contact.phone}`}>
                <td>{contact.priority}</td>
                <td>{contact.name}</td>
                <td>{contact.phone || '—'}</td>
                <td>{contact.position || '—'}</td>
              </tr>
            ))}
            <SpacerRow colSpan={4} height={virtualRows.bottomPaddingPx} />
            {rows.length === 0 && (
              <tr>
                <td colSpan={4}>Дані відсутні</td>
              </tr>
            )}
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

  const toggleableColumns = useMemo(() =>
    allColumns.map((col) => ({ ...col, isVisible: columnVisibility[col.id] !== false })),
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
                <th style={{ width: 80 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <ColumnVisibilityButton columns={toggleableColumns} onToggle={toggleColumn} onReset={resetAll} />
                    Дата
                  </div>
                </th>
              )}
              {visibleSet.has('time') && (
                <th style={{ width: 64 }}>
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
            <SpacerRow colSpan={visibleColCount} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((item) => (
              <tr key={`event-modal-${item.rowID}`}>
                {visibleSet.has('date') && <td>{item.date}</td>}
                {visibleSet.has('time') && <td>{item.time}</td>}
                {visibleSet.has('typeText') && <td className={resolveJournalTypeClass(item)}>{item.typeText}</td>}
                {visibleSet.has('line') && <td>{item.line}</td>}
                {visibleSet.has('code') && <td>{item.code}</td>}
                {visibleSet.has('details') && <td>{item.details}</td>}
              </tr>
            ))}
            <SpacerRow colSpan={visibleColCount} height={virtualRows.bottomPaddingPx} />
            {feed.isInitialLoading && virtualRows.totalCount === 0 && (
              <tr>
                <td colSpan={visibleColCount}>Завантаження подій...</td>
              </tr>
            )}
            {!feed.isInitialLoading && rows.length === 0 && (
              <tr>
                <td colSpan={visibleColCount}>Подій для об'єкта не знайдено</td>
              </tr>
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
      <div
        style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexDirection: 'column',
          gap: 12,
          color: 'var(--tx2)',
        }}
      >
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

function InputRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="irow">
      <label>{label}</label>
      <input value={value} readOnly />
    </div>
  )
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
  if (height <= 0) {
    return null
  }
  return (
    <tr className="vt-spacer" aria-hidden>
      <td colSpan={colSpan} style={{ height }} />
    </tr>
  )
}

function LoadStatus({ loadedCount, totalCount }: { loadedCount: number; totalCount: number }) {
  if (loadedCount >= totalCount) {
    return null
  }
  return (
    <div className="table-load-status">
      Показано {loadedCount} з {totalCount}. Прокрутіть вниз для підвантаження.
    </div>
  )
}
