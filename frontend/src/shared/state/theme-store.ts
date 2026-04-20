import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'

export type ThemeMode = 'dark' | 'light'

type ThemeState = {
  themeMode: ThemeMode
  setThemeMode: (mode: ThemeMode) => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      themeMode: 'dark',
      setThemeMode: (mode) => set({ themeMode: mode }),
    }),
    {
      name: 'operator-theme',
      storage: createJSONStorage(() => localStorage),
    },
  ),
)
