import { useEffect, useMemo, useState } from 'react'
import type { FrontendResponseGroup } from '../../shared/api/types'

type DispatchGroupModalProps = {
  isOpen: boolean
  groups: FrontendResponseGroup[]
  preferredGroupID?: string
  preferredGroupName?: string
  objectGroupHint?: string
  busy: boolean
  error: string
  onClose: () => void
  onConfirm: (groupID: string) => void
}

type SuggestedGroup = {
  id: string
  reason: string
}

function normalizeText(value: string): string {
  return value.trim().toLowerCase()
}

function normalizeDigits(value: string): string {
  return value.replace(/\D+/g, '')
}

function extractObjectGroupHint(value: string): string {
  const lines = value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== '')

  const explicit = lines.find((line) => /(?:гмр|мгр|реаг|груп)/i.test(line))
  if (explicit) return explicit

  const compact = value.trim()
  if (compact.length <= 140) return compact
  return compact.slice(0, 140).trim() + '...'
}

function resolveSuggestedGroup(groups: FrontendResponseGroup[], preferredGroupID: string, hint: string): SuggestedGroup | null {
  const normalizedPreferredID = normalizeText(preferredGroupID)
  if (normalizedPreferredID !== '') {
    const explicitGroup = groups.find((group) => normalizeText(group.id) === normalizedPreferredID)
    if (explicitGroup != null) {
      return {
        id: explicitGroup.id,
        reason: 'група встановлена в картці обʼєкта',
      }
    }
  }

  const normalizedHint = normalizeText(hint)
  if (normalizedHint === '') return null
  const digitsHint = normalizeDigits(hint)

  let best: { score: number; suggestion: SuggestedGroup } | null = null
  for (const group of groups) {
    const name = normalizeText(group.name)
    const callsign = normalizeText(group.callsign)
    const phoneDigits = normalizeDigits(group.phone)
    const idText = normalizeText(group.id)

    let score = 0
    let reason = ''

    if (name !== '' && normalizedHint.includes(name)) {
      score = 100
      reason = 'назва збігається з карткою обʼєкта'
    } else if (callsign !== '' && normalizedHint.includes(callsign)) {
      score = 90
      reason = 'позивний збігається з карткою обʼєкта'
    } else if (digitsHint !== '' && phoneDigits !== '' && digitsHint.includes(phoneDigits)) {
      score = 80
      reason = 'телефон збігається з карткою обʼєкта'
    } else if (idText !== '' && normalizedHint.includes(idText)) {
      score = 70
      reason = 'ID групи згадується в картці обʼєкта'
    }

    if (score === 0) continue
    if (best == null || score > best.score) {
      best = { score, suggestion: { id: group.id, reason } }
    }
  }

  return best?.suggestion ?? null
}

export function DispatchGroupModal({
  isOpen,
  groups,
  preferredGroupID = '',
  preferredGroupName = '',
  objectGroupHint = '',
  busy,
  error,
  onClose,
  onConfirm,
}: DispatchGroupModalProps) {
  const [selectedID, setSelectedID] = useState('')
  const visibleHint = useMemo(() => extractObjectGroupHint(objectGroupHint), [objectGroupHint])
  const suggestedGroup = useMemo(
    () => resolveSuggestedGroup(groups, preferredGroupID, objectGroupHint),
    [groups, preferredGroupID, objectGroupHint],
  )
  const preferredGroup = useMemo(
    () => groups.find((item) => normalizeText(item.id) === normalizeText(preferredGroupID)) ?? null,
    [groups, preferredGroupID],
  )

  useEffect(() => {
    if (!isOpen) {
      setSelectedID('')
      return
    }
    if (suggestedGroup != null) {
      setSelectedID(suggestedGroup.id)
      return
    }
    setSelectedID('')
  }, [isOpen, suggestedGroup])

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
          {preferredGroupID.trim() !== '' && (
            <div
              style={{
                margin: '0 12px 10px',
                padding: '8px 10px',
                border: '1px solid var(--bd)',
                borderRadius: 6,
                background: 'rgba(255,255,255,0.03)',
                fontSize: 12,
                lineHeight: 1.35,
              }}
            >
              <div style={{ color: 'var(--tx2)', marginBottom: 4 }}>У картці обʼєкта встановлена ГМР:</div>
              <div className="bright">
                {preferredGroup?.name || preferredGroupName || `ID ${preferredGroupID}`}
              </div>
              {preferredGroup == null && preferredGroupName.trim() === '' && (
                <div style={{ color: 'var(--tx2)', marginTop: 4 }}>
                  Цієї групи немає у поточному списку доступних ГМР.
                </div>
              )}
            </div>
          )}
          {visibleHint !== '' && (
            <div
              style={{
                margin: '0 12px 10px',
                padding: '8px 10px',
                border: '1px solid var(--bd)',
                borderRadius: 6,
                background: 'rgba(255,255,255,0.03)',
                fontSize: 12,
                lineHeight: 1.35,
              }}
            >
              <div style={{ color: 'var(--tx2)', marginBottom: 4 }}>У картці обʼєкта вказано:</div>
              <div className="bright">{visibleHint}</div>
              {suggestedGroup != null && (
                <div style={{ color: 'var(--tx2)', marginTop: 4 }}>
                  Підібрано групу зі списку: {groups.find((item) => item.id === suggestedGroup.id)?.name || suggestedGroup.id}
                </div>
              )}
            </div>
          )}
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
                    style={{
                      cursor: 'pointer',
                      outline: suggestedGroup?.id === g.id ? '1px solid rgba(255,255,255,0.14)' : undefined,
                    }}
                    onClick={() => setSelectedID(g.id)}
                  >
                    <td style={{ textAlign: 'center' }}>
                      <input type="radio" checked={selectedID === g.id} onChange={() => setSelectedID(g.id)} />
                    </td>
                    <td className="bright">
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <span>{g.name || '—'}</span>
                        {suggestedGroup?.id === g.id && (
                          <span
                            style={{
                              fontSize: 10,
                              color: 'var(--tx2)',
                              border: '1px solid var(--bd)',
                              borderRadius: 999,
                              padding: '1px 6px',
                            }}
                          >
                            Підказка
                          </span>
                        )}
                      </div>
                    </td>
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
