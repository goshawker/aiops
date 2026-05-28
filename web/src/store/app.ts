import { create } from 'zustand'

interface AppState {
  // Theme
  theme: 'light' | 'dark'
  toggleTheme: () => void

  // Time range
  timeRange: { start: string; end: string }
  setTimeRange: (range: { start: string; end: string }) => void

  // Sidebar
  sidebarCollapsed: boolean
  toggleSidebar: () => void

  // User
  token: string | null
  user: { username: string; role: string } | null
  setAuth: (token: string, user: { username: string; role: string }) => void
  logout: () => void
}

export const useAppStore = create<AppState>((set) => ({
  theme: (localStorage.getItem('theme') as 'light' | 'dark') || 'light',
  toggleTheme: () =>
    set((state) => {
      const theme = state.theme === 'light' ? 'dark' : 'light'
      localStorage.setItem('theme', theme)
      return { theme }
    }),

  timeRange: { start: 'now-1h', end: 'now' },
  setTimeRange: (range) => set({ timeRange: range }),

  sidebarCollapsed: false,
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

  token: localStorage.getItem('token'),
  user: null,
  setAuth: (token, user) => {
    localStorage.setItem('token', token)
    set({ token, user })
  },
  logout: () => {
    localStorage.removeItem('token')
    set({ token: null, user: null })
  },
}))
