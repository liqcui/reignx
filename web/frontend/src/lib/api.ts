import axios from 'axios'

// API base URL - will use relative path since frontend is served by same backend
const API_BASE_URL = '/api/v1'

// Create axios instance with default config
export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
apiClient.interceptors.request.use(
  (config) => {
    const authStorage = localStorage.getItem('auth-storage')
    if (authStorage) {
      try {
        const { state } = JSON.parse(authStorage)
        if (state?.token) {
          config.headers.Authorization = `Bearer ${state.token}`
        }
      } catch (error) {
        console.error('Failed to parse auth storage:', error)
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear auth and redirect to login
      localStorage.removeItem('auth-storage')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// Auth API
export const authAPI = {
  login: async (username: string, password: string) => {
    const response = await apiClient.post('/auth/login', { username, password })
    return response.data
  },
  logout: async () => {
    const response = await apiClient.post('/auth/logout')
    return response.data
  },
}

// Servers API
export const serversAPI = {
  list: async () => {
    const response = await apiClient.get('/servers')
    return response.data
  },
  get: async (id: string) => {
    const response = await apiClient.get(`/servers/${id}`)
    return response.data
  },
  create: async (data: any) => {
    const response = await apiClient.post('/servers', data)
    return response.data
  },
  update: async (id: string, data: any) => {
    const response = await apiClient.put(`/servers/${id}`, data)
    return response.data
  },
  delete: async (id: string) => {
    const response = await apiClient.delete(`/servers/${id}`)
    return response.data
  },
  powerAction: async (id: string, action: string) => {
    const response = await apiClient.post(`/servers/${id}/power/${action}`)
    return response.data
  },
  executeCommand: async (id: string, command: string) => {
    const response = await apiClient.post(`/servers/${id}/execute`, { command })
    return response.data
  },
  deployPackage: async (id: string, packageData: any) => {
    const response = await apiClient.post(`/servers/${id}/deploy`, packageData)
    return response.data
  },
  installOS: async (id: string, osData: any) => {
    const response = await apiClient.post(`/servers/${id}/install-os`, osData)
    return response.data
  },
}

// Jobs API
export const jobsAPI = {
  list: async () => {
    const response = await apiClient.get('/jobs')
    return response.data
  },
  get: async (id: string) => {
    const response = await apiClient.get(`/jobs/${id}`)
    return response.data
  },
  create: async (data: any) => {
    const response = await apiClient.post('/jobs', data)
    return response.data
  },
  cancel: async (id: string) => {
    const response = await apiClient.delete(`/jobs/${id}`)
    return response.data
  },
}

// Tasks API
export const tasksAPI = {
  list: async () => {
    const response = await apiClient.get('/tasks')
    return response.data
  },
  get: async (id: string) => {
    const response = await apiClient.get(`/tasks/${id}`)
    return response.data
  },
  retry: async (id: string) => {
    const response = await apiClient.post(`/tasks/${id}/retry`)
    return response.data
  },
}

// Metrics API
export const metricsAPI = {
  getDashboard: async () => {
    const response = await apiClient.get('/metrics/dashboard')
    return response.data
  },
}
