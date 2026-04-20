import type { JournalRow, ObjectRow } from '../operator/types'

type ObjectInfoBarProps = {
  selectedSignalRow: JournalRow | null
  selectedObjectRow: ObjectRow | null
}

export function ObjectInfoBar({ selectedSignalRow, selectedObjectRow }: ObjectInfoBarProps) {
  return (
    <div className="obj-info-bar">
      <div style={{ display: 'flex', gap: 10 }}>
        <div style={{ flex: 1 }}>
          <div className="oib-grid">
            <InfoInput icon="card" value={selectedSignalRow?.objectNumber ?? '—'} />
            <InfoInput icon="lock" value={selectedSignalRow?.code ?? '—'} />
            <InfoInput icon="map" value={selectedObjectRow?.address ?? selectedSignalRow?.details ?? '—'} />
            <InfoInput icon="alarm" value={selectedSignalRow?.typeText ?? '—'} />

            <InfoInput icon="user" value={selectedSignalRow?.objectName ?? '—'} />
            <InfoInput icon="group" value={selectedObjectRow?.group ?? '—'} />
            <InfoInput icon="group" value={selectedObjectRow?.phone ?? '—'} />
            <InfoInput icon="mobile" value={selectedObjectRow?.contract ?? '—'} />
          </div>
          <div className="oib-bottom">
            <InfoInput icon="doc" value={selectedObjectRow?.name ?? '—'} />
            <InfoInput icon="key" value={selectedObjectRow?.lastTestAt ?? '—'} />
            <InfoInput icon="panel" value={selectedObjectRow?.note ?? '—'} />
            <InfoStatus value={selectedObjectRow?.statusLabel ?? 'НЕВІДОМО'} />
          </div>
        </div>
        <div className="oib-right" style={{ width: 220, justifyContent: 'center' }}>
          <div className="oib-right-row">
            <input className="oib-right-inp" value="Тип реагув." readOnly />
            <input className="oib-right-inp" value={selectedSignalRow?.group ?? '—'} readOnly />
          </div>
          <div className="oib-right-row">
            <input className="oib-right-inp" value={selectedSignalRow?.zone ?? '—'} readOnly />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
            <span className="workreg-lbl">Режим роботи</span>
            <input className="oib-right-inp" style={{ width: 50 }} value="24" readOnly />
            <input className="oib-right-inp" style={{ width: 50 }} value="7" readOnly />
          </div>
          <button className="redirect-btn" style={{ marginTop: 6 }}>
            Переадресація ContactID
          </button>
        </div>
      </div>
    </div>
  )
}

function InfoInput({ icon, value }: { icon: 'card' | 'lock' | 'map' | 'alarm' | 'user' | 'group' | 'mobile' | 'doc' | 'key' | 'panel'; value: string }) {
  return (
    <div className="oib-row">
      <div className="oib-icon">
        <Icon kind={icon} />
      </div>
      <input className="oib-inp" value={value} readOnly />
    </div>
  )
}

function InfoStatus({ value }: { value: string }) {
  return (
    <div className="oib-row">
      <div className="oib-icon">
        <Icon kind="phone" />
      </div>
      <div className="status-unknown">{value}</div>
    </div>
  )
}

function Icon({ kind }: { kind: string }) {
  const common = { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: '2' } as const

  switch (kind) {
    case 'card':
      return (
        <svg {...common}>
          <rect x="3" y="4" width="18" height="16" rx="2" />
          <line x1="7" y1="9" x2="17" y2="9" />
          <line x1="7" y1="13" x2="13" y2="13" />
        </svg>
      )
    case 'lock':
      return (
        <svg {...common}>
          <rect x="3" y="11" width="18" height="11" rx="2" />
          <path d="M7 11V7a5 5 0 0110 0v4" />
        </svg>
      )
    case 'map':
      return (
        <svg {...common}>
          <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z" />
          <circle cx="12" cy="10" r="3" />
        </svg>
      )
    case 'alarm':
      return (
        <svg {...common}>
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
        </svg>
      )
    case 'user':
      return (
        <svg {...common}>
          <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2" />
          <circle cx="12" cy="7" r="4" />
        </svg>
      )
    case 'group':
      return (
        <svg {...common}>
          <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
          <circle cx="9" cy="7" r="4" />
          <path d="M23 21v-2a4 4 0 00-3-3.87" />
        </svg>
      )
    case 'mobile':
      return (
        <svg {...common}>
          <rect x="5" y="2" width="14" height="20" rx="2" />
          <line x1="12" y1="18" x2="12.01" y2="18" />
        </svg>
      )
    case 'doc':
      return (
        <svg {...common}>
          <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
          <polyline points="14 2 14 8 20 8" />
        </svg>
      )
    case 'key':
      return (
        <svg {...common}>
          <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 11-7.778 7.778 5.5 5.5 0 017.777-7.777zm0 0L15.5 7.5" />
        </svg>
      )
    case 'panel':
      return (
        <svg {...common}>
          <rect x="2" y="7" width="20" height="14" rx="2" />
          <path d="M16 3h-8v4h8V3z" />
        </svg>
      )
    default:
      return (
        <svg {...common}>
          <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013 7.82a2 2 0 012-2.18h3a2 2 0 012 1.72 12.84 12.84 0 00.7 2.81 2 2 0 01-.45 2.11L9.91 13a16 16 0 006 6l.27-.27a2 2 0 012.11-.45 12.84 12.84 0 002.81.7A2 2 0 0122 20.89z" />
        </svg>
      )
  }
}
