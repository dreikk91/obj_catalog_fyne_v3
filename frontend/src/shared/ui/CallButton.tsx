import { useCallStore } from '../state/call-store'

type CallButtonProps = {
  phone: string
  contactName?: string
}

export function CallButton({ phone, contactName = phone }: CallButtonProps) {
  const { activeCall, dialerAvailable, dial, hangup } = useCallStore()

  const isBusy = activeCall != null
  const isThisCall = activeCall?.phone === phone

  if (!dialerAvailable) return null

  return (
    <button
      type="button"
      className={`call-btn ${isThisCall ? 'call-btn--active' : ''}`}
      disabled={isBusy && !isThisCall}
      title={isThisCall ? 'Завершити дзвінок' : `Зателефонувати: ${phone}`}
      onClick={() => {
        if (isThisCall) void hangup()
        else if (!isBusy) dial(phone, contactName)
      }}
    >
      <PhoneIcon />
    </button>
  )
}

function PhoneIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013 7.82a2 2 0 012-2.18h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L9.91 13a16 16 0 006 6l.27-.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 20.89z" />
    </svg>
  )
}
