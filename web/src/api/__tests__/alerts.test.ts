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
import { alertsApi } from '@/api/alerts'

const mockClient = vi.mocked(client)

describe('alertsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('listEvents', () => {
    it('calls GET /alerts/events with params', async () => {
      const response = { data: [], total: 0 }
      mockClient.get.mockResolvedValue(response)
      const result = await alertsApi.listEvents({ status: 'firing', limit: 10, offset: 0 })
      expect(mockClient.get).toHaveBeenCalledWith('/alerts/events', { params: { status: 'firing', limit: 10, offset: 0 } })
      expect(result).toEqual(response)
    })

    it('works without params', async () => {
      mockClient.get.mockResolvedValue({ data: [], total: 0 })
      await alertsApi.listEvents()
      expect(mockClient.get).toHaveBeenCalledWith('/alerts/events', { params: undefined })
    })
  })

  describe('listIncidents', () => {
    it('calls GET /alerts/incidents with params', async () => {
      const response = { data: [], total: 0 }
      mockClient.get.mockResolvedValue(response)
      const result = await alertsApi.listIncidents({ status: 'open', limit: 20, offset: 5 })
      expect(mockClient.get).toHaveBeenCalledWith('/alerts/incidents', { params: { status: 'open', limit: 20, offset: 5 } })
      expect(result).toEqual(response)
    })
  })

  describe('acknowledge', () => {
    it('calls POST /alerts/incidents/{id}/acknowledge', async () => {
      mockClient.post.mockResolvedValue({})
      await alertsApi.acknowledge(42)
      expect(mockClient.post).toHaveBeenCalledWith('/alerts/incidents/42/acknowledge')
    })
  })

  describe('resolve', () => {
    it('calls POST /alerts/incidents/{id}/resolve', async () => {
      mockClient.post.mockResolvedValue({})
      await alertsApi.resolve(7)
      expect(mockClient.post).toHaveBeenCalledWith('/alerts/incidents/7/resolve')
    })
  })

  describe('listRules', () => {
    it('calls GET /alerts/rules with params', async () => {
      const response = { data: [], total: 0 }
      mockClient.get.mockResolvedValue(response)
      const result = await alertsApi.listRules({ limit: 50, offset: 10 })
      expect(mockClient.get).toHaveBeenCalledWith('/alerts/rules', { params: { limit: 50, offset: 10 } })
      expect(result).toEqual(response)
    })

    it('works without params', async () => {
      mockClient.get.mockResolvedValue({ data: [], total: 0 })
      await alertsApi.listRules()
      expect(mockClient.get).toHaveBeenCalledWith('/alerts/rules', { params: undefined })
    })
  })

  describe('getRule', () => {
    it('calls GET /alerts/rules/{id}', async () => {
      const rule = { id: 1, name: 'test' }
      mockClient.get.mockResolvedValue(rule)
      const result = await alertsApi.getRule(1)
      expect(mockClient.get).toHaveBeenCalledWith('/alerts/rules/1')
      expect(result).toEqual(rule)
    })
  })

  describe('createRule', () => {
    it('calls POST /alerts/rules with data', async () => {
      const data = { name: 'test', rule_type: 'threshold' }
      const created = { id: 1, ...data }
      mockClient.post.mockResolvedValue(created)
      const result = await alertsApi.createRule(data)
      expect(mockClient.post).toHaveBeenCalledWith('/alerts/rules', data)
      expect(result).toEqual(created)
    })
  })

  describe('updateRule', () => {
    it('calls PUT /alerts/rules/{id} with data', async () => {
      const data = { name: 'updated' }
      const updated = { id: 1, name: 'updated' }
      mockClient.put.mockResolvedValue(updated)
      const result = await alertsApi.updateRule(1, data)
      expect(mockClient.put).toHaveBeenCalledWith('/alerts/rules/1', data)
      expect(result).toEqual(updated)
    })
  })

  describe('deleteRule', () => {
    it('calls DELETE /alerts/rules/{id}', async () => {
      mockClient.delete.mockResolvedValue({})
      await alertsApi.deleteRule(5)
      expect(mockClient.delete).toHaveBeenCalledWith('/alerts/rules/5')
    })
  })
})
