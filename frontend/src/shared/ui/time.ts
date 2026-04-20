const formatter = new Intl.DateTimeFormat('uk-UA', {
  day: '2-digit',
  month: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
})

export function formatEventTime(value: string): string {
  if (!value) {
    return '—'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return formatter.format(date)
}
