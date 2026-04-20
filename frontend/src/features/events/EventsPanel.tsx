import type { FrontendEventItem } from '../../shared/api/types'
import { formatEventTime } from '../../shared/ui/time'
import { sourceLabel } from '../../shared/ui/source'
import './events-panel.css'

type Props = {
  events: FrontendEventItem[]
  loading: boolean
}

export function EventsPanel({ events, loading }: Props) {
  return (
    <section className="panel panel--events">
      <header className="panel-header">
        <h2>Архів</h2>
        <p>{loading ? 'Оновлення…' : `${events.length} записів`}</p>
      </header>

      <div className="journal-table-wrap">
        <table className="journal-table">
          <thead>
            <tr>
              <th>Лінія</th>
              <th>Об'єкт</th>
              <th>Подія</th>
              <th>Час</th>
              <th>Опис</th>
            </tr>
          </thead>
          <tbody>
            {events.slice(0, 400).map((event) => (
              <tr key={`${event.id}-${event.time}`}>
                <td>{sourceLabel(event.source)}</td>
                <td>
                  <strong>{event.objectNumber || event.objectID}</strong>
                  <span>{event.objectName || '—'}</span>
                </td>
                <td>{event.typeText || 'Подія'}</td>
                <td>{formatEventTime(event.time)}</td>
                <td>{event.details || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
