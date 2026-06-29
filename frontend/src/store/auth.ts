import { create } from 'zustand'
import api from '@/lib/api'
import wsService from '@/lib/websocket'

interface User {
  id: string
  name: string
  email: string
  company_id: string
  role_slug: string
  role_name: string
  avatar_url?: string
  phone?: string
  is_online: boolean
  availability_status?: 'online' | 'offline' | 'busy'
  is_super_admin?: boolean
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (companyName: string, name: string, email: string, password: string) => Promise<void>
  updateUser: (user: User) => void
  updateProfile: (data: { name: string; phone?: string }) => Promise<void>
  updateStatus: (status: 'online' | 'offline' | 'busy') => Promise<void>
  uploadAvatar: (file: File) => Promise<void>
  logout: () => void
  checkAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: typeof window !== 'undefined' && !!localStorage.getItem('access_token'),
  isLoading: false,

  login: async (email: string, password: string) => {
    set({ isLoading: true })
    try {
      const response = await api.post('/auth/login', { email, password })
      const { access_token, refresh_token, user } = response.data

      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)

      wsService.connect(access_token)

      set({ user, isAuthenticated: true, isLoading: false })
    } catch (error: any) {
      set({ isLoading: false })
      throw new Error(error.response?.data?.error || 'Erro ao fazer login')
    }
  },

  register: async (companyName: string, name: string, email: string, password: string) => {
    set({ isLoading: true })
    try {
      const response = await api.post('/auth/register', {
        company_name: companyName,
        name,
        email,
        password,
      })
      const { access_token, refresh_token, user } = response.data

      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)

      wsService.connect(access_token)

      set({ user, isAuthenticated: true, isLoading: false })
    } catch (error: any) {
      set({ isLoading: false })
      throw new Error(error.response?.data?.error || 'Erro ao criar conta')
    }
  },

  updateUser: (user: User) => set({ user }),

  updateProfile: async (data) => {
    const response = await api.put('/me', data)
    set({ user: response.data })
  },

  updateStatus: async (status) => {
    const response = await api.put('/me/status', { status })
    set({ user: response.data })
  },

  uploadAvatar: async (file) => {
    const formData = new FormData()
    formData.append('avatar', file)
    const response = await api.post('/me/avatar', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    set({ user: response.data })
  },

  logout: () => {
    api.put('/me/status', { status: 'offline' }).catch(() => {})
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    wsService.disconnect()
    set({ user: null, isAuthenticated: false })
  },

  checkAuth: () => {
    const token = localStorage.getItem('access_token')
    if (token) {
      wsService.connect(token)
      api.get('/me').then((response) => {
        set({ user: response.data, isAuthenticated: true })
      }).catch((error) => {
        if (error.response?.status === 429) return
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        set({ user: null, isAuthenticated: false })
      })
    }
  },
}))
