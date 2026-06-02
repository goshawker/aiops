import { create } from 'zustand'

export interface BreadcrumbItem {
  title: string
  path?: string
}

interface AppState {
  // Theme
  theme: 'light' | 'dark'
  toggleTheme: () => void

  // Time range
  timeRange: { start: string; end: string; label: string }
  setTimeRange: (range: { start: string; end: string; label: string }) => void

  // Sidebar
  sidebarCollapsed: boolean
  toggleSidebar: () => void

  // Current top-level module
  currentModule: string
  setCurrentModule: (module: string) => void

  // Breadcrumbs
  breadcrumbs: BreadcrumbItem[]
  setBreadcrumbs: (crumbs: BreadcrumbItem[]) => void

  // Global search
  searchVisible: boolean
  setSearchVisible: (visible: boolean) => void

  // Notifications
  notificationCount: number
  setNotificationCount: (count: number) => void

  // LLM assistant
  assistantVisible: boolean
  setAssistantVisible: (visible: boolean) => void

  // User
  token: string | null
  user: { username: string; role: string; userId?: number; tenantId?: number } | null
  setAuth: (token: string, user: { username: string; role: string; userId?: number; tenantId?: number }) => void
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

  timeRange: { start: 'now-1h', end: 'now', label: '最近 1 小时' },
  setTimeRange: (range) => set({ timeRange: range }),

  sidebarCollapsed: false,
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

  currentModule: 'dashboard',
  setCurrentModule: (module) => set({ currentModule: module }),

  breadcrumbs: [],
  setBreadcrumbs: (crumbs) => set({ breadcrumbs: crumbs }),

  searchVisible: false,
  setSearchVisible: (visible) => set({ searchVisible: visible }),

  notificationCount: 0,
  setNotificationCount: (count) => set({ notificationCount: count }),

  assistantVisible: false,
  setAssistantVisible: (visible) => set({ assistantVisible: visible }),

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
