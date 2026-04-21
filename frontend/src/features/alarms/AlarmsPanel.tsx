import type { FrontendAlarmItem } from '../../shared/api/types'
import { formatEventTime } from '../../shared/ui/time'
import { sourceLabel } from '../../shared/ui/source'
import './alarms-panel.css'

type Props = {
  alarms: FrontendAlarmItem[]
  loading: boolean
}

export function AlarmsPanel({ alarms, loading }: Props) {
  return (
    <section className="panel panel--alarms">
      <header className="panel-header">
        <h2>Необроблені</h2>
        <p>{loading ? 'Оновлення…' : `${alarms.length} записів`}</p>
      </header>

      <div className="alarm-table-wrap">
        <table className="alarm-table">
          <thead>
            <tr>
              <th style={{ width: 48 }}>Лінія</th>
              <th style={{ width: 140 }}>Об'єкт</th>
              <th style={{ width: 150 }}>Тривога</th>
              <th style={{ width: 80 }}>Час</th>
              <th>Опис</th>
            </tr>
          </thead>
          <tbody>
            {alarms.slice(0, 400).map((alarm) => (
              <tr key={`${alarm.id}-${alarm.time}`}>
                <td>{sourceLabel(alarm.source)}</td>
                <td>
                  <strong>{alarm.objectNumber || alarm.objectID}</strong>
                  <span>{alarm.objectName || '—'}</span>
                </td>
                <td>{alarm.typeText || 'Тривога'}</td>
                <td>{formatEventTime(alarm.time)}</td>
                <td>{alarm.details || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
