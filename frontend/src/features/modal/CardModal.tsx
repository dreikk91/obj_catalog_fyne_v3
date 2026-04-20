import { useCallback } from 'react'
import type { UIEvent } from 'react'
import type { FrontendContact, FrontendZone } from '../../shared/api/types'
import type { ModalTab } from '../../shared/state/ui-store'
import { useVirtualRows } from '../../hooks/useVirtualRows'
import { BASE_GROUP_NAMES, BASE_KEY_OWNERS, CARD_DEVICE_ROWS, MODAL_TABS } from '../operator/constants'
import type { JournalRow, ObjectRow } from '../operator/types'
import { pad2 } from '../operator/utils'

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
  const zonesVirtual = useVirtualRows(selectedObjectZones, { rowHeight: 28, initialCount: 120, step: 120 })
  const responseVirtual = useVirtualRows(responseRows, { rowHeight: 28, initialCount: 80, step: 80 })
  const keysVirtual = useVirtualRows(keyOwners, { rowHeight: 28, initialCount: 80, step: 80 })
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
          {tab === 'zones' && <ZonesPane virtualRows={zonesVirtual} rows={selectedObjectZones} emptyText="Дані відсутні" compact />}
          {tab === 'response' && <ResponsePane virtualRows={responseVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'keys' && <KeysPane virtualRows={keysVirtual} phone={selectedObjectRow?.phone ?? '—'} />}
          {tab === 'resp' && <ContactsPane virtualRows={contactsVirtual} rows={selectedObjectContacts} />}
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
  emptyText,
  compact = false,
}: {
  virtualRows: ReturnType<typeof useVirtualRows<FrontendZone>>
  rows: FrontendZone[]
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
                <th>№</th>
                <th>Назва зони</th>
                <th>Тип датчика</th>
                <th>Стан</th>
              </tr>
            ) : (
              <tr>
                <th style={{ width: 36 }}>№</th>
                <th style={{ width: 40 }}>Прил.</th>
                <th>Назва зони</th>
                <th style={{ width: 130 }}>Тип датчика</th>
                <th style={{ width: 110 }}>Стан</th>
                <th style={{ width: 70 }}>Обхід</th>
              </tr>
            )}
          </thead>
          <tbody>
            <SpacerRow colSpan={colSpan} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((zone) => (
              <tr key={`${zone.number}-${zone.name}`}>
                <td className={compact ? undefined : 'mono bright'} style={compact ? undefined : { textAlign: 'center' }}>
                  {zone.number}
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
            ))}
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
  virtualRows: ReturnType<typeof useVirtualRows<FrontendContact>>
  rows: FrontendContact[]
}) {
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
              <tr key={`${contact.name}-${contact.phone}`}>
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
  return (
    <div className="modal-pane active">
      <div className="mtable-wrap" ref={virtualRows.containerRef} onScroll={onScroll}>
        <table className="mtable">
          <thead>
            <tr>
              <th>Дата</th>
              <th>Час</th>
              <th>Тип події</th>
              <th>Лінія</th>
              <th>Код</th>
              <th>Опис</th>
            </tr>
          </thead>
          <tbody>
            <SpacerRow colSpan={6} height={virtualRows.topPaddingPx} />
            {virtualRows.visibleRows.map((item) => (
              <tr key={item.rowID}>
                <td>{item.date}</td>
                <td>{item.time}</td>
                <td className={item.alarm ? 'red' : ''}>{item.typeText}</td>
                <td>{item.line}</td>
                <td>{item.code}</td>
                <td>{item.details}</td>
              </tr>
            ))}
            <SpacerRow colSpan={6} height={virtualRows.bottomPaddingPx} />
            {feed.isInitialLoading && virtualRows.totalCount === 0 && (
              <tr>
                <td colSpan={6}>Завантаження подій...</td>
              </tr>
            )}
            {!feed.isInitialLoading && rows.length === 0 && (
              <tr>
                <td colSpan={6}>Подій для об'єкта не знайдено</td>
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
