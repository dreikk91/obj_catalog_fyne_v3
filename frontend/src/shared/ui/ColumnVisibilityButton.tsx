import { useState, useRef, useEffect } from 'react'

export type ColumnToggleItem = {
  id: string
  label: string
  isVisible: boolean
}

type ColumnVisibilityButtonProps = {
  columns: ColumnToggleItem[]
  onToggle: (columnId: string) => void
  onReset: () => void
}

/**
 * Renders a small grid icon button meant to be placed inside a <th> cell.
 * On click it opens a dropdown with column checkboxes.
 */
export function ColumnVisibilityButton({ columns, onToggle, onReset }: ColumnVisibilityButtonProps) {
  const [isOpen, setIsOpen] = useState(false)
  const wrapperRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!isOpen) return
    const handleClickOutside = (event: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen])

  return (
    <div className="col-vis-th" ref={wrapperRef}>
      <button
        type="button"
        className={isOpen ? 'col-vis-th-btn active' : 'col-vis-th-btn'}
        onClick={(e) => { e.stopPropagation(); setIsOpen((v) => !v) }}
        title="Вибір колонок"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
          <rect x="3" y="3" width="7" height="7" />
          <rect x="14" y="3" width="7" height="7" />
          <rect x="3" y="14" width="7" height="7" />
          <rect x="14" y="14" width="7" height="7" />
        </svg>
      </button>
      {isOpen && (
        <div className="col-vis-menu">
          <div className="col-vis-header">
            <span>Видимість колонок</span>
            <button className="col-vis-reset" onClick={onReset}>Скинути</button>
          </div>
          {columns.map((col) => (
            <label key={col.id} className="col-vis-item">
              <input
                type="checkbox"
                checked={col.isVisible}
                onChange={() => onToggle(col.id)}
              />
              <span>{col.label}</span>
            </label>
          ))}
        </div>
      )}
    </div>
  )
}
