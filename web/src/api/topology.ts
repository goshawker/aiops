import client from './client'

export interface GraphEdge {
  source: string
  target: string
  confidence: number
  lag: number
}

export interface GraphData {
  nodes: string[]
  edges: GraphEdge[]
  metric_count: number
  last_discovery: number
}

export const topologyApi = {
  graph: () =>
    client.get<any, GraphData>('/rca/graph'),
}
