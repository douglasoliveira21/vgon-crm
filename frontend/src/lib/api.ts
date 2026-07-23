import axios from 'axios'
import wsService from '@/lib/websocket'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001'

export const api = axios.create({
  baseURL: `${API_URL}/api`,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

const emitRequestActivity = (active: boolean) => {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent('api:activity', { detail: { active } }))
}

api.interceptors.request.use(
  (config) => {
    emitRequestActivity(true)
    return config
  },
  (error) => {
    emitRequestActivity(false)
    return Promise.reject(error)
  }
)

// Response interceptor for token refresh
api.interceptors.response.use(
  (response) => {
    emitRequestActivity(false)
    return response
  },
  async (error) => {
    emitRequestActivity(false)
    const originalRequest = error.config

    if (error.response?.status === 429) {
      return Promise.reject(error)
    }

    if (error.response?.status === 401 && !originalRequest._retry) {
      if (originalRequest.url?.includes('/auth/login') || originalRequest.url?.includes('/auth/refresh')) {
        return Promise.reject(error)
      }

      originalRequest._retry = true

      try {
        await axios.post(`${API_URL}/api/auth/refresh`, {}, { withCredentials: true })
        wsService.connect()
        return api(originalRequest)
      } catch (refreshError: any) {
        if (refreshError.response?.status === 429 || refreshError.response?.status >= 500) {
          return Promise.reject(refreshError)
        }
        wsService.pauseReconnect()
        window.location.href = '/login'
      }
    }

    return Promise.reject(error)
  }
)

export default api
