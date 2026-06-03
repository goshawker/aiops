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
import { metricsApi } from '@/api/metrics'

const mockClient = vi.mocked(client)

describe('metricsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('query', () => {
    it('calls POST /metrics/query with full query object', async () => {
      const response = { data: [] }
      mockClient.post.mockResolvedValue(response)
      const params = { query: 'cpu_usage', start: '2024-01-01', end: '2024-01-02', step: '1m' }
      const result = await metricsApi.query(params)
      expect(mockClient.post).toHaveBeenCalledWith('/metrics/query', params)
      expect(result).toEqual(response)
    })

    it('works with only required query field', async () => {
      mockClient.post.mockResolvedValue({ data: [] })
      const params = { query: 'mem_usage' }
      await metricsApi.query(params)
      expect(mockClient.post).toHaveBeenCalledWith('/metrics/query', params)
    })
  })

  describe('instant', () => {
    it('calls POST /metrics/query with only query string', async () => {
      const response = { data: [] }
      mockClient.post.mockResolvedValue(response)
      const result = await metricsApi.instant('cpu_usage')
      expect(mockClient.post).toHaveBeenCalledWith('/metrics/query', { query: 'cpu_usage' })
      expect(result).toEqual(response)
    })
  })
})
