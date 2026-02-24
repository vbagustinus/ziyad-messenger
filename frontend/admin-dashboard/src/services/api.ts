import axios, { type AxiosInstance, type InternalAxiosRequestConfig, type AxiosResponse, AxiosError } from 'axios';

const getToken = () => {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('admin_token') ?? '';
};

import { AppConfig } from '../config';

const getBaseURL = () => {
  const envVal = process.env.NEXT_PUBLIC_ADMIN_API;
  // If NEXT_PUBLIC_ADMIN_API is set (e.g. for proxy), use it
  if (envVal && !envVal.includes('localhost')) return envVal;
  
  // Otherwise use our centralized AppConfig
  return AppConfig.adminBaseUrl;
};

export const api: AxiosInstance = axios.create({
  baseURL: getBaseURL(),
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (r: AxiosResponse) => r,
  (err: AxiosError) => {
    if (err.response?.status === 401) {
      if (typeof window !== 'undefined') {
        localStorage.removeItem('admin_token');
        window.location.href = '/login';
      }
    }
    return Promise.reject(err);
  }
);

export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ token: string; user: { id: string; username: string; role: string }; expires_at: number }>('/admin/login', { username, password }),
  me: () => api.get('/admin/me'),
  createAdmin: (data: { username: string; password: string; role?: string }) =>
    api.post<{ id: string; username: string; role: string }>('/admin/admins', data),
};

export const usersApi = {
  list: () => api.get<{ users: Array<{ id: string; username: string; full_name: string; role_id: string; department_id: string; created_at: number; updated_at: number }> }>('/admin/users'),
  create: (data: { username: string; full_name?: string; password: string; role_id?: string; department_id?: string }) => api.post('/admin/users', data),
  update: (id: string, data: { username?: string; full_name?: string; role_id?: string; department_id?: string }) => api.put(`/admin/users/${id}`, data),
  delete: (id: string) => api.delete(`/admin/users/${id}`),
  resetPassword: (id: string) => api.post(`/admin/users/${id}/reset-password`),
};

export const departmentsApi = {
  list: () => api.get<{ departments: Array<{ id: string; name: string; created_at: number; updated_at: number }> }>('/admin/departments'),
  create: (data: { name: string }) => api.post('/admin/departments', data),
  delete: (id: string) => api.delete(`/admin/departments/${id}`),
};

export const rolesApi = {
  list: () => api.get<{ roles: Array<{ id: string; name: string; permissions: string[]; created_at: number; updated_at: number }> }>('/admin/roles'),
  create: (data: { name: string; permissions?: string[] }) => api.post('/admin/roles', data),
  update: (id: string, data: { name?: string; permissions?: string[] }) => api.put(`/admin/roles/${id}`, data),
  delete: (id: string) => api.delete(`/admin/roles/${id}`),
};

export const devicesApi = {
  list: () => api.get<{ devices: Array<{ id: string; user_id: string; device_name: string; fingerprint: string; last_seen: number; created_at: number }> }>('/admin/devices'),
  delete: (id: string) => api.delete(`/admin/devices/${id}`),
};

export const channelsApi = {
  list: () => api.get<{ channels: Array<{ id: string; name: string; department_id: string; created_at: number; updated_at: number; created_by: string }> }>('/admin/channels'),
  create: (data: { name: string; department_id?: string }) => api.post('/admin/channels', data),
  delete: (id: string) => api.delete(`/admin/channels/${id}`),
};

export const monitoringApi = {
  overview: () => api.get<{
    generated_at: number;
    network: { nodes_online: number; peers_known: number; uptime_seconds: number; latency_ms: number };
    users: { total_users: number; online_now: number; active_today: number };
    messages: { messages_last_hour: number; messages_today: number; total_messages: number };
    files: { total_files: number; total_bytes: number; transfers_today: number };
    system: { go_version: string; num_cpu: number; memory_alloc_mb: number; uptime_seconds: number };
  }>('/admin/monitoring/overview'),
};

export const auditApi = {
  list: (params?: { offset?: number; limit?: number; actor_id?: string; action?: string }) =>
    api.get<{ logs: Array<{ id: number; timestamp: string; actor_id: string; actor_username: string; action: string; target_resource: string; details: string; ip_address: string }> }>('/admin/audit', { params }),
};

export const systemApi = {
  health: () => api.get<{ status: string; version: string; uptime_seconds: number; memory_alloc_mb: number }>('/admin/system/health'),
};

export const clusterApi = {
  status: () => api.get<{ cluster_id: string; leader_id: string; nodes: Array<{ node_id: string; address: string; role: string; last_seen: number; is_leader: boolean }>; total_nodes: number; healthy: boolean }>('/admin/cluster/status'),
};
