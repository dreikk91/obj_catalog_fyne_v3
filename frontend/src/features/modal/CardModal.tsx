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

type GroupedRowHeader = { type: 'header'; groupName: string; groupNumber: number; groupStateText: string; groupID: string; id: string }
type GroupedRowItem<T> = { type: 'item'; item: T; id: string }
type GroupedRow<T> = GroupedRowHeader | GroupedRowItem<T>

function useGroupedRows<T extends { groupName: string; groupNumber: number; groupStateText: string; groupID: string; number?: number; name?: string; phone?: string }>(items: T[]): GroupedRow<T>[] {
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
      if (groupItems.length === 0) continue;
      const ref = groupItems[0]
      const gname = ref.groupName || 'Без групи'
      grouped.push({ type: 'header', groupName: gname, groupNumber: ref.groupNumber, groupStateText: ref.groupStateText, groupID: ref.groupID, id: `header-${key}` })
      for (let i = 0; i < groupItems.length; i++) {
        const item = groupItems[i]
        grouped.push({ type: 'item', item, id: `item-${key}-${i}` })
      }
    }
    return grouped
  }, [items])
}

type CardModalProps = {
  isOpen: boolean
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
  tab: ModalTab
  onSelectTab: (tab: ModalTab) => void
  onClose: () => void
}

export function CardModal({
  isOpen,
  selectedObjectRow,
  selectedObjectZones,
  selectedObjectContacts,
  selectedObjectEvents,
  objectEventsFeed,
  tab,
  onSelectTab,
  onClose,
}: CardModalProps) {
  const responseRows = BASE_GROUP_NAMES
  const keyOwners = BASE_KEY_OWNERS

  const devicesVirtual = useVirtualRows(CARD_DEVICE_ROWS, { rowHeight: 28, initialCount: 80, step: 80 })
  const groupedZones = useGroupedRows(selectedObjectZones)
  const zonesVirtual = useVirtualRows(groupedZones, { rowHeight: 28, initialCount: 120, step: 120 })
  const responseVirtual = useVirtualRows(responseRows, { rowHeight: 28, initialCount: 80, step: 80 })
  const keysVirtual = useVirtualRows(keyOwners, { rowHeight: 28, initialCount: 80, step: 80 })
  const groupedContacts = useGroupedRows(selectedObjectContacts)
  const contactsVirtual = useVirtualRows(groupedContacts, { rowHeight: 28, initialCount: 120, step: 120 })

  // Події використовуються в тому порядку, в якому їх надає бекенд (вже відсортовані)
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
      <div className="modal">
        <div className="modal-tb">
          <div className="modal-tb-icon">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="3">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </div>
          <span className="modal-tb-title">Картка об'єкта —</span>
          <span className="modal-tb-obj">
            № {selectedObjectRow?.number ?? '—'} · {selectedObjectRow?.name ?? '—'}
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
          <div className="obj-num-badge">{selectedObjectRow?.number ?? '—'}</div>
          <HeaderField label="Назва:" value={selectedObjectRow?.name ?? '—'} />
          <HeaderField label="Адреса:" value={selectedObjectRow?.address ?? '—'} />
          <HeaderField label="Група:" value={selectedObjectRow?.group ?? '—'} mono />
          <div className={selectedObjectRow?.statusKey === 'alarm' ? 'obj-hdr-status obj-hdr-status--alarm' : 'obj-hdr-status'}>
            <div className="chip-dot" />
            <span>{selectedObjectRow?.statusLabel ?? '—'}</span>
          </div>
        </div>
        <div className="modal-content">
          {tab === 'kartochka' && <CardSummaryPane selectedObjectRow={selectedObjectRow} />}
          {tab === 'devices' && <DevicesPane virtualRows={devicesVirtual} />}
          {tab === 'zones' && <ZonesPane virtualRows={zonesVirtual} rows={groupedZones} contacts={selectedObjectContacts} emptyText="Дані відсутні" compact />}
          {tab === 'response' && <ResponsePane virtualRows={responseVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'keys' && <KeysPane virtualRows={keysVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'resp' && <ContactsPane virtualRows={contactsVirtual} rows={groupedContacts} />}
          {tab === 'photo' && <PhotoPane />}
          {tab === 'events_tab' && (
            <ObjectEventsPane virtualRows={eventsVirtual} rows={selectedObjectEvents} feed={objectEventsFeed} onScroll={handleEventsScroll} />
          )}
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

function CardSummaryPane({ selectedObjectRow }: { selectedObjectRow: ObjectRow | null }) {
  return (
    <div className="modal-pane active">
      <div className="igrid">
        <div className="isection">
          <div className="isect-title">Загальні відомості</div>
          <InputRow label="Номер об'єкта" value={selectedObjectRow?.number ?? '—'} />
          <InputRow label="Повна назва" value={selectedObjectRow?.name ?? '—'} />
          <InputRow label="Адреса" value={selectedObjectRow?.address ?? '—'} />
          <InputRow label="Примітка" value={selectedObjectRow?.note ?? '—'} />
        </div>
        <div className="isection">
          <div className="isect-title">Параметри охорони</div>
          <InputRow label="Група" value={selectedObjectRow?.group ?? '—'} />
          <InputRow label="Договір" value={selectedObjectRow?.contract ?? '—'} />
          <InputRow label="Час взяття" value={selectedObjectRow?.lastEventAt ?? '—'} />
          <InputRow label="Час здачі" value={selectedObjectRow?.lastTestAt ?? '—'} />
          <InputRow label="Телефон" value={selectedObjectRow?.phone ?? '—'} />
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
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>
          + Додати
        </button>
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
                <td>
                  <span className="chip chip-green">{item.status}</span>
                </td>
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
                          <span>
                            {header.groupNumber > 0 ? `Група ${header.groupNumber}: ` : ''}
                            {header.groupName}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                          {groupContacts.length > 0 && (
                            <div style={{ display: 'flex', gap: 6, fontSize: 11, color: 'var(--tx2)' }}>
                              <span style={{ opacity: 0.7 }}>Відп:</span>
                              {groupContacts.map((c, idx) => (
                                <span key={c.name} style={{ color: 'var(--ac2)' }}>
                                  {c.name}
                                  {idx < groupContacts.length - 1 ? ',' : ''}
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
                  {!compact && <td className="dim">—</td>}
                </tr>
              )
            })}
            <SpacerRow colSpan={colSpan} height={virtualRows.bottomPaddingPx} />
            {rows.length === 0 && (
              <tr>
                <td colSpan={colSpan}>{emptyText}</td>
              </tr>
            )}
          </tbody>
        </table>
        <LoadStatus loadedCount={virtualRows.loadedCount} totalCount={virtualRows.totalCount} />
      </div>
    </div>
  )
}

function ResponsePane({
  virtualRows,
  phone,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<string>>
  phone: string
}) {
  return (
    <div className="modal-pane active">
      <div className="mtable-toolbar">
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>
          + Додати
        </button>
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

function KeysPane({
  virtualRows,
  phone,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<string>>
  phone: string
}) {
  return (
    <div className="modal-pane active">
      <div className="mtable-toolbar">
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>
          + Додати
        </button>
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
                <td>
                  <span className="chip chip-green">НОРМА</span>
                </td>
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
                          <span>
                            {row.groupNumber > 0 ? `Група ${row.groupNumber}: ` : ''}
                            {row.groupName}
                          </span>
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

function ObjectEventsPane({
  virtualRows,
  rows,
  feed,
  onScroll,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<JournalRow>>
  rows: JournalRow[]
  feed: CardModalProps['objectEventsFeed']
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
              {visibleSet.has('code') && (
                <th style={visibleSet.has('details') ? { width: 68 } : undefined}>
                  Код
                </th>
              )}
              {visibleSet.has('details') && <th>Опис</th>}
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={visibleColCount} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((item) => (
              <tr key={item.rowID}>
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
        <button className="btn btn-blue" style={{ height: 24, fontSize: 11 }}>
          + Додати
        </button>
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

function HeaderField({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="hdr-field">
      <label>{label}</label>
      <span className={mono ? 'v mono' : 'v'}>{value}</span>
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