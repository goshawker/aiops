import client from './client'

// ── Types ──────────────────────────────────────────────────────

export interface User {
  id: number
  username: string
  display_name: string
  email: string
  role: string
  status: string
  tenant_id: number
  created_at: string
  updated_at: string
}

export interface AuditLog {
  id: number
  user_id: number
  username: string
  action: string
  resource: string
  resource_id: string
  detail: string
  ip: string
  created_at: string
}

export interface Tenant {
  id: number
  name: string
  code: string
  status: string
  plan: string
  max_hosts: number
  max_users: number
  settings: string
  created_at: string
  updated_at: string
}

export interface LoginResponse {
  token: string
  user_id: number
  username: string
  display_name: string
  role: string
  tenant_id: number
}

// ── API ────────────────────────────────────────────────────────

export const adminApi = {
  // Auth
  login: (username: string, password: string) =>
    client.post<any, LoginResponse>('/auth/login', { username, password }),

  // Users
  listUsers: (params?: { limit?: number; offset?: number }) =>
    client.get<any, { data: User[]; total: number }>('/users', { params }),

  createUser: (data: { username: string; password: string; display_name?: string; email?: string; role?: string }) =>
    client.post<any, User>('/users', data),

  updateUser: (id: number, data: { display_name?: string; email?: string; role?: string; status?: string }) =>
    client.put<any, User>(`/users/${id}`, data),

  deleteUser: (id: number) =>
    client.delete(`/users/${id}`),

  // Audit Logs
  listAuditLogs: (params?: { limit?: number; offset?: number; user_id?: number; action?: string }) =>
    client.get<any, { data: AuditLog[]; total: number }>('/audit-logs', { params }),

  // Tenants
  listTenants: (params?: { limit?: number; offset?: number }) =>
    client.get<any, { data: Tenant[]; total: number }>('/tenants', { params }),

  createTenant: (data: { name: string; code: string; plan?: string; max_hosts?: number; max_users?: number }) =>
    client.post<any, Tenant>('/tenants', data),

  updateTenant: (id: number, data: { name?: string; code?: string; status?: string; plan?: string; max_hosts?: number; max_users?: number }) =>
    client.put<any, Tenant>(`/tenants/${id}`, data),

  deleteTenant: (id: number) =>
    client.delete(`/tenants/${id}`),
}
