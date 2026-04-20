import { useState } from 'react'
import type { FrontendResponseGroup } from '../../shared/api/types'

type DispatchGroupModalProps = {
  isOpen: boolean
  groups: FrontendResponseGroup[]
  busy: boolean
  error: string
  onClose: () => void
  onConfirm: (groupID: string) => void
}

export function DispatchGroupModal({ isOpen, groups, busy, error, onClose, onConfirm }: DispatchGroupModalProps) {
  const [selectedID, setSelectedID] = useState('')

  if (!isOpen) return null

  return (
    <div className="modal-overlay open" style={{ zIndex: 1100 }}>
      <div className="modal" style={{ width: 480, maxHeight: 480 }}>
        <div className="modal-tb">
          <div className="modal-tb-icon" style={{ background: 'var(--ac3)' }}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
              <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" />
            </svg>
          </div>
          <span className="modal-tb-title">Вислати групу реагування</span>
          <div className="modal-tb-close" onClick={onClose}>✕</div>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: '8px 0' }}>
          {groups.length === 0 ? (
            <div style={{ padding: '24px', textAlign: 'center', color: 'var(--tx2)', fontSize: 12 }}>
              Групи реагування відсутні
            </div>
          ) : (
            <table className="mtable" style={{ width: '100%' }}>
              <thead>
                <tr>
                  <th style={{ width: 28 }} />
                  <th>Назва</th>
                  <th>Позивний</th>
                  <th>Телефон</th>
                </tr>
              </thead>
              <tbody>
                {groups.map((g) => (
                  <tr
                    key={g.id}
                    className={selectedID === g.id ? 'selected' : ''}
                    style={{ cursor: 'pointer' }}
                    onClick={() => setSelectedID(g.id)}
                  >
                    <td style={{ textAlign: 'center' }}>
                      <input type="radio" checked={selectedID === g.id} onChange={() => setSelectedID(g.id)} />
                    </td>
                    <td className="bright">{g.name || '—'}</td>
                    <td className="dim">{g.callsign || '—'}</td>
                    <td className="mono dim">{g.phone || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
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
            className="btn btn-green"
            style={{ height: 28 }}
            onClick={() => selectedID !== '' && onConfirm(selectedID)}
            disabled={busy || selectedID === ''}
          >
            {busy ? 'Надсилання...' : 'Вислати'}
          </button>
        </div>
      </div>
    </div>
  )
}
