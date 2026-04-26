import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('cc_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('cc_token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export const containersApi = {
  list: (onlyRunning = false) =>
    api.get('/containers', { params: { running: onlyRunning } }).then((r) => r.data),
  inspect: (id: string) => api.get(`/containers/${id}`).then((r) => r.data),
  start: (id: string) => api.post(`/containers/${id}/start`).then((r) => r.data),
  stop: (id: string, timeout = 10) =>
    api.post(`/containers/${id}/stop`, { timeout }).then((r) => r.data),
  remove: (id: string, force = false) =>
    api.delete(`/containers/${id}`, { params: { force } }).then((r) => r.data),
  logs: (id: string, tail = '100') =>
    api.get(`/containers/${id}/logs`, { params: { tail } }).then((r) => r.data),
  stats: (id: string) => api.get(`/containers/${id}/stats`).then((r) => r.data),
  updateLimits: (id: string, limits: { cpu_quota?: number; memory_mb?: number }) =>
    api.patch(`/containers/${id}/limits`, limits).then((r) => r.data),
}

export const projectsApi = {
  list: () => api.get('/projects').then((r) => r.data),
  get: (id: string) => api.get(`/projects/${id}`).then((r) => r.data),
  create: (data: {
    name: string
    stack: string
    db_name?: string
    db_user?: string
    db_password?: string
    app_port?: string
  }) => api.post('/projects', data).then((r) => r.data),
  up: (id: string) => api.post(`/projects/${id}/up`).then((r) => r.data),
  down: (id: string) => api.post(`/projects/${id}/down`).then((r) => r.data),
  delete: (id: string) => api.delete(`/projects/${id}`).then((r) => r.data),
  listStacks: () => api.get('/stacks').then((r) => r.data),
}

export const aiopsApi = {
  analyze: (containerId: string) =>
    api.post('/aiops/analyze', { container_id: containerId }).then((r) => r.data),
  audit: (fileName: string, content: string, projectId?: string) =>
    api.post('/aiops/audit', { file_name: fileName, content, project_id: projectId }).then((r) => r.data),
  analyzeLogs: (containerId: string, tail = '200') =>
    api.post('/aiops/logs', { container_id: containerId, tail }).then((r) => r.data),
}

export const authApi = {
  login: (email: string, password: string) =>
    api.post('/auth/login', { email, password }).then((r) => r.data),
  register: (email: string, password: string) =>
    api.post('/auth/register', { email, password }).then((r) => r.data),
  me: () => api.get('/auth/me').then((r) => r.data),
}

export const healthApi = {
  check: () => api.get('/health').then((r) => r.data),
}

export default api
