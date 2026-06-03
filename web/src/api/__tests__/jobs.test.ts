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
import { jobsApi } from '@/api/jobs'

const mockClient = vi.mocked(client)

describe('jobsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('calls GET /jobs with params', async () => {
      const response = { data: [] }
      mockClient.get.mockResolvedValue(response)
      const result = await jobsApi.list({ limit: 10 })
      expect(mockClient.get).toHaveBeenCalledWith('/jobs', { params: { limit: 10 } })
      expect(result).toEqual(response)
    })

    it('works without params', async () => {
      mockClient.get.mockResolvedValue({ data: [] })
      await jobsApi.list()
      expect(mockClient.get).toHaveBeenCalledWith('/jobs', { params: undefined })
    })
  })

  describe('create', () => {
    it('calls POST /jobs with data', async () => {
      const data = { name: 'test-job', job_type: 'script', content: 'echo hi' }
      const created = { id: 1, ...data }
      mockClient.post.mockResolvedValue(created)
      const result = await jobsApi.create(data)
      expect(mockClient.post).toHaveBeenCalledWith('/jobs', data)
      expect(result).toEqual(created)
    })
  })

  describe('run', () => {
    it('calls POST /jobs/{id}/run', async () => {
      mockClient.post.mockResolvedValue({})
      await jobsApi.run(7)
      expect(mockClient.post).toHaveBeenCalledWith('/jobs/7/run')
    })
  })

  describe('delete', () => {
    it('calls DELETE /jobs/{id}', async () => {
      mockClient.delete.mockResolvedValue({})
      await jobsApi.delete(3)
      expect(mockClient.delete).toHaveBeenCalledWith('/jobs/3')
    })
  })

  describe('executions', () => {
    it('calls GET /jobs/{id}/executions', async () => {
      const response = { data: [] }
      mockClient.get.mockResolvedValue(response)
      const result = await jobsApi.executions(5)
      expect(mockClient.get).toHaveBeenCalledWith('/jobs/5/executions')
      expect(result).toEqual(response)
    })
  })
})
