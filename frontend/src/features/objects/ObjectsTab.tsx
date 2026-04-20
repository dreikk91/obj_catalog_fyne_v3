import { useMemo } from 'react'
import { flexRender, getCoreRowModel, useReactTable, type ColumnDef } from '@tanstack/react-table'
import type { StatusFilter } from '../../shared/state/ui-store'
import { OBJECT_FILTER_TABS } from '../operator/constants'
import type { ObjectRow, TableColumnMeta } from '../operator/types'
import { resolveObjectIndicatorColor } from '../operator/utils'
import { useVirtualRows } from '../../hooks/useVirtualRows'

type ObjectsTabProps = {
  rows: ObjectRow[]
  searchValue: string
  statusFilter: StatusFilter
  selectedObjectID: number | null
  onSearchChange: (value: string) => void
  onStatusFilterChange: (filter: StatusFilter) => void
  onRefresh: () => void
  onSelectObject: (objectID: number) => void
}

export function ObjectsTab({
  rows,
  searchValue,
  statusFilter,
  selectedObjectID,
  onSearchChange,
  onStatusFilterChange,
  onRefresh,
  onSelectObject,
}: ObjectsTabProps) {
  const filteredRows = useMemo(() => {
    const query = searchValue.trim().toLowerCase()
    return rows.filter((item) => {
      if (query !== '') {
        const searchHit =
          item.number.toLowerCase().includes(query) ||
          item.name.toLowerCase().includes(query) ||
          item.address.toLowerCase().includes(query) ||
          item.contract.toLowerCase().includes(query)
        if (!searchHit) {
          return false
        }
      }

      switch (statusFilter) {
        case 'all':
          return true
        case 'guarded':
          return item.statusKey === 'guarded'
        case 'unguarded':
          return item.statusKey === 'unguarded'
        case 'call':
          return item.statusKey === 'call'
        case 'alarm':
          return item.statusKey === 'alarm'
        case 'late':
        case 'banned':
          return false
        default:
          return true
      }
    })
  }, [rows, searchValue, statusFilter])

  const columns = useMemo<ColumnDef<ObjectRow>[]>(() => {
    return [
      {
        id: 'indicator',
        header: '',
        size: 28,
        minSize: 24,
        maxSize: 44,
        cell: ({ row }) => (
          <span className="chip-dot evt-dot" style={{ background: resolveObjectIndicatorColor(row.original.statusKey), margin: '0 auto', display: 'block' }} />
        ),
      },
      {
        accessorKey: 'number',
        header: "Об'єкт",
        size: 70,
        minSize: 58,
        cell: ({ getValue }) => <span className="mono bright">{String(getValue())}</span>,
      },
      {
        accessorKey: 'group',
        header: 'Група',
        size: 60,
        minSize: 48,
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'contract',
        header: 'Договір',
        size: 110,
        minSize: 88,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'name',
        header: 'Назва',
        enableResizing: false,
        meta: { fluid: true, minWidth: 260 } satisfies TableColumnMeta,
      },
      {
        accessorKey: 'address',
        header: 'Адреса',
        enableResizing: false,
        meta: { fluid: true, minWidth: 240 } satisfies TableColumnMeta,
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'statusLabel',
        header: 'Стан',
        size: 120,
        minSize: 98,
        cell: ({ row, getValue }) => <span className={`chip ${row.original.statusClass}`}>{String(getValue())}</span>,
      },
      {
        accessorKey: 'lastEventAt',
        header: 'Час взяття',
        size: 130,
        minSize: 118,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'lastTestAt',
        header: 'Час здачі',
        size: 130,
        minSize: 118,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'phone',
        header: 'Телефон',
        size: 110,
        minSize: 92,
        cell: ({ getValue }) => <span className="mono dim">{String(getValue())}</span>,
      },
      {
        accessorKey: 'note',
        header: 'Примітка',
        size: 160,
        minSize: 120,
        cell: ({ getValue }) => <span className="dim">{String(getValue())}</span>,
      },
    ]
  }, [])

  const table = useReactTable({
    data: filteredRows,
    columns,
    enableColumnResizing: true,
    columnResizeMode: 'onChange',
    getCoreRowModel: getCoreRowModel(),
  })

  const tableRows = table.getRowModel().rows
  const virtualRows = useVirtualRows(tableRows, { rowHeight: 30, initialCount: 220, step: 220 })
  const columnCount = table.getAllLeafColumns().length

  return (
    <div className="obj-layout">
      <div className="obj-toolbar">
        <div className="search-box">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--tx2)" strokeWidth="2">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            type="text"
            placeholder="Пошук за номером, назвою або адресою"
            value={searchValue}
            onChange={(event) => onSearchChange(event.target.value)}
          />
          <span onClick={() => onSearchChange('')}>✕</span>
        </div>
        <button className="btn btn-blue" style={{ height: 26 }} onClick={onRefresh}>
          Оновити
        </button>
        <div
          style={{
            marginLeft: 'auto',
            background: 'var(--bg)',
            border: '1px solid var(--bd2)',
            borderRadius: 2,
            padding: '2px 10px',
            fontFamily: "'Share Tech Mono', monospace",
            fontSize: 13,
            color: 'var(--ac2)',
          }}
        >
          Об'єктів: <span>{filteredRows.length}</span>
        </div>
      </div>

      <div className="obj-filter-tabs">
        {OBJECT_FILTER_TABS.map(([key, label]) => (
          <div key={key} className={statusFilter === key ? 'oft active' : 'oft'} onClick={() => onStatusFilterChange(key)}>
            {label}
          </div>
        ))}
      </div>

      <div className="obj-table-wrap" ref={virtualRows.containerRef} onScroll={virtualRows.onScroll}>
        <table className="obj-table">
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
                className={selectedObjectID === tableRow.original.id ? 'selected' : ''}
                onClick={() => onSelectObject(tableRow.original.id)}
              >
                {tableRow.getVisibleCells().map((cell) => (
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
