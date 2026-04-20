import './status-pill.css'

type Tone = 'ok' | 'warn' | 'info'

type Props = {
  tone: Tone
  label: string
}

export function StatusPill({ tone, label }: Props) {
  const className = `status-pill status-pill--${tone}`
  return <span className={className}>{label}</span>
}
