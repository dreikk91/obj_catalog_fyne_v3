import { useMemo } from 'react'
import { flexRender, getCoreRowModel, useReactTable, type ColumnDef, type Row, type Table } from '@tanstack/react-table'
import type { BottomTab } from '../../shared/state/ui-store'
import { useVirtualRows } from '../../hooks/useVirtualRows'
import { useColumnVisibility } from '../../hooks/useColumnVisibility'
import { ColumnVisibilityButton, type ColumnToggleItem } from '../../shared/ui/ColumnVisibilityButton'
import type { JournalRow, TableColumnMeta, UnprocessedAlarmGroup, UnprocessedRowMeta } from '../operator/types'
import {
  resolveJournalIndicatorColor,
  resolveJournalRowClass,
  resolveJournalStateChipClass,
  resolveJournalTypeClass,
} from '../operator/utils'

type BottomEventTablesProps = {
  height?: number
  isResizing?: boolean
  bottomTab: BottomTab
  onSelectBottomTab: (tab: BottomTab) => void
  unprocessedAlarmGroups: UnprocessedAlarmGroup[]
  journalArchiveRows: JournalRow[]
  unprocessedFlatRows: JournalRow[]
  unprocessedRowMetaByID: Map<string, UnprocessedRowMeta>
  expandedUnprocessedGroups: Record<string, boolean>
  showAllAlarms: boolean
  onToggleGroup: (groupID: string) => void
  onToggleShowAll: () => void
  selectedSignalRowID: string | null
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

export function BottomEventTables({
  height,
  isResizing = false,
  bottomTab,
  onSelectBottomTab,
  unprocessedAlarmGroups,
  journalArchiveRows,
  unprocessedFlatRows,
  unprocessedRowMetaByID,
  expandedUnprocessedGroups,
  showAllAlarms,
  onToggleGroup,
  onToggleShowAll,
  selectedSignalRowID,
  onSelectSignalRow,
  onOpenEventModal,
  onOpenCardModal,
  isInWorkflow,
  groupDispatched,
  groupArrived,
  workflowBusy,
  onPickAlarm,
  onStandby,
  onCancelAlarm,
  onDispatchGroup,
  onGroupAction,
  onOpenProcessAlarm,
}: BottomEventTablesProps) {
  const selectedUnprocessedRow = useMemo(
    () => unprocessedFlatRows.find((row) => row.rowID === selectedSignalRowID) ?? null,
    [unprocessedFlatRows, selectedSignalRowID],
  )
  const journalColumns = useMemo<ColumnDef<JournalRow>[]>(() => {
    return [
      {
        id: 'indicator',
        header: '',
        size: 28,
        minSize: 24,
        maxSize: 44,
        cell: ({ row }) => (
          <span
            className="chip-dot evt-dot"
            style={{
              background: resolveJournalIndicatorColor(row.original),
              boxShadow: row.original.alarm || row.original.severity === 'critical' ? '0 0 5px var(--ac5)' : undefined,
              margin: '0 auto',
              display: 'block',
            }}
          />
        ),
      },
      {
        accessorKey: 'line',
        header: 'Лінія',
        size: 56,
        minSize: 42,
        cell: ({ row, getValue }) => {
          const rowMeta = unprocessedRowMetaByID.get(row.original.rowID)
          if (rowMeta?.isChild) {
            return <span className="dim"></span>
          }
          return <span className="dim">{String(getValue())}</span>
        },
      },
      {
        accessorKey: 'objectNumber',
        header: "Об'єкт",
        size: 74,
        minSize: 56,
        cell: ({ row, getValue }) => {
          const rowMeta = unprocessedRowMetaByID.get(row.original.rowID)
          if (rowMeta?.isChild) {
            return <span className="mono bright"></span>
          }
          return <span className="mono bright">{String(getValue())}</span>
        },
      },
      {
        accessorKey: 'code',
        header: 'Код',
        size: 50,
        minSize: 40,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'typeText',
        header: 'Тип коду',
        size: 160,
        minSize: 120,
        cell: ({ row, getValue }) => {
          const value = String(getValue())
          const rowMeta = unprocessedRowMetaByID.get(row.original.rowID)
          const typeClass = resolveJournalTypeClass(row.original)
          if (rowMeta == null) {
            return <span className={typeClass}>{value}</span>
          }

          const isExpandableParent = rowMeta.isParent && rowMeta.groupSize > 1
          const isExpanded = rowMeta.isParent ? expandedUnprocessedGroups[rowMeta.groupID] === true : false
          return (
            <span className="unproc-type-cell">
              {isExpandableParent && (
                <button
                  type="button"
                  className="group-toggle-btn"
                  onClick={(event) => {
                    event.stopPropagation()
                    onToggleGroup(rowMeta.groupID)
                  }}
                  title={isExpanded ? 'Згорнути події обʼєкта' : 'Розгорнути події обʼєкта'}
                  style={{ width: 16, height: 16, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', border: '1px solid currentColor', borderRadius: 2, marginRight: 6, cursor: 'pointer', background: 'transparent', color: 'inherit', padding: 0, fontSize: 14, lineHeight: 1 }}
                >
                                    <svg
                    width="10"
                    height="10"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="3"
                    style={{
                      transform: isExpanded ? 'rotate(180deg)' : 'rotate(90deg)',
                      transition: 'transform 0.15s ease',
                      flexShrink: 0
                    }}
                  >
                    <path d="M18 15l-6-6-6 6" />
                  </svg>
                </button>
              )}
                            {rowMeta.isChild && (
                <span className="tree-branch-symbol" style={{ color: 'var(--tx2)', opacity: 0.5, marginRight: 8, fontSize: 14 }}>
                  {rowMeta.isLastChild ? '└─' : '├─'}
                </span>
              )}
                            <span className={typeClass} style={rowMeta.isChild ? { fontSize: '0.95em' } : undefined}>{value}</span>
              {isExpandableParent && (
                <span className="badge-small" style={{ marginLeft: 6, background: 'rgba(255,255,255,0.06)', borderRadius: 10, padding: '1px 5px', fontSize: 9 }}>
                  {rowMeta.groupSize}
                </span>
              )}
            </span>
          )
        },
      },
      {
        accessorKey: 'date',
        header: 'Дата',
        size: 88,
        minSize: 80,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'time',
        header: 'Час',
        size: 88,
        minSize: 80,
        cell: ({ getValue }) => <span className="mono">{String(getValue())}</span>,
      },
      {
        accessorKey: 'group',
        header: 'Гр.',
        size: 44,
        minSize: 38,
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'zone',
        header: 'Шл.',
        size: 44,
        minSize: 38,
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'objectName',
        header: 'Назва',
        enableResizing: false,
        meta: { fluid: true, minWidth: 180 } satisfies TableColumnMeta,
        cell: ({ row, getValue }) => {
          const rowMeta = unprocessedRowMetaByID.get(row.original.rowID)
          if (rowMeta?.isChild) {
            return <span className="dim"></span>
          }
          return <span className="dim">{String(getValue())}</span>
        },
      },
      {
        accessorKey: 'state',
        header: 'Стан',
        size: 100,
        minSize: 90,
        cell: ({ row, getValue }) => <span className={`chip ${resolveJournalStateChipClass(row.original)}`}>{String(getValue())}</span>,
      },
      {
        accessorKey: 'details',
        header: 'Опис події',
        enableResizing: false,
        meta: { fluid: true, minWidth: 280 } satisfies TableColumnMeta,
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
    ]
  }, [expandedUnprocessedGroups, onToggleGroup, unprocessedRowMetaByID])

  const { columnVisibility, toggleColumn, resetAll: resetColumnVisibility } = useColumnVisibility('journal')

  const unprocessedTable = useReactTable({
    data: unprocessedFlatRows,
    columns: journalColumns,
    enableColumnResizing: true,
    columnResizeMode: 'onChange',
    getCoreRowModel: getCoreRowModel(),
    state: { columnVisibility },
  })

  const archiveTable = useReactTable({
    data: journalArchiveRows,
    columns: journalColumns,
    enableColumnResizing: true,
    columnResizeMode: 'onChange',
    getCoreRowModel: getCoreRowModel(),
    state: { columnVisibility },
  })

  const unprocessedVirtual = useVirtualRows(unprocessedTable.getRowModel().rows, { rowHeight: 28, initialCount: 220, step: 220 })
  const archiveVirtual = useVirtualRows(archiveTable.getRowModel().rows, { rowHeight: 28, initialCount: 220, step: 220 })

  const toggleableColumns: ColumnToggleItem[] = useMemo(() => {
    return journalColumns
      .filter((col) => {
        const id = 'accessorKey' in col ? String(col.accessorKey) : col.id
        return id !== 'indicator'
      })
      .map((col) => {
        const id = 'accessorKey' in col ? String(col.accessorKey) : col.id ?? ''
        const label = typeof col.header === 'string' ? col.header : id
        const isVisible = columnVisibility[id] !== false
        return { id, label, isVisible }
      })
  }, [journalColumns, columnVisibility])
  return (
    <div className={isResizing ? 'ps-bottom is-resizing' : 'ps-bottom'} style={height != null ? { height } : undefined}>
      <div className="bot-tabs">
        <button className={bottomTab === 'unproc' ? 'bttab unproc active' : 'bttab unproc'} onClick={() => onSelectBottomTab('unproc')}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
          </svg>
          Необроблені <span className="badge">{unprocessedAlarmGroups.length}</span>
        </button>
        <button className={bottomTab === 'archive' ? 'bttab archive active' : 'bttab archive'} onClick={() => onSelectBottomTab('archive')}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="21 8 21 21 3 21 3 8" />
            <rect x="1" y="3" width="22" height="5" />
            <line x1="10" y1="12" x2="14" y2="12" />
          </svg>
          Архів <span className="badge">{journalArchiveRows.length}</span>
        </button>
        {bottomTab === 'unproc' && (
          <div className="bot-toolbar" style={{ border: '1px solid var(--bd)', padding: '4px', borderRadius: '4px', position: 'relative' }}>
            <span style={{ position: 'absolute', top: -8, left: 8, background: 'var(--bg)', padding: '0 4px', fontSize: 10, color: 'var(--tx2)' }}>Оброблення події</span>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', width: '100%' }}>
              {!isInWorkflow ? (
                <button
                  className="btn btn-violet"
                  style={{ height: 24, fontSize: 11 }}
                  disabled={selectedUnprocessedRow == null || workflowBusy || groupDispatched}
                  onClick={onPickAlarm}
                >
                  {selectedUnprocessedRow?.inProgress ? 'Перехопити' : 'Обробити'}
                </button>
              ) : (
                <>
                  <button
                    className="btn btn-green"
                    style={{ height: 24, fontSize: 11 }}
                    disabled={workflowBusy || groupDispatched}
                    onClick={onOpenProcessAlarm}
                  >
                    Закінчити оброблення
                  </button>
                  {groupDispatched && (
                    <button
                      className="btn btn-gray"
                      style={{ height: 24, fontSize: 11 }}
                      disabled={workflowBusy}
                      onClick={onGroupAction}
                    >
                      {groupArrived ? 'Зняти групу' : 'Підтвердити прибуття'}
                    </button>
                  )}
                  {!groupDispatched && (
                    <button
                      className="btn btn-violet"
                      style={{ height: 24, fontSize: 11 }}
                      disabled={workflowBusy}
                      onClick={onDispatchGroup}
                    >
                      Вислати групи
                    </button>
                  )}
                  <button
                    className="btn btn-gray"
                    style={{ height: 24, fontSize: 11 }}
                    disabled={workflowBusy}
                    onClick={onStandby}
                  >
                    До стендів
                  </button>
                  <button
                    className="btn btn-gray"
                    style={{ height: 24, fontSize: 11 }}
                    disabled={workflowBusy || groupDispatched}
                    onClick={onCancelAlarm}
                  >
                    Скасувати тривогу
                  </button>
                </>
              )}
              
              <div style={{ marginLeft: 'auto' }} />
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 11, cursor: 'pointer' }}>
                <input type="checkbox" checked={showAllAlarms} onChange={onToggleShowAll} />
                Показати всі
              </label>
            </div>
          </div>
        )}
      </div>

      <JournalPane
        active={bottomTab === 'unproc'}
        table={unprocessedTable}
        virtualRows={unprocessedVirtual}

        selectedSignalRowID={selectedSignalRowID}
        onSelectSignalRow={onSelectSignalRow}
        onDoubleClickRow={onOpenEventModal}
        colVisColumns={toggleableColumns}
        onColVisToggle={toggleColumn}
        onColVisReset={resetColumnVisibility}
        rowClassName={(row) => {
          const rowMeta = unprocessedRowMetaByID.get(row.rowID)
          const isSelected =
            selectedSignalRowID === row.rowID ||
            (rowMeta?.isParent === true && selectedSignalRowID != null && rowMeta.memberRowIDs.includes(selectedSignalRowID))
          const baseClass = resolveJournalRowClass(row, isSelected, true)
          if (rowMeta?.isChild) {
            return `${baseClass} unproc-child-row`
          }
          if (rowMeta?.isParent && rowMeta.groupSize > 1) {
            return `${baseClass} unproc-parent-row`
          }
          return baseClass
        }}
      />

      <JournalPane
        active={bottomTab === 'archive'}
        table={archiveTable}
        virtualRows={archiveVirtual}

        selectedSignalRowID={selectedSignalRowID}
        onSelectSignalRow={onSelectSignalRow}
        onDoubleClickRow={onOpenCardModal}
        colVisColumns={toggleableColumns}
        onColVisToggle={toggleColumn}
        onColVisReset={resetColumnVisibility}
        rowClassName={(row) => resolveJournalRowClass(row, selectedSignalRowID === row.rowID)}
      />
    </div>
  )
}

function JournalPane({
  active,
  table,
  virtualRows,
  selectedSignalRowID,
  onSelectSignalRow,
  onDoubleClickRow,
  colVisColumns,
  onColVisToggle,
  onColVisReset,
  rowClassName,
}: {
  active: boolean
  table: Table<JournalRow>
  virtualRows: ReturnType<typeof useVirtualRows<Row<JournalRow>>>
  selectedSignalRowID: string | null
  onSelectSignalRow: (row: JournalRow) => void
  onDoubleClickRow: (row: JournalRow) => void
  colVisColumns: ColumnToggleItem[]
  onColVisToggle: (id: string) => void
  onColVisReset: () => void
  rowClassName: (row: JournalRow) => string
}) {
  return (
    <div className={active ? 'bot-pane active' : 'bot-pane'}>
      <div style={{ flex: 1, overflow: 'auto' }} ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="evt-table">
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header, idx, allHeaders) => {
                  const meta = header.column.columnDef.meta as TableColumnMeta | undefined
                  const isIndicator = header.column.id === 'indicator'
                  const isLast = idx === allHeaders.length - 1
                  const isFluid = meta?.fluid === true || isLast
                  return (
                    <th
                      key={header.id}
                      className={isFluid ? 'col-fluid' : undefined}
                      style={
                        isFluid
                          ? { width: '100%', minWidth: meta?.minWidth ?? (isIndicator ? undefined : 60) }
                          : { width: header.getSize(), minWidth: header.column.columnDef.minSize }
                      }
                    >
                      {isIndicator
                        ? <ColumnVisibilityButton columns={colVisColumns} onToggle={onColVisToggle} onReset={onColVisReset} />
                        : header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                      {header.column.getCanResize() && (
                        <div
                          className={`col-resizer ${header.column.getIsResizing() ? 'is-resizing' : ''}`}
                          onMouseDown={header.getResizeHandler()}
                          onTouchStart={header.getResizeHandler()}
                          onClick={(event) => event.stopPropagation()}
                        />
                      )}
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((tableRow) => (
              <tr
                key={tableRow.id}
                className={rowClassName(tableRow.original)}
                onClick={() => onSelectSignalRow(tableRow.original)}
                onDoubleClick={() => onDoubleClickRow(tableRow.original)}
                title="Двічі клацніть для відкриття"
                data-selected={selectedSignalRowID === tableRow.original.rowID}
              >
                {tableRow.getVisibleCells().map((cell, idx, allCells) => {
                  const meta = cell.column.columnDef.meta as TableColumnMeta | undefined
                  const isLast = idx === allCells.length - 1
                  const isFluid = meta?.fluid === true || isLast
                  return (
                    <td
                      key={cell.id}
                      className={isFluid ? 'col-fluid' : undefined}
                      style={
                        isFluid
                          ? { width: '100%', minWidth: meta?.minWidth ?? 60 }
                          : { width: cell.column.getSize(), minWidth: cell.column.columnDef.minSize }
                      }
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
        {virtualRows.loadedCount < virtualRows.totalCount && (
          <div className="table-load-status">
            Показано {virtualRows.loadedCount} з {virtualRows.totalCount}. Прокрутіть вниз для підвантаження.
          </div>
        )}
      </div>
    </div>
  )
}
