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

// Multiple requests can 401 at almost the same moment (e.g. several fetches
// firing together when switching between chats). Without this, each one
// independently POSTs /auth/refresh — and since the backend deletes the old
// refresh token and issues a new one, the second concurrent call finds the
// token already gone and fails, leaving a dead refresh cookie behind that
// F5 keeps resending forever (only a fresh login/incognito gets a new one).
// Sharing a single in-flight refresh promise means only one call ever goes
// out per expiry, so there's nothing left to race.
let refreshPromise: Promise<void> | null = null

const refreshAccessToken = (): Promise<void> => {
  if (!refreshPromise) {
    refreshPromise = axios
      .post(`${API_URL}/api/auth/refresh`, {}, { withCredentials: true })
      .then(() => {
        wsService.connect()
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
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
        await refreshAccessToken()
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
