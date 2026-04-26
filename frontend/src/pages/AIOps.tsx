import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { containersApi, aiopsApi } from '@/services/api'
import { Container, ScalingRecommendation } from '@/types'
import {
  Activity,
  TrendingUp,
  TrendingDown,
  Minus,
  Cpu,
  MemoryStick,
  FileText,
  Loader2,
  ChevronDown,
  ChevronUp,
  AlertTriangle,
} from 'lucide-react'
import clsx from 'clsx'

type Tab = 'metrics' | 'logs'

interface AnalysisResult {
  container: string
  metrics: {
    cpu_percent: number
    mem_usage_mb: number
    mem_limit_mb: number
    net_rx_mb: number
    net_tx_mb: number
  }
  recommendation: ScalingRecommendation
}

interface LogResult {
  container: string
  analysis: string
}

const ACTION_CONFIG = {
  scale_up: {
    icon: TrendingUp,
    color: 'text-red-400',
    bg: 'bg-red-900/20 border-red-800',
    label: 'Escalar hacia arriba',
  },
  scale_down: {
    icon: TrendingDown,
    color: 'text-emerald-400',
    bg: 'bg-emerald-900/20 border-emerald-800',
    label: 'Reducir recursos',
  },
  ok: {
    icon: Minus,
    color: 'text-slate-400',
    bg: 'bg-slate-800 border-slate-700',
    label: 'Sin cambios necesarios',
  },
}

export default function AIOps() {
  const [tab, setTab] = useState<Tab>('metrics')
  const [selectedContainer, setSelectedContainer] = useState('')
  const [logTail, setLogTail] = useState('200')
  const [metricsResult, setMetricsResult] = useState<AnalysisResult | null>(null)
  const [logResult, setLogResult] = useState<LogResult | null>(null)
  const [expandedRaw, setExpandedRaw] = useState(false)

  const { data: containersData } = useQuery({
    queryKey: ['containers'],
    queryFn: () => containersApi.list(),
  })
  const runningContainers: Container[] = (containersData?.containers ?? []).filter(
    (c: Container) => c.state === 'running'
  )

  const analyzeMetricsMutation = useMutation({
    mutationFn: aiopsApi.analyze,
    onSuccess: (data) => setMetricsResult(data),
  })

  const analyzeLogsMutation = useMutation({
    mutationFn: ({ id, tail }: { id: string; tail: string }) =>
      aiopsApi.analyzeLogs(id, tail),
    onSuccess: (data) => setLogResult(data),
  })

  return (
    <div className="p-8 flex flex-col gap-6 max-w-5xl">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Activity size={24} className="text-brand-400" />
          AIOps — Orquestación Inteligente
        </h1>
        <p className="text-slate-400 text-sm mt-1">
          Análisis de métricas y logs de contenedores con IA local (Ollama)
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-surface-800 rounded-lg border border-surface-700 p-1 w-fit">
        <TabButton active={tab === 'metrics'} onClick={() => setTab('metrics')} icon={<Cpu size={14} />} label="Análisis de Métricas" />
        <TabButton active={tab === 'logs'} onClick={() => setTab('logs')} icon={<FileText size={14} />} label="Análisis de Logs" />
      </div>

      {/* Container selector */}
      <div className="bg-surface-800 rounded-xl border border-surface-700 p-5 flex flex-col gap-4">
        <h2 className="text-sm font-semibold text-slate-300">Seleccionar contenedor</h2>
        {runningContainers.length === 0 ? (
          <p className="text-sm text-slate-500 flex items-center gap-2">
            <AlertTriangle size={15} />
            No hay contenedores en ejecución
          </p>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
            {runningContainers.map((c) => (
              <button
                key={c.id}
                onClick={() => setSelectedContainer(c.name)}
                className={clsx(
                  'text-left px-3 py-2.5 rounded-lg border text-sm transition-colors',
                  selectedContainer === c.name
                    ? 'bg-brand-600/20 border-brand-600 text-brand-300'
                    : 'bg-surface-700 border-surface-600 text-slate-300 hover:border-surface-500'
                )}
              >
                <p className="font-mono text-xs truncate font-medium">{c.name}</p>
                <p className="text-xs text-slate-500 mt-0.5 truncate">{c.image}</p>
                <div className="flex items-center gap-2 mt-1.5">
                  <MetricPill icon={<Cpu size={10} />} value={`${c.cpu_percent?.toFixed(1)}%`} />
                  <MetricPill icon={<MemoryStick size={10} />} value={`${c.mem_usage_mb?.toFixed(0)}MB`} />
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Tab: Métricas */}
      {tab === 'metrics' && (
        <div className="flex flex-col gap-5">
          <div className="flex items-center gap-3">
            <button
              onClick={() => {
                if (!selectedContainer) return
                analyzeMetricsMutation.mutate(selectedContainer)
                setMetricsResult(null)
              }}
              disabled={!selectedContainer || analyzeMetricsMutation.isPending}
              className="flex items-center gap-2 px-5 py-2.5 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors"
            >
              {analyzeMetricsMutation.isPending ? (
                <><Loader2 size={15} className="animate-spin" />Analizando con IA...</>
              ) : (
                <><Activity size={15} />Analizar métricas</>
              )}
            </button>
            {!selectedContainer && (
              <span className="text-xs text-slate-500">Selecciona un contenedor primero</span>
            )}
          </div>

          {analyzeMetricsMutation.isError && (
            <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-4 py-3">
              Error al conectar con Ollama. ¿Está el servidor activo?
            </div>
          )}

          {metricsResult && <MetricsResultCard result={metricsResult} expandedRaw={expandedRaw} onToggleRaw={() => setExpandedRaw(!expandedRaw)} />}
        </div>
      )}

      {/* Tab: Logs */}
      {tab === 'logs' && (
        <div className="flex flex-col gap-5">
          <div className="flex items-center gap-3 flex-wrap">
            <div className="flex flex-col gap-1">
              <label className="text-xs text-slate-400">Líneas a analizar</label>
              <select
                value={logTail}
                onChange={(e) => setLogTail(e.target.value)}
                className="bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-brand-500"
              >
                {['50', '100', '200', '500', '1000'].map((n) => (
                  <option key={n} value={n}>Últimas {n} líneas</option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-slate-400 opacity-0">Acción</label>
              <button
                onClick={() => {
                  if (!selectedContainer) return
                  analyzeLogsMutation.mutate({ id: selectedContainer, tail: logTail })
                  setLogResult(null)
                }}
                disabled={!selectedContainer || analyzeLogsMutation.isPending}
                className="flex items-center gap-2 px-5 py-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors"
              >
                {analyzeLogsMutation.isPending ? (
                  <><Loader2 size={15} className="animate-spin" />Analizando logs...</>
                ) : (
                  <><FileText size={15} />Analizar logs</>
                )}
              </button>
            </div>
          </div>

          {analyzeLogsMutation.isError && (
            <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-4 py-3">
              Error al analizar los logs. Verifica que Ollama esté activo.
            </div>
          )}

          {logResult && (
            <div className="bg-surface-800 rounded-xl border border-surface-700 p-5 flex flex-col gap-3">
              <h3 className="text-sm font-semibold text-slate-300 flex items-center gap-2">
                <FileText size={15} className="text-brand-400" />
                Análisis de logs — {logResult.container}
              </h3>
              <div className="bg-surface-900 rounded-lg p-4 text-sm text-slate-300 whitespace-pre-wrap leading-relaxed border border-surface-700">
                {logResult.analysis}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Historial vacío */}
      {!metricsResult && !logResult && !analyzeMetricsMutation.isPending && !analyzeLogsMutation.isPending && (
        <div className="flex flex-col items-center justify-center h-32 text-slate-600 border-2 border-dashed border-surface-700 rounded-xl gap-2">
          <Activity size={24} />
          <p className="text-sm">Selecciona un contenedor y ejecuta un análisis</p>
        </div>
      )}
    </div>
  )
}

function MetricsResultCard({
  result,
  expandedRaw,
  onToggleRaw,
}: {
  result: AnalysisResult
  expandedRaw: boolean
  onToggleRaw: () => void
}) {
  const action = result.recommendation.action
  const cfg = ACTION_CONFIG[action] ?? ACTION_CONFIG.ok
  const Icon = cfg.icon

  return (
    <div className="flex flex-col gap-4">
      {/* Snapshot de métricas */}
      <div className="bg-surface-800 rounded-xl border border-surface-700 p-5">
        <h3 className="text-sm font-semibold text-slate-300 mb-4">
          Métricas actuales — {result.container}
        </h3>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricBox label="CPU" value={`${result.metrics.cpu_percent.toFixed(2)}%`} icon={<Cpu size={14} />} />
          <MetricBox label="RAM usada" value={`${result.metrics.mem_usage_mb.toFixed(0)} MB`} icon={<MemoryStick size={14} />} />
          <MetricBox label="RAM límite" value={result.metrics.mem_limit_mb > 0 ? `${result.metrics.mem_limit_mb.toFixed(0)} MB` : 'Sin límite'} icon={<MemoryStick size={14} />} />
          <MetricBox label="Red Rx" value={`${result.metrics.net_rx_mb.toFixed(2)} MB`} icon={<Activity size={14} />} />
        </div>
      </div>

      {/* Recomendación */}
      <div className={clsx('rounded-xl border p-5 flex gap-4', cfg.bg)}>
        <Icon size={24} className={clsx('flex-shrink-0 mt-0.5', cfg.color)} />
        <div className="flex flex-col gap-2 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className={clsx('font-semibold', cfg.color)}>{cfg.label}</span>
            {(result.recommendation.new_cpu_limit || result.recommendation.new_mem_limit_mb) && (
              <span className="text-xs text-slate-400">Límites aplicados automáticamente</span>
            )}
          </div>
          <p className="text-sm text-slate-300">{result.recommendation.reason}</p>

          {(result.recommendation.new_cpu_limit || result.recommendation.new_mem_limit_mb) && (
            <div className="flex gap-4 mt-1">
              {result.recommendation.new_cpu_limit && (
                <div className="text-xs">
                  <span className="text-slate-500">CPU nuevo límite:</span>{' '}
                  <span className="font-mono text-slate-200">{result.recommendation.new_cpu_limit}%</span>
                </div>
              )}
              {result.recommendation.new_mem_limit_mb && (
                <div className="text-xs">
                  <span className="text-slate-500">RAM nuevo límite:</span>{' '}
                  <span className="font-mono text-slate-200">{result.recommendation.new_mem_limit_mb} MB</span>
                </div>
              )}
            </div>
          )}

          <button
            onClick={onToggleRaw}
            className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 mt-1 w-fit transition-colors"
          >
            {expandedRaw ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
            {expandedRaw ? 'Ocultar' : 'Ver'} análisis completo
          </button>

          {expandedRaw && (
            <pre className="text-xs text-slate-400 bg-surface-900 rounded-lg p-3 whitespace-pre-wrap leading-relaxed mt-1 border border-surface-700 overflow-auto max-h-60 font-mono">
              {result.recommendation.raw_analysis}
            </pre>
          )}
        </div>
      </div>
    </div>
  )
}

function MetricBox({ label, value, icon }: { label: string; value: string; icon: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-slate-500 flex items-center gap-1">{icon}{label}</span>
      <span className="font-mono text-sm text-slate-200 font-medium">{value}</span>
    </div>
  )
}

function MetricPill({ icon, value }: { icon: React.ReactNode; value: string }) {
  return (
    <span className="flex items-center gap-0.5 text-slate-500 text-xs">
      {icon}{value}
    </span>
  )
}

function TabButton({ active, onClick, icon, label }: { active: boolean; onClick: () => void; icon: React.ReactNode; label: string }) {
  return (
    <button
      onClick={onClick}
      className={clsx(
        'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
        active ? 'bg-brand-600 text-white' : 'text-slate-400 hover:text-slate-200'
      )}
    >
      {icon}{label}
    </button>
  )
}
