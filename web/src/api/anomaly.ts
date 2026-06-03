import client from './client'

export interface ModelStatus {
  river_available: boolean
  model_count: number
  models: Record<string, {
    point_count: number
    warmup_done: boolean
    has_river_model: boolean
    recent_score: number
  }>
}

export interface DetectResult {
  anomaly: boolean
  result: Record<string, unknown>
}

export interface ThresholdForm {
  metric_name: string
  op: string
  value: number
}

export const anomalyApi = {
  status: () =>
    client.get<any, ModelStatus>('/anomaly/status'),

  detect: (data: { metric_name: string; value: number }) =>
    client.post<any, DetectResult>('/anomaly/detect', data),

  setThreshold: (data: ThresholdForm) =>
    client.post<any, unknown>('/anomaly/thresholds', data),
}
