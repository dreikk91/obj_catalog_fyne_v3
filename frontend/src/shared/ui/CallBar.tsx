import { useCallStore } from '../state/call-store'
import { useNow } from '../../hooks/useNow'

export function CallBar() {
  const { activeCall, hangup } = useCallStore()
  const now = useNow()

  if (activeCall == null) return null

  const elapsedSec = Math.floor((now - activeCall.startedAt) / 1000)
  const mm = String(Math.floor(elapsedSec / 60)).padStart(2, '0')
  const ss = String(elapsedSec % 60).padStart(2, '0')
  const elapsed = `${mm}:${ss}`

  const isFailed = activeCall.phase === 'failed'
  const isDialing = activeCall.phase === 'dialing'

  return (
    <div className={`callbar ${isFailed ? 'callbar--failed' : isDialing ? 'callbar--dialing' : 'callbar--active'}`}>
      <div className="callbar-icon">
        {isFailed ? <FailIcon /> : <PhoneRingIcon animate={isDialing} />}
      </div>
      <div className="callbar-info">
        <span className="callbar-name">{activeCall.contactName}</span>
        <span className="callbar-phone">{activeCall.phone}</span>
      </div>
      <span className="callbar-status">
        {isFailed ? 'Помилка' : isDialing ? 'Набір…' : elapsed}
      </span>
      <button
        type="button"
        className="callbar-hangup"
        title={isFailed ? 'Закрити' : 'Завершити дзвінок'}
        onClick={() => void hangup()}
      >
        {isFailed ? <DismissIcon /> : <HangupIcon />}
      </button>
    </div>
  )
}

function PhoneRingIcon({ animate }: { animate: boolean }) {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
      style={animate ? { animation: 'phone-ring 0.6s ease-in-out infinite' } : undefined}>
      <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013 7.82a2 2 0 012-2.18h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L9.91 13a16 16 0 006 6l.27-.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 20.89z" />
    </svg>
  )
}

function HangupIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
      <path d="M10.68 13.31a16 16 0 003.41 2.6l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07 19.42 19.42 0 01-3.33-2.67m-2.67-3.34a19.79 19.79 0 01-3.07-8.63A2 2 0 014.11 2h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L8.09 9.91" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </svg>
  )
}

function DismissIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  )
}

function FailIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="8" x2="12" y2="12" />
      <line x1="12" y1="16" x2="12.01" y2="16" />
    </svg>
  )
}
