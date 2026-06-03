import client from './client'

export interface RootCause {
  metric_name: string
  score: number
  reason: string
  related_metrics: string[]
  evidence: string[]
}

export interface CausalGraph {
  nodes: string[]
  edges: Array<{
    source: string
    target: string
    confidence: number
    lag: number
  }>
}

export interface AnalyzeRequest {
  affected_metrics: string[]
}

export interface AnalyzeResponse {
  root_causes: RootCause[]
}

export const rcaApi = {
  analyze: (data: AnalyzeRequest) =>
    client.post<any, AnalyzeResponse>('/rca/analyze', data),

  graph: () =>
    client.get<any, CausalGraph>('/rca/graph'),
}
