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
import { anomalyApi } from '@/api/anomaly'

const mockClient = vi.mocked(client)

describe('anomalyApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('status', () => {
    it('calls GET /anomaly/status', async () => {
      const response = { river_available: true, model_count: 2, models: {} }
      mockClient.get.mockResolvedValue(response)
      const result = await anomalyApi.status()
      expect(mockClient.get).toHaveBeenCalledWith('/anomaly/status')
      expect(result).toEqual(response)
    })
  })

  describe('detect', () => {
    it('calls POST /anomaly/detect with data', async () => {
      const data = { metric_name: 'cpu_usage', value: 95.5 }
      const response = { anomaly: true, result: {} }
      mockClient.post.mockResolvedValue(response)
      const result = await anomalyApi.detect(data)
      expect(mockClient.post).toHaveBeenCalledWith('/anomaly/detect', data)
      expect(result).toEqual(response)
    })
  })

  describe('setThreshold', () => {
    it('calls POST /anomaly/thresholds with data', async () => {
      const data = { metric_name: 'cpu_usage', op: '>', value: 90 }
      mockClient.post.mockResolvedValue({})
      const result = await anomalyApi.setThreshold(data)
      expect(mockClient.post).toHaveBeenCalledWith('/anomaly/thresholds', data)
      expect(result).toEqual({})
    })
  })
})
