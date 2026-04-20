import { useEffect, useState } from 'react'
import type { FrontendAlarmProcessingOption } from '../../shared/api/types'

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
    setCauseCode((prev) => (prev !== '' ? prev : options[0]?.code ?? ''))
  }, [isOpen, options])

  return (
    <div className={isOpen ? 'modal-overlay open' : 'modal-overlay'}>
      <div className="modal" style={{ width: 560, height: 'auto', maxWidth: '96vw' }}>
        <div className="modal-tb">
          <div className="modal-tb-icon" style={{ background: 'var(--ac5)' }}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
            </svg>
          </div>
          <span className="modal-tb-title">Завершення обробки тривоги</span>
          <div className="modal-tb-close" onClick={onClose}>✕</div>
        </div>

        <div className="modal-content">
          <div className="modal-pane active settings-pane">
            <div className="igrid" style={{ gridTemplateColumns: '1fr' }}>
              <div className="isection">
                <div className="isect-title">Тривога</div>
                <div className="irow">
                  <label>Об'єкт</label>
                  <input value={objectName} readOnly />
                </div>
                <div className="irow">
                  <label>Тип</label>
                  <input value={alarmTypeText} readOnly />
                </div>
                <div className="irow">
                  <label>Причина</label>
                  <select value={causeCode} onChange={(event) => setCauseCode(event.target.value)} disabled={loading || busy || options.length === 0}>
                    {options.map((item) => (
                      <option key={item.code} value={item.code}>
                        {item.label || item.code}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="irow" style={{ alignItems: 'flex-start', minHeight: 96 }}>
                  <label>Коментар</label>
                  <textarea
                    className="note-area"
                    style={{ minHeight: 72, resize: 'vertical' }}
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
                    disabled={busy}
                  />
                </div>
              </div>
            </div>

            {loading && <div className="settings-loading">Завантаження причин відпрацювання...</div>}
            {error !== '' && <div className="settings-banner settings-banner-error">{error}</div>}
          </div>
        </div>

        <div className="modal-footer">
          <button className="btn btn-gray" style={{ width: 140, height: 28 }} onClick={onClose} disabled={busy}>
            СКАСУВАТИ
          </button>
          <div style={{ marginLeft: 'auto' }} />
          <button
            className="btn btn-green"
            style={{ width: 200, height: 28 }}
            onClick={() => onSubmit({ causeCode, note })}
            disabled={busy || loading || causeCode.trim() === ''}
          >
            {busy ? 'ЗБЕРЕЖЕННЯ...' : 'ЗАВЕРШИТИ ОБРОБКУ'}
          </button>
        </div>
      </div>
    </div>
  )
}
