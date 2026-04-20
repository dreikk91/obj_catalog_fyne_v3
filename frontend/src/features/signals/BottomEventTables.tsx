import { useMemo } from 'react'
import { flexRender, getCoreRowModel, useReactTable, type Cell, type ColumnDef, type Row, type Table } from '@tanstack/react-table'
import type { BottomTab } from '../../shared/state/ui-store'
import { useVirtualRows } from '../../hooks/useVirtualRows'
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
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'objectNumber',
        header: "Об'єкт",
        size: 74,
        minSize: 56,
        cell: ({ getValue }) => <span className="mono bright">{String(getValue())}</span>,
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
                  title={isExpanded ? 'Згорнути події' : 'Розгорнути події'}
                >
                  {isExpanded ? '▾' : '▸'}
                </button>
              )}
              {rowMeta.isChild && <span className="group-row-indent" />}
              <span className={typeClass}>{value}</span>
              {isExpandableParent && <span className="dim group-size-label">+{rowMeta.groupSize - 1}</span>}
            </span>
          )
        },
      },
      {
        accessorKey: 'date',
        header: 'Дата',
        size: 76,
        minSize: 68,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'time',
        header: 'Час',
        size: 76,
        minSize: 68,
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
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
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

  const unprocessedTable = useReactTable({
    data: unprocessedFlatRows,
    columns: journalColumns,
    enableColumnResizing: true,
    columnResizeMode: 'onChange',
    getCoreRowModel: getCoreRowModel(),
  })

  const archiveTable = useReactTable({
    data: journalArchiveRows,
    columns: journalColumns,
    enableColumnResizing: true,
    columnResizeMode: 'onChange',
    getCoreRowModel: getCoreRowModel(),
  })

  const unprocessedVirtual = useVirtualRows(unprocessedTable.getRowModel().rows, { rowHeight: 28, initialCount: 220, step: 220 })
  const archiveVirtual = useVirtualRows(archiveTable.getRowModel().rows, { rowHeight: 28, initialCount: 220, step: 220 })
  const columnCount = unprocessedTable.getAllLeafColumns().length

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
          <div className="bot-toolbar">
            <button
              className="btn btn-violet"
              style={{ height: 24, fontSize: 11 }}
              disabled={selectedUnprocessedRow == null}
              onClick={() => selectedUnprocessedRow != null && onOpenEventModal(selectedUnprocessedRow)}
            >
              Взяти в роботу
            </button>
            <button
              className={showAllAlarms ? 'btn btn-blue' : 'btn btn-gray'}
              style={{ height: 24, fontSize: 11 }}
              onClick={onToggleShowAll}
            >
              {showAllAlarms ? '● Всі оператори' : 'Показати всі'}
            </button>
          </div>
        )}
      </div>

      <JournalPane
        active={bottomTab === 'unproc'}
        table={unprocessedTable}
        virtualRows={unprocessedVirtual}
        columnCount={columnCount}
        selectedSignalRowID={selectedSignalRowID}
        onSelectSignalRow={onSelectSignalRow}
        onDoubleClickRow={onOpenEventModal}
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
        columnCount={columnCount}
        selectedSignalRowID={selectedSignalRowID}
        onSelectSignalRow={onSelectSignalRow}
        onDoubleClickRow={onOpenCardModal}
        rowClassName={(row) => resolveJournalRowClass(row, selectedSignalRowID === row.rowID)}
      />
    </div>
  )
}

function JournalPane({
  active,
  table,
  virtualRows,
  columnCount,
  selectedSignalRowID,
  onSelectSignalRow,
  onDoubleClickRow,
  rowClassName,
}: {
  active: boolean
  table: Table<JournalRow>
  virtualRows: ReturnType<typeof useVirtualRows<Row<JournalRow>>>
  columnCount: number
  selectedSignalRowID: string | null
  onSelectSignalRow: (row: JournalRow) => void
  onDoubleClickRow: (row: JournalRow) => void
  rowClassName: (row: JournalRow) => string
}) {
  return (
    <div className={active ? 'bot-pane active' : 'bot-pane'}>
      <div style={{ flex: 1, overflow: 'auto' }} ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="evt-table">
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  const meta = header.column.columnDef.meta as TableColumnMeta | undefined
                  const isFluid = meta?.fluid === true
                  return (
                    <th
                      key={header.id}
                      className={isFluid ? 'col-fluid' : undefined}
                      style={
                        isFluid
                          ? { minWidth: meta?.minWidth }
                          : { width: header.getSize(), minWidth: header.column.columnDef.minSize }
                      }
                    >
                      {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
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
            {virtualRows.topPaddingPx > 0 && (
              <tr className="vt-spacer" aria-hidden>
                <td colSpan={columnCount} style={{ height: virtualRows.topPaddingPx }} />
              </tr>
            )}
            {virtualRows.visibleRows.map((tableRow) => (
              <tr
                key={tableRow.id}
                className={rowClassName(tableRow.original)}
                onClick={() => onSelectSignalRow(tableRow.original)}
                onDoubleClick={() => onDoubleClickRow(tableRow.original)}
                title="Двічі клацніть для відкриття"
                data-selected={selectedSignalRowID === tableRow.original.rowID}
              >
                {tableRow.getVisibleCells().map((cell: Cell<JournalRow, unknown>) => (
                  <td
                    key={cell.id}
                    className={(cell.column.columnDef.meta as TableColumnMeta | undefined)?.fluid ? 'col-fluid' : undefined}
                    style={
                      (cell.column.columnDef.meta as TableColumnMeta | undefined)?.fluid
                        ? { minWidth: (cell.column.columnDef.meta as TableColumnMeta | undefined)?.minWidth }
                        : { width: cell.column.getSize(), minWidth: cell.column.columnDef.minSize }
                    }
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
            {virtualRows.bottomPaddingPx > 0 && (
              <tr className="vt-spacer" aria-hidden>
                <td colSpan={columnCount} style={{ height: virtualRows.bottomPaddingPx }} />
              </tr>
            )}
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
