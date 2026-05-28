import client from './client'

export interface AlertEvent {
  id: number
  rule_id: number
  rule_name: string
  source_type: string
  source: string
  host: string
  service: string
  severity: 'critical' | 'warning' | 'info'
  title: string
  message: string
  value: string
  status: 'firing' | 'resolved' | 'suppressed'
  fired_at: string
}

export interface Incident {
  id: number
  title: string
  description: string
  severity: 'critical' | 'warning' | 'info'
  affected_services: string[]
  affected_hosts: string[]
  event_count: number
  ai_summary: string
  status: 'open' | 'acknowledged' | 'resolved' | 'closed'
  created_at: string
}

export const alertsApi = {
  listEvents: (params?: { status?: string; limit?: number; offset?: number }) =>
    client.get<any, { data: AlertEvent[]; total: number }>('/alerts/events', { params }),

  listIncidents: (params?: { status?: string; limit?: number; offset?: number }) =>
    client.get<any, { data: Incident[]; total: number }>('/alerts/incidents', { params }),

  acknowledge: (id: number) =>
    client.post(`/alerts/incidents/${id}/acknowledge`),

  resolve: (id: number) =>
    client.post(`/alerts/incidents/${id}/resolve`),
}
