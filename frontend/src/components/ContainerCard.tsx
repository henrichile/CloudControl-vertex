import { Container } from '@/types'
import { Play, Square, Trash2, Activity, Cpu, MemoryStick } from 'lucide-react'
import clsx from 'clsx'

interface Props {
  container: Container
  onStart: (id: string) => void
  onStop: (id: string) => void
  onRemove: (id: string) => void
  onAnalyze: (id: string) => void
}

const STATE_COLORS: Record<string, string> = {
  running: 'bg-emerald-500',
  exited: 'bg-slate-500',
  paused: 'bg-yellow-500',
  restarting: 'bg-blue-500',
  dead: 'bg-red-500',
}

export default function ContainerCard({ container, onStart, onStop, onRemove, onAnalyze }: Props) {
  const isRunning = container.state === 'running'
  const dotColor = STATE_COLORS[container.state] ?? 'bg-slate-500'
  const memPercent = container.mem_limit_mb > 0
    ? Math.round((container.mem_usage_mb / container.mem_limit_mb) * 100)
    : 0

  return (
    <div className="bg-surface-800 rounded-xl border border-surface-700 p-5 flex flex-col gap-4 hover:border-brand-500 transition-colors">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex flex-col gap-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={clsx('w-2 h-2 rounded-full flex-shrink-0', dotColor)} />
            <h3 className="font-semibold text-white truncate">{container.name}</h3>
          </div>
          <p className="text-xs text-slate-400 font-mono truncate">{container.image}</p>
        </div>
        <span className={clsx(
          'text-xs px-2 py-0.5 rounded-full flex-shrink-0',
          isRunning ? 'bg-emerald-900/50 text-emerald-300' : 'bg-slate-700 text-slate-400'
        )}>
          {container.state}
        </span>
      </div>

      {/* Metrics (only when running) */}
      {isRunning && (
        <div className="grid grid-cols-2 gap-3">
          <MetricBar icon={<Cpu size={12} />} label="CPU" value={container.cpu_percent} unit="%" max={100} />
          <MetricBar icon={<MemoryStick size={12} />} label="RAM" value={memPercent} unit="%" max={100} />
        </div>
      )}

      {/* Ports */}
      {container.ports?.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {container.ports.map((p, i) => (
            <span key={i} className="text-xs bg-surface-700 text-slate-300 px-2 py-0.5 rounded font-mono">
              {p.host_port}:{p.container_port}
            </span>
          ))}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 pt-1 border-t border-surface-700">
        {isRunning ? (
          <button onClick={() => onStop(container.id)} className="btn-icon text-yellow-400 hover:text-yellow-300" title="Detener">
            <Square size={15} />
          </button>
        ) : (
          <button onClick={() => onStart(container.id)} className="btn-icon text-emerald-400 hover:text-emerald-300" title="Iniciar">
            <Play size={15} />
          </button>
        )}
        <button onClick={() => onRemove(container.id)} className="btn-icon text-red-400 hover:text-red-300" title="Eliminar">
          <Trash2 size={15} />
        </button>
        {isRunning && (
          <button onClick={() => onAnalyze(container.id)} className="btn-icon text-blue-400 hover:text-blue-300 ml-auto" title="Analizar con IA">
            <Activity size={15} />
            <span className="text-xs">IA</span>
          </button>
        )}
      </div>
    </div>
  )
}

interface MetricBarProps {
  icon: React.ReactNode
  label: string
  value: number
  unit: string
  max: number
}

function MetricBar({ icon, label, value, unit, max }: MetricBarProps) {
  const pct = Math.min((value / max) * 100, 100)
  const color = pct > 80 ? 'bg-red-500' : pct > 60 ? 'bg-yellow-500' : 'bg-brand-500'

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span className="flex items-center gap-1">{icon}{label}</span>
        <span className="font-mono text-slate-200">{value.toFixed(1)}{unit}</span>
      </div>
      <div className="h-1.5 bg-surface-700 rounded-full overflow-hidden">
        <div className={clsx('h-full rounded-full transition-all', color)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}
