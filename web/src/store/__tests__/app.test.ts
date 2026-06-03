import { vi, describe, it, expect, beforeEach } from 'vitest'
import { useAppStore } from '@/store/app'

// Reset store between tests
beforeEach(() => {
  localStorage.clear()
  useAppStore.setState({
    theme: 'light',
    timeRange: { start: 'now-1h', end: 'now', label: '最近 1 小时' },
    sidebarCollapsed: false,
    currentModule: 'dashboard',
    breadcrumbs: [],
    searchVisible: false,
    notificationCount: 0,
    assistantVisible: false,
    token: null,
    user: null,
  })
})

describe('useAppStore', () => {
  describe('initial state', () => {
    it('has correct defaults', () => {
      const state = useAppStore.getState()
      expect(state.theme).toBe('light')
      expect(state.sidebarCollapsed).toBe(false)
      expect(state.currentModule).toBe('dashboard')
      expect(state.token).toBeNull()
      expect(state.user).toBeNull()
    })
  })

  describe('toggleTheme', () => {
    it('toggles from light to dark', () => {
      useAppStore.getState().toggleTheme()
      expect(useAppStore.getState().theme).toBe('dark')
      expect(localStorage.getItem('theme')).toBe('dark')
    })

    it('toggles from dark to light', () => {
      useAppStore.setState({ theme: 'dark' })
      useAppStore.getState().toggleTheme()
      expect(useAppStore.getState().theme).toBe('light')
      expect(localStorage.getItem('theme')).toBe('light')
    })
  })

  describe('toggleSidebar', () => {
    it('toggles sidebar collapsed state', () => {
      expect(useAppStore.getState().sidebarCollapsed).toBe(false)
      useAppStore.getState().toggleSidebar()
      expect(useAppStore.getState().sidebarCollapsed).toBe(true)
      useAppStore.getState().toggleSidebar()
      expect(useAppStore.getState().sidebarCollapsed).toBe(false)
    })
  })

  describe('setAuth', () => {
    it('stores token in localStorage and updates state', () => {
      const user = { username: 'admin', role: 'admin', userId: 1, tenantId: 1 }
      useAppStore.getState().setAuth('my-token', user)
      expect(localStorage.getItem('token')).toBe('my-token')
      expect(useAppStore.getState().token).toBe('my-token')
      expect(useAppStore.getState().user).toEqual(user)
    })
  })

  describe('logout', () => {
    it('clears token from localStorage and resets state', () => {
      localStorage.setItem('token', 'existing')
      useAppStore.setState({ token: 'existing', user: { username: 'admin', role: 'admin' } })
      useAppStore.getState().logout()
      expect(localStorage.getItem('token')).toBeNull()
      expect(useAppStore.getState().token).toBeNull()
      expect(useAppStore.getState().user).toBeNull()
    })
  })
})
