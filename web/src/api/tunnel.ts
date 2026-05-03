import api from './index'

export interface Tunnel {
  id?: number
  tunnel_id: string
  name: string
  type: string
  transport_mode: string
  local_addr: string
  remote_addr: string
  obfuscation_mode: string
  obfuscation_config?: string
  e2e_enabled: boolean
  tls_fingerprint: boolean
  tls_profile: string
  status: string
  node_id: string
  bytes_in?: number
  bytes_out?: number
  created_at?: string
  updated_at?: string
}

export interface CreateTunnelRequest {
  name: string
  type: string
  transport_mode?: string
  local_addr?: string
  remote_addr?: string
  obfuscation_mode?: string
  obfuscation_config?: string
  e2e_enabled?: boolean
  tls_fingerprint?: boolean
  tls_profile?: string
  node_id: string
}

export interface UpdateTunnelRequest {
  name?: string
  type?: string
  transport_mode?: string
  local_addr?: string
  remote_addr?: string
  obfuscation_mode?: string
  obfuscation_config?: string
  e2e_enabled?: boolean
  tls_fingerprint?: boolean
  tls_profile?: string
  status?: string
}

export interface TunnelStats {
  tunnel_id: string
  bytes_in: number
  bytes_out: number
  status: string
}

export const tunnelApi = {
  list: async (): Promise<Tunnel[]> => {
    const res = await api.get('/tunnels')
    return res.data || []
  },

  get: async (id: number): Promise<Tunnel> => {
    const res = await api.get(`/tunnels/${id}`)
    return res.data
  },

  create: async (data: CreateTunnelRequest): Promise<Tunnel> => {
    const res = await api.post('/tunnels', data)
    return res.data
  },

  update: async (id: number, data: UpdateTunnelRequest): Promise<void> => {
    await api.put(`/tunnels/${id}`, data)
  },

  delete: async (id: number): Promise<void> => {
    await api.delete(`/tunnels/${id}`)
  },

  toggle: async (id: number): Promise<{ status: string }> => {
    const res = await api.post(`/tunnels/${id}/toggle`)
    return res.data
  },

  stats: async (id: number): Promise<TunnelStats> => {
    const res = await api.get(`/tunnels/${id}/stats`)
    return res.data
  }
}

export default tunnelApi