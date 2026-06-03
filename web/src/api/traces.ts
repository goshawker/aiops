import client from './client'

export interface TraceSummary {
  trace_id: string
  root_service: string
  root_operation: string
  span_count: number
  duration_ms: number
  status_code: string
  start_time: string
}

export interface Span {
  timestamp: string
  trace_id: string
  span_id: string
  parent_span_id: string
  service: string
  operation: string
  duration_ms: number
  status_code: string
  attributes: Record<string, string>
}

export interface TraceQueryParams {
  limit?: number
  service?: string
}

export const tracesApi = {
  list: (params?: TraceQueryParams) =>
    client.get<any, { data: TraceSummary[] }>('/traces', { params }),

  services: () =>
    client.get<any, { data: string[] }>('/traces/services'),

  detail: (traceId: string) =>
    client.get<any, { data: Span[] }>(`/traces/${traceId}`),
}
