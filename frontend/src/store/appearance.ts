import { create } from 'zustand'

type Theme = 'light' | 'dark'

interface AppearanceState {
  theme: Theme
  sidebarPinned: boolean
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
  setSidebarPinned: (pinned: boolean) => void
  initAppearance: () => void
}

const getStoredTheme = (): Theme => {
  if (typeof window === 'undefined') return 'light'
  return (localStorage.getItem('theme') as Theme) || 'light'
}

const getStoredPinned = () => {
  if (typeof window === 'undefined') return false
  return localStorage.getItem('sidebar_pinned') === 'true'
}

const applyTheme = (theme: Theme) => {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('dark', theme === 'dark')
}

export const useAppearanceStore = create<AppearanceState>((set, get) => ({
  theme: getStoredTheme(),
  sidebarPinned: getStoredPinned(),
  setTheme: (theme) => {
    localStorage.setItem('theme', theme)
    applyTheme(theme)
    set({ theme })
  },
  toggleTheme: () => {
    const next = get().theme === 'dark' ? 'light' : 'dark'
    localStorage.setItem('theme', next)
    applyTheme(next)
    set({ theme: next })
  },
  setSidebarPinned: (pinned) => {
    localStorage.setItem('sidebar_pinned', String(pinned))
    set({ sidebarPinned: pinned })
  },
  initAppearance: () => {
    const theme = getStoredTheme()
    const sidebarPinned = getStoredPinned()
    applyTheme(theme)
    set({ theme, sidebarPinned })
  },
}))
