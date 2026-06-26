import { create } from 'zustand'
import { getClient } from '../api/client'

export type CallPhase = 'dialing' | 'active' | 'failed'

export type ActiveCall = {
  callID: string
  phone: string
  contactName: string
  phase: CallPhase
  startedAt: number
}

type CallStore = {
  activeCall: ActiveCall | null
  dialerAvailable: boolean
  dial: (phone: string, contactName: string) => Promise<void>
  hangup: () => Promise<void>
  setDialerAvailable: (v: boolean) => void
}

export const useCallStore = create<CallStore>((set, get) => ({
  activeCall: null,
  dialerAvailable: true,

  setDialerAvailable: (v) => set({ dialerAvailable: v }),

  dial: async (phone, contactName) => {
    if (get().activeCall != null) return

    set({
      activeCall: {
        callID: '',
        phone,
        contactName,
        phase: 'dialing',
        startedAt: Date.now(),
      },
    })

    try {
      const { callID } = await getClient().dialPhone(phone)
      set((s) => {
        if (s.activeCall == null) return {}
        return { activeCall: { ...s.activeCall, callID, phase: 'active' } }
      })
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      const isUnavailable = msg.includes('503') || msg.includes('not configured')
      set((s) => {
        if (s.activeCall == null) return {}
        return {
          activeCall: { ...s.activeCall, phase: 'failed' },
          dialerAvailable: !isUnavailable,
        }
      })
      // Автоматично прибрати повідомлення про помилку через 4 секунди
      setTimeout(() => {
        set((s) => (s.activeCall?.phase === 'failed' ? { activeCall: null } : {}))
      }, 4000)
    }
  },

  hangup: async () => {
    const { activeCall } = get()
    if (activeCall == null) return
    set({ activeCall: null })
    if (activeCall.callID) {
      try {
        await getClient().hangupCall(activeCall.callID)
      } catch {
        // ignore — best effort
      }
    }
  },
}))
