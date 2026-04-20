import { create } from 'zustand'

export type OperatorTab = 'object' | 'events' | 'alarms'
export type MainTab = 'objects' | 'signals'
export type InnerTab = 'notes' | 'extra' | 'subs' | 'rent'
export type BottomTab = 'unproc' | 'archive'
export type StatusFilter = 'all' | 'guarded' | 'unguarded' | 'late' | 'call' | 'alarm' | 'banned'
export type ModalTab = 'kartochka' | 'devices' | 'zones' | 'response' | 'keys' | 'resp' | 'photo' | 'events_tab'

type OperatorUIState = {
  mainTab: MainTab
  innerTab: InnerTab
  bottomTab: BottomTab
  statusFilter: StatusFilter
  searchValue: string
  selectedObjectID: number | null
  selectedSignalRowID: string | null
  isCardModalOpen: boolean
  isEventModalOpen: boolean
  cardModalTab: ModalTab
  eventModalTab: ModalTab
  eventModalRowID: string | null
  activeTab: OperatorTab
  setMainTab: (tab: MainTab) => void
  setInnerTab: (tab: InnerTab) => void
  setBottomTab: (tab: BottomTab) => void
  setStatusFilter: (filter: StatusFilter) => void
  setSearchValue: (value: string) => void
  setSelectedObjectID: (id: number | null) => void
  setSelectedSignalRowID: (rowID: string | null) => void
  setIsCardModalOpen: (open: boolean) => void
  setIsEventModalOpen: (open: boolean) => void
  setCardModalTab: (tab: ModalTab) => void
  setEventModalTab: (tab: ModalTab) => void
  setEventModalRowID: (rowID: string | null) => void
  setActiveTab: (tab: OperatorTab) => void
}

export const useOperatorUIStore = create<OperatorUIState>((set) => ({
  mainTab: 'signals',
  innerTab: 'notes',
  bottomTab: 'unproc',
  statusFilter: 'all',
  searchValue: '',
  selectedObjectID: null,
  selectedSignalRowID: null,
  isCardModalOpen: false,
  isEventModalOpen: false,
  cardModalTab: 'kartochka',
  eventModalTab: 'kartochka',
  eventModalRowID: null,
  activeTab: 'object',
  setMainTab: (tab) => set({ mainTab: tab }),
  setInnerTab: (tab) => set({ innerTab: tab }),
  setBottomTab: (tab) => set({ bottomTab: tab }),
  setStatusFilter: (filter) => set({ statusFilter: filter }),
  setSearchValue: (value) => set({ searchValue: value }),
  setSelectedObjectID: (id) => set({ selectedObjectID: id }),
  setSelectedSignalRowID: (rowID) => set({ selectedSignalRowID: rowID }),
  setIsCardModalOpen: (open) => set({ isCardModalOpen: open }),
  setIsEventModalOpen: (open) => set({ isEventModalOpen: open }),
  setCardModalTab: (tab) => set({ cardModalTab: tab }),
  setEventModalTab: (tab) => set({ eventModalTab: tab }),
  setEventModalRowID: (rowID) => set({ eventModalRowID: rowID }),
  setActiveTab: (tab) => set({ activeTab: tab }),
}))
