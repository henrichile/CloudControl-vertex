import { SecurityLog } from '@/types'
import { AlertTriangle, AlertCircle, Info, CheckCircle2, XCircle } from 'lucide-react'
import clsx from 'clsx'

const SEVERITY_CONFIG = {
  critical: { icon: XCircle, color: 'text-red-400', bg: 'bg-red-900/20 border-red-800', label: 'Crítico' },
  high:     { icon: AlertCircle, color: 'text-orange-400', bg: 'bg-orange-900/20 border-orange-800', label: 'Alto' },
  medium:   { icon: AlertTriangle, color: 'text-yellow-400', bg: 'bg-yellow-900/20 border-yellow-800', label: 'Medio' },
  low:      { icon: Info, color: 'text-blue-400', bg: 'bg-blue-900/20 border-blue-800', label: 'Bajo' },
  info:     { icon: Info, color: 'text-slate-400', bg: 'bg-slate-800 border-slate-700', label: 'Info' },
}

interface Props {
  log: SecurityLog
  onResolve?: (id: string) => void
}

export default function SecurityAlert({ log, onResolve }: Props) {
  const cfg = SEVERITY_CONFIG[log.severity] ?? SEVERITY_CONFIG.info
  const Icon = cfg.icon

  return (
    <div className={clsx('rounded-lg border p-4 flex gap-3', cfg.bg, log.resolved && 'opacity-50')}>
      <Icon size={18} className={clsx('flex-shrink-0 mt-0.5', cfg.color)} />
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <div>
            <span className={clsx('text-xs font-semibold uppercase tracking-wide', cfg.color)}>{cfg.label}</span>
            {log.file_path && (
              <span className="ml-2 text-xs text-slate-500 font-mono">{log.file_path}{log.line_number ? `:${log.line_number}` : ''}</span>
            )}
          </div>
          {log.resolved ? (
            <span className="flex items-center gap-1 text-xs text-emerald-400"><CheckCircle2 size={12} />Resuelto</span>
          ) : onResolve && (
            <button onClick={() => onResolve(log.id)} className="text-xs text-slate-500 hover:text-slate-300 transition-colors">
              Marcar resuelto
            </button>
          )}
        </div>
        <p className="mt-1 text-sm text-slate-200">{log.finding}</p>
        {log.suggestion && (
          <p className="mt-2 text-xs text-slate-400 border-l-2 border-slate-600 pl-2">{log.suggestion}</p>
        )}
      </div>
    </div>
  )
}
