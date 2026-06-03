import { vi, describe, it, expect, beforeEach } from 'vitest'

vi.mock('@/api/client', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

import client from '@/api/client'
import { adminApi } from '@/api/admin'

const mockClient = vi.mocked(client)

describe('adminApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('login', () => {
    it('calls POST /auth/login with credentials', async () => {
      const response = { token: 'abc', user_id: 1, username: 'admin', display_name: 'Admin', role: 'admin', tenant_id: 1 }
      mockClient.post.mockResolvedValue(response)
      const result = await adminApi.login('admin', 'pass')
      expect(mockClient.post).toHaveBeenCalledWith('/auth/login', { username: 'admin', password: 'pass' })
      expect(result).toEqual(response)
    })
  })

  describe('listUsers', () => {
    it('calls GET /users with params', async () => {
      const response = { data: [], total: 0 }
      mockClient.get.mockResolvedValue(response)
      const result = await adminApi.listUsers({ limit: 10, offset: 0 })
      expect(mockClient.get).toHaveBeenCalledWith('/users', { params: { limit: 10, offset: 0 } })
      expect(result).toEqual(response)
    })

    it('works without params', async () => {
      mockClient.get.mockResolvedValue({ data: [], total: 0 })
      await adminApi.listUsers()
      expect(mockClient.get).toHaveBeenCalledWith('/users', { params: undefined })
    })
  })

  describe('createUser', () => {
    it('calls POST /users with data', async () => {
      const data = { username: 'newuser', password: 'pass', display_name: 'New', role: 'viewer' }
      const created = { id: 1, ...data }
      mockClient.post.mockResolvedValue(created)
      const result = await adminApi.createUser(data)
      expect(mockClient.post).toHaveBeenCalledWith('/users', data)
      expect(result).toEqual(created)
    })
  })

  describe('updateUser', () => {
    it('calls PUT /users/{id} with data', async () => {
      const data = { display_name: 'Updated' }
      mockClient.put.mockResolvedValue({ id: 1, ...data })
      const result = await adminApi.updateUser(1, data)
      expect(mockClient.put).toHaveBeenCalledWith('/users/1', data)
      expect(result).toEqual({ id: 1, ...data })
    })
  })

  describe('deleteUser', () => {
    it('calls DELETE /users/{id}', async () => {
      mockClient.delete.mockResolvedValue({})
      await adminApi.deleteUser(3)
      expect(mockClient.delete).toHaveBeenCalledWith('/users/3')
    })
  })

  describe('listAuditLogs', () => {
    it('calls GET /audit-logs with params', async () => {
      const response = { data: [], total: 0 }
      mockClient.get.mockResolvedValue(response)
      const result = await adminApi.listAuditLogs({ limit: 50, offset: 0, user_id: 1, action: 'login' })
      expect(mockClient.get).toHaveBeenCalledWith('/audit-logs', { params: { limit: 50, offset: 0, user_id: 1, action: 'login' } })
      expect(result).toEqual(response)
    })

    it('works without params', async () => {
      mockClient.get.mockResolvedValue({ data: [], total: 0 })
      await adminApi.listAuditLogs()
      expect(mockClient.get).toHaveBeenCalledWith('/audit-logs', { params: undefined })
    })
  })

  describe('listTenants', () => {
    it('calls GET /tenants with params', async () => {
      const response = { data: [], total: 0 }
      mockClient.get.mockResolvedValue(response)
      const result = await adminApi.listTenants({ limit: 10, offset: 0 })
      expect(mockClient.get).toHaveBeenCalledWith('/tenants', { params: { limit: 10, offset: 0 } })
      expect(result).toEqual(response)
    })
  })

  describe('createTenant', () => {
    it('calls POST /tenants with data', async () => {
      const data = { name: 'Tenant', code: 't1', plan: 'free' }
      const created = { id: 1, ...data }
      mockClient.post.mockResolvedValue(created)
      const result = await adminApi.createTenant(data)
      expect(mockClient.post).toHaveBeenCalledWith('/tenants', data)
      expect(result).toEqual(created)
    })
  })

  describe('updateTenant', () => {
    it('calls PUT /tenants/{id} with data', async () => {
      const data = { name: 'Updated' }
      mockClient.put.mockResolvedValue({ id: 1, ...data })
      const result = await adminApi.updateTenant(1, data)
      expect(mockClient.put).toHaveBeenCalledWith('/tenants/1', data)
      expect(result).toEqual({ id: 1, ...data })
    })
  })

  describe('deleteTenant', () => {
    it('calls DELETE /tenants/{id}', async () => {
      mockClient.delete.mockResolvedValue({})
      await adminApi.deleteTenant(2)
      expect(mockClient.delete).toHaveBeenCalledWith('/tenants/2')
    })
  })
})
