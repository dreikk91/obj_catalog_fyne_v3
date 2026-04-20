import type { FrontendObjectDetails, FrontendObjectSummary } from '../../shared/api/types'
import { formatEventTime } from '../../shared/ui/time'
import { sourceLabel } from '../../shared/ui/source'
import './object-details-panel.css'

type Props = {
  objectSummary: FrontendObjectSummary | null
  details?: FrontendObjectDetails
  loading: boolean
}

export function ObjectDetailsPanel({ objectSummary, details, loading }: Props) {
  if (objectSummary == null) {
    return (
      <section className="panel panel--details panel--empty">
        <p>Оберіть об'єкт у лівій панелі</p>
      </section>
    )
  }

  return (
    <section className="panel panel--details">
      <header className="panel-header">
        <h2>Картка об'єкта</h2>
        <p>{loading ? 'Оновлення…' : objectSummary.statusText || '—'}</p>
      </header>

      <div className="details-grid">
        <article className="info-card info-card--primary">
          <h3>{objectSummary.name}</h3>
          <dl>
            <dt>Номер</dt>
            <dd>{objectSummary.displayNumber || objectSummary.id}</dd>
            <dt>Джерело</dt>
            <dd>{sourceLabel(objectSummary.source)}</dd>
            <dt>Адреса</dt>
            <dd>{objectSummary.address || '—'}</dd>
            <dt>Договір</dt>
            <dd>{objectSummary.contractNumber || '—'}</dd>
            <dt>Телефон</dt>
            <dd>{details?.phones || objectSummary.phone || '—'}</dd>
            <dt>Останній тест</dt>
            <dd>{formatEventTime(objectSummary.lastTestTime)}</dd>
          </dl>
        </article>

        <article className="info-card">
          <h3>Зони</h3>
          <div className="mini-table">
            <div className="mini-row mini-row--head">
              <span>№</span>
              <span>Назва</span>
              <span>Стан</span>
            </div>
            {(details?.zones ?? []).slice(0, 12).map((zone) => (
              <div className="mini-row" key={`${zone.number}-${zone.name}`}>
                <span>{zone.number}</span>
                <span>{zone.name}</span>
                <span>{zone.status || '—'}</span>
              </div>
            ))}
            {(details?.zones?.length ?? 0) === 0 ? <p className="empty-row">Дані відсутні</p> : null}
          </div>
        </article>

        <article className="info-card">
          <h3>Відповідальні</h3>
          <div className="mini-table">
            <div className="mini-row mini-row--head">
              <span>Пріор.</span>
              <span>ПІБ</span>
              <span>Телефон</span>
            </div>
            {[...(details?.contacts ?? [])]
              .sort((a, b) => a.priority - b.priority)
              .slice(0, 12)
              .map((contact) => (
                <div className="mini-row" key={`${contact.name}-${contact.phone}`}>
                  <span>{contact.priority || '—'}</span>
                  <span>{contact.name}</span>
                  <span>{contact.phone || '—'}</span>
                </div>
              ))}
            {(details?.contacts?.length ?? 0) === 0 ? <p className="empty-row">Дані відсутні</p> : null}
          </div>
        </article>

        <article className="info-card info-card--wide">
          <h3>Події об'єкта</h3>
          <div className="mini-table">
            <div className="mini-row mini-row--head">
              <span>Час</span>
              <span>Лінія</span>
              <span>Тип</span>
              <span>Опис</span>
            </div>
            {(details?.events ?? []).slice(0, 20).map((event) => (
              <div className="mini-row" key={`${event.id}-${event.time}`}>
                <span>{formatEventTime(event.time)}</span>
                <span>{sourceLabel(event.source)}</span>
                <span>{event.typeText || 'Подія'}</span>
                <span>{event.details || '—'}</span>
              </div>
            ))}
            {(details?.events?.length ?? 0) === 0 ? <p className="empty-row">Дані відсутні</p> : null}
          </div>
        </article>
      </div>
    </section>
  )
}
