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
import { tracesApi } from '@/api/traces'

const mockClient = vi.mocked(client)

describe('tracesApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('calls GET /traces with params', async () => {
      const response = { data: [] }
      mockClient.get.mockResolvedValue(response)
      const result = await tracesApi.list({ limit: 20, service: 'api' })
      expect(mockClient.get).toHaveBeenCalledWith('/traces', { params: { limit: 20, service: 'api' } })
      expect(result).toEqual(response)
    })

    it('works without params', async () => {
      mockClient.get.mockResolvedValue({ data: [] })
      await tracesApi.list()
      expect(mockClient.get).toHaveBeenCalledWith('/traces', { params: undefined })
    })
  })

  describe('services', () => {
    it('calls GET /traces/services', async () => {
      const response = { data: ['api', 'web'] }
      mockClient.get.mockResolvedValue(response)
      const result = await tracesApi.services()
      expect(mockClient.get).toHaveBeenCalledWith('/traces/services')
      expect(result).toEqual(response)
    })
  })

  describe('detail', () => {
    it('calls GET /traces/{traceId}', async () => {
      const response = { data: [] }
      mockClient.get.mockResolvedValue(response)
      const result = await tracesApi.detail('abc-123')
      expect(mockClient.get).toHaveBeenCalledWith('/traces/abc-123')
      expect(result).toEqual(response)
    })
  })
})
