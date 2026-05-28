import client from './client'

export interface MetricSnapshot {
  name: string
  value: number
  unit?: string
  change_pct?: number
}

export interface SummaryResponse {
  status: 'normal' | 'warning' | 'critical'
  summary: string
  details: string[]
  recommendations: string[]
  source: string
}

export interface ChatResponse {
  response: string
  source: string
}

export const llmApi = {
  summary: (data: {
    metrics?: MetricSnapshot[]
    log_summary?: { error_count: number; top_errors?: string[] }
    incidents?: { severity: string; title: string }[]
  }) => client.post<any, SummaryResponse>('/llm/summary', data),

  chat: (message: string) =>
    client.post<any, ChatResponse>('/llm/chat', { message }),
}
