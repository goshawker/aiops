import client from './client'

export interface Collector {
  id: number
  name: string
  hostname: string
  ip: string
  version: string
  status: string
  last_heartbeat: string | null
  tags: string
  created_at: string
  updated_at: string
}

export interface CollectorStatus {
  total: number
  online: number
  offline: number
}

export const collectorsApi = {
  list: (params?: { limit?: number; offset?: number; status?: string }) =>
    client.get('/collectors', { params }) as Promise<{ data: Collector[]; total: number }>,

  status: () =>
    client.get('/collectors/status') as Promise<CollectorStatus>,
}
