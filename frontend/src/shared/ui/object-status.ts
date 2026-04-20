import type { FrontendObjectSummary } from '../api/types'

export type ObjectStatusView = {
  label: string
  tone: 'ok' | 'warn' | 'danger' | 'muted'
}

export function resolveObjectStatus(item: FrontendObjectSummary): ObjectStatusView {
  if (item.monitoringStatus === 'blocked' || item.connectionStatus === 'offline') {
    return { label: 'Проблема звʼязку', tone: 'warn' }
  }
  if (item.guardStatus === 'disarmed') {
    return { label: 'Без охорони', tone: 'muted' }
  }
  if (item.statusText.trim() !== '') {
    return { label: item.statusText, tone: 'ok' }
  }
  return { label: 'Під охороною', tone: 'ok' }
}

