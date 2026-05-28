import client from './client'

export interface LogEntry {
  timestamp: string
  level: string
  service: string
  host: string
  message: string
  trace_id?: string
  span_id?: string
}

export interface LogSearchParams {
  q?: string
  service?: string
  host?: string
  level?: string
  trace_id?: string
  start?: string
  end?: string
  limit?: number
  offset?: number
}

export const logsApi = {
  search: (params: LogSearchParams) =>
    client.get<any, { data: LogEntry[]; total: number }>('/logs/search', { params }),
}
