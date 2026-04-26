export interface Container {
  id: string
  name: string
  image: string
  status: string
  state: string
  ports: PortBinding[]
  labels: Record<string, string>
  created_at: string
  cpu_percent: number
  mem_usage_mb: number
  mem_limit_mb: number
  net_rx_mb: number
  net_tx_mb: number
}

export interface PortBinding {
  host_port: string
  container_port: string
  protocol: string
}

export interface Project {
  id: string
  name: string
  stack_type: string
  compose_content: string
  work_dir: string
  status: 'draft' | 'running' | 'stopped' | 'error'
  user_id: string
  created_at: string
  updated_at: string
  containers?: Container[]
  security_logs?: SecurityLog[]
}

export interface SecurityLog {
  id: string
  project_id: string
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info'
  finding: string
  suggestion: string
  ai_analysis: string
  file_path: string
  line_number: number
  resolved: boolean
  created_at: string
}

export interface ScalingRecommendation {
  action: 'scale_up' | 'scale_down' | 'ok'
  reason: string
  new_cpu_limit?: number
  new_mem_limit_mb?: number
  raw_analysis: string
}

export interface User {
  id: string
  email: string
  role: 'admin' | 'operator' | 'viewer'
  created_at: string
}

export interface SecurityFinding {
  severity: string
  finding: string
  suggestion: string
  line_number?: number
}

export interface HealthStatus {
  status: 'ok' | 'degraded'
  timestamp: string
  services: {
    docker: 'up' | 'down'
    ollama: 'up' | 'down'
  }
}
