import axios from 'axios'
import wsService from '@/lib/websocket'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001'

export const api = axios.create({
  baseURL: `${API_URL}/api`,
  headers: {
    'Content-Type': 'application/json',
  },
})

const emitRequestActivity = (active: boolean) => {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent('api:activity', { detail: { active } }))
}

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    emitRequestActivity(true)
    const token = localStorage.getItem('access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
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

      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken) {
        try {
          const response = await axios.post(`${API_URL}/api/auth/refresh`, {
            refresh_token: refreshToken,
          })

          const { access_token, refresh_token: newRefreshToken } = response.data
          localStorage.setItem('access_token', access_token)
          localStorage.setItem('refresh_token', newRefreshToken)

          originalRequest.headers.Authorization = `Bearer ${access_token}`
          return api(originalRequest)
        } catch (refreshError: any) {
          if (refreshError.response?.status === 429 || refreshError.response?.status >= 500) {
            return Promise.reject(refreshError)
          }

          wsService.pauseReconnect()
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          window.location.href = '/login'
        }
      } else {
        window.location.href = '/login'
      }
    }

    return Promise.reject(error)
  }
)

export default api
