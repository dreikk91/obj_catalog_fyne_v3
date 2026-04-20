import type { FrontendSource } from '../api/types'

export function sourceLabel(source: FrontendSource): string {
  switch (source) {
    case 'casl':
      return 'CASL'
    case 'phoenix':
      return 'Фенікс'
    case 'bridge':
      return 'Міст'
    default:
      return 'Невідомо'
  }
}

