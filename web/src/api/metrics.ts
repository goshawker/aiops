import client from './client'

export interface MetricsQuery {
  query: string
  start?: string
  end?: string
  step?: string
}

export interface DataPoint {
  timestamp: number
  value: number
}

export interface MetricSeries {
  metric: Record<string, string>
  values: DataPoint[]
}

export const metricsApi = {
  query: (q: MetricsQuery) =>
    client.post<any, { data: MetricSeries[] }>('/metrics/query', q),

  // Instant query helper
  instant: (query: string) =>
    client.post<any, { data: MetricSeries[] }>('/metrics/query', { query }),
}
