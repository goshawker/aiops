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
import { rcaApi } from '@/api/rca'

const mockClient = vi.mocked(client)

describe('rcaApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('analyze', () => {
    it('calls POST /rca/analyze with data', async () => {
      const data = { affected_metrics: ['cpu_usage', 'mem_usage'] }
      const response = { root_causes: [] }
      mockClient.post.mockResolvedValue(response)
      const result = await rcaApi.analyze(data)
      expect(mockClient.post).toHaveBeenCalledWith('/rca/analyze', data)
      expect(result).toEqual(response)
    })
  })

  describe('graph', () => {
    it('calls GET /rca/graph', async () => {
      const response = { nodes: [], edges: [] }
      mockClient.get.mockResolvedValue(response)
      const result = await rcaApi.graph()
      expect(mockClient.get).toHaveBeenCalledWith('/rca/graph')
      expect(result).toEqual(response)
    })
  })
})
