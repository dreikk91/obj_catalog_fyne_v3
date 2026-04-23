import type { FrontendEventItem, FrontendObjectDetails } from '../../shared/api/types'

export type GuardPresentation = {
  state: 'guarded' | 'disarmed' | 'unknown'
  text: string
}

export function resolveMainNotes(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }

  switch (details.summary.source) {
    case 'casl':
      return firstNonEmpty(details.description, details.notes)
    case 'phoenix':
      return firstNonEmpty(details.notes, details.description)
    case 'bridge':
      return firstNonEmpty(details.location)
    default:
      return firstNonEmpty(details.notes, details.description, details.location)
  }
}

export function resolveExtraNotes(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }
  if (details.summary.source === 'bridge') {
    return firstNonEmpty(details.notes)
  }
  return ''
}

export function resolveSecondaryAddressLine(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }

  switch (details.summary.source) {
    case 'casl':
      return firstNonEmpty(details.description, details.notes)
    case 'phoenix':
    case 'bridge':
      return firstNonEmpty(details.phones, details.summary.phone)
    default:
      return firstNonEmpty(details.phones, details.description, details.notes)
  }
}

export function resolveCity(details: FrontendObjectDetails | null | undefined): string {
  const address = details?.summary.address.trim() ?? ''
  if (address === '') {
    return ''
  }

  const raw = address.split(',')[0]?.trim() ?? ''
  if (raw === '') {
    return ''
  }

  return raw
    .replace(/\s+м\.$/i, '')
    .replace(/\s+м$/i, '')
    .trim()
}

export function resolvePanelSummary(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }

  const panelMark = details.summary.panelMark.trim()
  const deviceType = details.summary.deviceType.trim()
  if (panelMark !== '' && deviceType !== '' && panelMark !== deviceType) {
    return `${panelMark} + ${deviceType}`
  }

  return firstNonEmpty(deviceType, panelMark, details.summary.contractNumber)
}

export function resolvePanelLine(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }

  return firstNonEmpty(details.summary.panelMark, details.summary.deviceType, details.summary.statusText)
}

export function resolveContactNames(details: FrontendObjectDetails | null | undefined, limit = 3): string[] {
  if (details == null || limit <= 0) {
    return []
  }

  return [...details.contacts]
    .sort((left, right) => {
      if (left.priority !== right.priority) {
        return left.priority - right.priority
      }
      return left.name.localeCompare(right.name, 'uk')
    })
    .map((contact) => contact.name.trim())
    .filter((name) => name !== '')
    .slice(0, limit)
}

export function resolveAuxiliaryLine(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }

  switch (details.summary.source) {
    case 'bridge':
      return firstNonEmpty(details.location, details.notes, details.description)
    case 'phoenix':
      return firstNonEmpty(details.location, details.description, details.notes)
    case 'casl':
      return firstNonEmpty(details.description, details.notes)
    default:
      return firstNonEmpty(details.location, details.description, details.notes)
  }
}

export function resolveReactionLine(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }

  const preferredGroup = resolvePreferredResponseGroup(details)
  const trailing =
    details.summary.source === 'casl'
      ? firstNonEmpty(details.description, details.notes)
      : firstNonEmpty(details.phones, details.summary.phone)

  if (preferredGroup !== '' && trailing !== '') {
    return `Реагує ${preferredGroup} ${trailing}`
  }
  if (preferredGroup !== '') {
    return `Реагує ${preferredGroup}`
  }
  if (trailing !== '') {
    return trailing
  }
  return ''
}

export function resolveGuardPresentation(details: FrontendObjectDetails | null | undefined): GuardPresentation {
  const latestEvent = findLatestGuardEvent(details?.events ?? [])
  if (latestEvent == null) {
    return {
      state: 'unknown',
      text: 'НЕВІДОМО',
    }
  }

  return {
    state: latestEvent.typeCode === 'arm' ? 'guarded' : 'disarmed',
    text: formatGuardEventTime(latestEvent.time),
  }
}

export function resolvePreferredResponseGroup(details: FrontendObjectDetails | null | undefined): string {
  if (details == null) {
    return ''
  }
  const name = details.preferredResponseGroupName.trim()
  if (name !== '') {
    return name
  }
  const id = details.preferredResponseGroupID.trim()
  if (id !== '') {
    return `ГМР ${id}`
  }
  return ''
}

function findLatestGuardEvent(events: FrontendEventItem[]): FrontendEventItem | null {
  let latest: FrontendEventItem | null = null
  let latestTimestamp = Number.NEGATIVE_INFINITY

  for (const event of events) {
    if (event.typeCode !== 'arm' && event.typeCode !== 'disarm') {
      continue
    }

    const parsed = Date.parse(event.time)
    const timestamp = Number.isNaN(parsed) ? Number.NEGATIVE_INFINITY : parsed
    if (latest == null || timestamp > latestTimestamp) {
      latest = event
      latestTimestamp = timestamp
    }
  }

  return latest
}

function formatGuardEventTime(value: string): string {
  if (value.trim() === '') {
    return 'НЕВІДОМО'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const year = String(date.getFullYear())
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${day}.${month}.${year} ${hours}:${minutes}:${seconds}`
}

function firstNonEmpty(...values: string[]): string {
  for (const value of values) {
    const trimmed = value.trim()
    if (trimmed !== '') {
      return trimmed
    }
  }
  return ''
}
