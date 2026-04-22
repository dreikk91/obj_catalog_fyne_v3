import { useState } from 'react'

type StandbyModalProps = {
  isOpen: boolean
  busy: boolean
  error: string
  onClose: () => void
  onConfirm: (durationMinutes: number, reason: string) => void
}

export function StandbyModal({ isOpen, busy, error, onClose, onConfirm }: StandbyModalProps) {
  const [duration, setDuration] = useState('')
  const [reason, setReason] = useState('')

  if (!isOpen) return null

  const durationNum = duration === '' ? 0 : Math.min(1440, Math.max(0, parseInt(duration, 10) || 0))
  const durationLabel = durationNum === 0 ? 'Безстроково' : durationNum === 1440 ? '24 год (макс)' : `${durationNum} хв`

  const handleConfirm = () => {
    onConfirm(durationNum, reason.trim())
  }

  return (
    <div className="modal-overlay open" style={{ zIndex: 1100 }}>
      <div className="modal" style={{ width: 400 }}>
        <div className="modal-tb">
          <div className="modal-tb-icon" style={{ background: 'var(--ac2)' }}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
          </div>
          <span className="modal-tb-title">Вивести об'єкт в стенди</span>
          <div className="modal-tb-close" onClick={onClose}>✕</div>
        </div>

        <div style={{ padding: '16px 16px 8px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <div style={{ fontSize: 11, color: 'var(--tx2)', marginBottom: 4 }}>
              Тривалість, хв (0 = безстроково, макс 1440)
            </div>
            <input
              type="number"
              min={0}
              max={1440}
              placeholder="0"
              value={duration}
              onChange={(e) => setDuration(e.target.value)}
              style={{ width: '100%', height: 32, boxSizing: 'border-box', padding: '0 8px', borderRadius: 4, border: '1px solid var(--br1)', background: 'var(--bg2)', color: 'var(--tx1)', fontSize: 13 }}
            />
            <div style={{ fontSize: 11, color: 'var(--ac3)', marginTop: 3 }}>{durationLabel}</div>
          </div>
          <div>
            <div style={{ fontSize: 11, color: 'var(--tx2)', marginBottom: 4 }}>Причина</div>
            <input
              type="text"
              placeholder="Стенди"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              style={{ width: '100%', height: 32, boxSizing: 'border-box', padding: '0 8px', borderRadius: 4, border: '1px solid var(--br1)', background: 'var(--bg2)', color: 'var(--tx1)', fontSize: 13 }}
            />
          </div>
        </div>

        {error !== '' && (
          <div className="proc-error" style={{ margin: '0 12px 8px' }}>{error}</div>
        )}

        <div className="modal-footer">
          <div style={{ marginLeft: 'auto' }} />
          <button className="btn btn-gray" style={{ height: 28 }} onClick={onClose} disabled={busy}>
            Скасувати
          </button>
          <button
            className="btn btn-blue"
            style={{ height: 28 }}
            onClick={handleConfirm}
            disabled={busy}
          >
            {busy ? 'Надсилання...' : 'До стендів'}
          </button>
        </div>
      </div>
    </div>
  )
}
