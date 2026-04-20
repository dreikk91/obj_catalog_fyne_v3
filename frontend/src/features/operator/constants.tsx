import type { ReactNode } from 'react'
import type { ModalTab, StatusFilter } from '../../shared/state/ui-store'

export const RECENT_JOURNAL_WINDOW_MS = 60 * 60 * 1000
export const JOURNAL_WS_URL = import.meta.env.VITE_JOURNAL_WS_URL ?? 'ws://127.0.0.1:17891/ws/frontend/v1/journal'
export const VIRTUAL_OVERSCAN_ROWS = 8
export const VIRTUAL_INITIAL_CHUNK = 180
export const VIRTUAL_STEP_CHUNK = 180
export const JOURNAL_BOOTSTRAP_EVENTS_LIMIT = 100
export const OBJECT_EVENTS_PAGE_SIZE = 100
export const OBJECT_EVENTS_REFRESH_MS = 5_000

export const BASE_GROUP_NAMES = ['Альфа', 'Бета', 'Гамма']
export const BASE_KEY_OWNERS = ['Іваненко О.П.', 'Петренко В.М.', 'Сидоренко Л.І.', 'Коваль Т.О.']
export const CARD_DEVICE_ROWS = [
  { type: 'ГПО', model: 'Орлан-GPRS v3', serial: 'ORL-2023-00451', version: '3.14.2', channel: 'GPRS', status: 'НОРМА' },
  { type: 'РПД', model: 'Кварц-10A', serial: 'KVR-2021-07813', version: '2.8.1', channel: 'ТФ', status: 'НОРМА' },
]

export const OBJECT_FILTER_TABS: ReadonlyArray<readonly [StatusFilter, string]> = [
  ['all', 'Всі'],
  ['guarded', 'ПІД ОХОРОНОЮ'],
  ['unguarded', 'БЕЗ ОХОРОНИ'],
  ['late', 'Вчасно не під охороною'],
  ['call', 'На прозвон'],
  ['alarm', 'Тривожні'],
  ['banned', 'Діє заборона'],
]

export const MODAL_TABS: { id: ModalTab; label: string; icon: ReactNode }[] = [
  {
    id: 'kartochka',
    label: 'Картка',
    icon: (
      <>
        <rect x="3" y="4" width="18" height="16" rx="2" />
        <line x1="7" y1="9" x2="17" y2="9" />
        <line x1="7" y1="13" x2="13" y2="13" />
      </>
    ),
  },
  {
    id: 'devices',
    label: 'Прилади',
    icon: (
      <>
        <rect x="2" y="7" width="20" height="14" rx="2" />
        <path d="M16 3h-8v4h8V3z" />
      </>
    ),
  },
  {
    id: 'zones',
    label: 'Зони',
    icon: (
      <>
        <polygon points="12 2 2 7 12 12 22 7 12 2" />
        <polyline points="2 17 12 22 22 17" />
        <polyline points="2 12 12 17 22 12" />
      </>
    ),
  },
  {
    id: 'response',
    label: 'Реагування',
    icon: (
      <>
        <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
        <circle cx="9" cy="7" r="4" />
        <path d="M23 21v-2a4 4 0 00-3-3.87" />
      </>
    ),
  },
  {
    id: 'keys',
    label: 'Ключі',
    icon: <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 11-7.778 7.778 5.5 5.5 0 017.777-7.777zm0 0L15.5 7.5" />,
  },
  {
    id: 'resp',
    label: 'Відповідальні',
    icon: (
      <>
        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2" />
        <circle cx="12" cy="7" r="4" />
      </>
    ),
  },
  {
    id: 'photo',
    label: 'Фото',
    icon: (
      <>
        <path d="M23 19a2 2 0 01-2 2H3a2 2 0 01-2-2V8a2 2 0 012-2h4l2-3h6l2 3h4a2 2 0 012 2z" />
        <circle cx="12" cy="13" r="4" />
      </>
    ),
  },
  {
    id: 'events_tab',
    label: 'Події',
    icon: (
      <>
        <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
        <polyline points="14 2 14 8 20 8" />
      </>
    ),
  },
]
