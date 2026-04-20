import { useMemo, useState } from 'react'
import type { FrontendObjectSummary } from '../../shared/api/types'
import { useOperatorUIStore } from '../../shared/state/ui-store'
import { resolveObjectStatus } from '../../shared/ui/object-status'
import { sourceLabel } from '../../shared/ui/source'
import './object-list-panel.css'

type Props = {
  objects: FrontendObjectSummary[]
  selectedObjectID: number | null
  loading: boolean
}

export function ObjectListPanel({ objects, selectedObjectID, loading }: Props) {
  const setSelectedObjectID = useOperatorUIStore((s) => s.setSelectedObjectID)
  const [query, setQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'guarded' | 'disarmed' | 'problem'>('all')

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return objects.filter((item) => {
      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'guarded' && item.guardStatus === 'guarded') ||
        (statusFilter === 'disarmed' && item.guardStatus === 'disarmed') ||
        (statusFilter === 'problem' &&
          (item.connectionStatus === 'offline' || item.monitoringStatus === 'blocked'))

      if (!matchesStatus) {
        return false
      }

      if (q === '') {
        return true
      }

      return (
        item.displayNumber.toLowerCase().includes(q) ||
        item.name.toLowerCase().includes(q) ||
        item.address.toLowerCase().includes(q) ||
        sourceLabel(item.source).toLowerCase().includes(q)
      )
    })
  }, [objects, query, statusFilter])

  return (
    <section className="panel panel--objects">
      <header className="panel-header">
        <h2>Об'єкти</h2>
        <p>{loading ? 'Оновлення…' : `${filtered.length} записів`}</p>
      </header>

      <input
        className="panel-input"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Пошук: номер, назва, адреса, джерело"
      />

      <div className="object-filters" role="tablist" aria-label="Фільтр стану">
        <button
          type="button"
          className={statusFilter === 'all' ? 'status-filter status-filter--active' : 'status-filter'}
          onClick={() => setStatusFilter('all')}
        >
          Усі
        </button>
        <button
          type="button"
          className={statusFilter === 'guarded' ? 'status-filter status-filter--active' : 'status-filter'}
          onClick={() => setStatusFilter('guarded')}
        >
          Під охороною
        </button>
        <button
          type="button"
          className={statusFilter === 'disarmed' ? 'status-filter status-filter--active' : 'status-filter'}
          onClick={() => setStatusFilter('disarmed')}
        >
          Без охорони
        </button>
        <button
          type="button"
          className={statusFilter === 'problem' ? 'status-filter status-filter--active' : 'status-filter'}
          onClick={() => setStatusFilter('problem')}
        >
          Проблемні
        </button>
      </div>

      <div className="object-list" role="listbox" aria-label="Об'єкти">
        <div className="object-list-head">
          <span>№ / Джерело</span>
          <span>Назва</span>
          <span>Стан</span>
        </div>
        {filtered.map((item) => {
          const isActive = selectedObjectID === item.id
          const status = resolveObjectStatus(item)
          return (
            <button
              key={item.id}
              type="button"
              className={isActive ? 'object-row object-row--active' : 'object-row'}
              onClick={() => setSelectedObjectID(item.id)}
            >
              <span className="object-row-key">
                <strong>{item.displayNumber || item.id}</strong>
                <small>{sourceLabel(item.source)}</small>
              </span>
              <span className="object-row-main">
                <span className="object-row-name">{item.name}</span>
                <span className="object-row-address">{item.address || '—'}</span>
              </span>
              <span className={`object-status object-status--${status.tone}`}>{status.label}</span>
            </button>
          )
        })}
      </div>
    </section>
  )
}
