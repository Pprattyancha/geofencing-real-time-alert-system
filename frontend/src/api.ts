import type { AlertRule, Geofence, Vehicle, Violation } from './types'

export const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')
export const WS_URL = API_URL.replace(/^http/, 'ws') + '/ws/alerts'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options?.headers || {}) },
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`)
  return data as T
}
const post = (path: string, body: unknown) => request(path, { method: 'POST', body: JSON.stringify(body) })

export const api = {
  health: () => request<{status:string}>('/health'),
  geofences: (category = '') => request<{geofences:Geofence[]}>(`/geofences${category ? `?category=${encodeURIComponent(category)}` : ''}`),
  createGeofence: (body: Omit<Geofence,'id'>) => post('/geofences', body),
  vehicles: () => request<{vehicles:Vehicle[]}>('/vehicles'),
  createVehicle: (body: Omit<Vehicle,'id'|'status'>) => post('/vehicles', body),
  updateLocation: (body: {vehicle_id:string; latitude:number; longitude:number; timestamp:string}) => post('/vehicles/location', body),
  configureAlert: (body: {geofence_id:string; vehicle_id?:string; event_type:string}) => post('/alerts/configure', body),
  alerts: () => request<{alerts:AlertRule[]}>('/alerts'),
  history: (filters: {vehicle_id?:string; geofence_id?:string; limit?:number} = {}) => {
    const params = new URLSearchParams()
    Object.entries(filters).forEach(([k,v]) => { if (v) params.set(k,String(v)) })
    return request<{violations:Violation[];total_count:number}>(`/violations/history?${params}`)
  },
}