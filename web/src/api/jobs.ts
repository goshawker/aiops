import client from './client'

export interface Job {
  id: number
  name: string
  description: string
  job_type: string
  content: string
  schedule: string
  enabled: boolean
  status: string
  timeout: number
  last_run_at: string | null
}

export interface JobExecution {
  id: number
  job_id: number
  status: string
  output: string
  error: string
  duration: number
  started_at: string
  ended_at: string | null
}

export interface JobCreateForm {
  name: string
  description?: string
  job_type: string
  content: string
  schedule?: string
  timeout?: number
}

export const jobsApi = {
  list: (params?: { limit?: number }) =>
    client.get<any, { data: Job[] }>('/jobs', { params }),

  create: (data: JobCreateForm) =>
    client.post<any, Job>('/jobs', data),

  run: (id: number) =>
    client.post(`/jobs/${id}/run`),

  delete: (id: number) =>
    client.delete(`/jobs/${id}`),

  executions: (id: number) =>
    client.get<any, { data: JobExecution[] }>(`/jobs/${id}/executions`),
}
