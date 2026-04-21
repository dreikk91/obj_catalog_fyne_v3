import { useEffect, useState } from 'react'
import type { FrontendAlarmProcessingOption } from '../../shared/api/types'
import './AlarmProcessingModal.css'

type AlarmProcessingModalProps = {
  isOpen: boolean
  objectName: string
  alarmTypeText: string
  options: FrontendAlarmProcessingOption[]
  loading: boolean
  busy: boolean
  error: string
  onClose: () => void
  onSubmit: (payload: { causeCode: string; note: string }) => void
}

export function AlarmProcessingModal({
  isOpen,
  objectName,
  alarmTypeText,
  options,
  loading,
  busy,
  error,
  onClose,
  onSubmit,
}: AlarmProcessingModalProps) {
  const [causeCode, setCauseCode] = useState('')
  const [note, setNote] = useState('')

  useEffect(() => {
    if (!isOpen) {
      setNote('')
      return
    }
    if (options.length > 0 && causeCode === '') {
      setCauseCode(options[0].code)
    }
  }, [isOpen, options, causeCode])

  if (!isOpen) return null

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()} role="dialog" aria-labelledby="proc-modal-title">
      <div className="modal" style={{ width: 520, height: 'auto', maxHeight: '90vh' }}>
        <header className="alarm-proc-header">
          <div className="alarm-proc-title-row">
            <div className="alarm-proc-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
                <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              </svg>
            </div>
            <h2 id="proc-modal-title" className="alarm-proc-title">Завершення обробки</h2>
            <div className="modal-tb-close" style={{ marginLeft: 'auto' }} onClick={onClose} aria-label="Закрити">✕</div>
          </div>
          <p className="alarm-proc-subtitle">Вкажіть причину виникнення тривоги та додайте коментар</p>
        </header>

        <div className="alarm-proc-content">
          <section className="alarm-info-card" aria-label="Інформація про тривогу">
            <div className="info-field">
              <span className="info-label">Об'єкт</span>
              <span className="info-value">{objectName}</span>
            </div>
            <div className="info-field">
              <span className="info-label">Тип тривоги</span>
              <span className="info-value">{alarmTypeText}</span>
            </div>
          </section>

          <form className="proc-form" onSubmit={(e) => { e.preventDefault(); onSubmit({ causeCode, note }) }}>
            <div className="form-group">
              <label htmlFor="cause-select" className="form-label">Причина відпрацювання</label>
              <select
                id="cause-select"
                className="proc-select"
                value={causeCode}
                onChange={(e) => setCauseCode(e.target.value)}
                disabled={loading || busy || options.length === 0}
                required
              >
                {options.length === 0 && !loading && <option value="">Немає доступних варіантів</option>}
                {options.map((item) => (
                  <option key={item.code} value={item.code}>
                    {item.label || item.code}
                  </option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="note-area" className="form-label">Коментар оператора</label>
              <textarea
                id="note-area"
                className="proc-textarea"
                placeholder="Введіть подробиці обробки..."
                value={note}
                onChange={(e) => setNote(e.target.value)}
                disabled={busy}
              />
            </div>

            {loading && <div className="settings-loading" style={{ minHeight: 40 }}>Завантаження причин...</div>}
            {error !== '' && <div className="proc-error-msg">{error}</div>}
          </form>
        </div>

        <footer className="modal-actions">
          <button type="button" className="btn-cancel-modal" onClick={onClose} disabled={busy}>
            СКАСУВАТИ
          </button>
          <button
            type="submit"
            className="btn-finish"
            onClick={() => onSubmit({ causeCode, note })}
            disabled={busy || loading || causeCode === ''}
          >
            {busy ? (
              <>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="animate-spin" style={{ animation: 'spin 1s linear infinite' }}>
                  <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
                </svg>
                ЗБЕРЕЖЕННЯ...
              </>
            ) : (
              <>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                ЗАВЕРШИТИ ОБРОБКУ
              </>
            )}
          </button>
        </footer>
      </div>

      <style dangerouslySetInnerHTML={{ __html: `
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}} />
    </div>
  )
}
