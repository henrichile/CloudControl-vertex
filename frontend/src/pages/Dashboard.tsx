import { useQuery } from '@tanstack/react-query'
import { containersApi, healthApi } from '@/services/api'
import { Container, HealthStatus } from '@/types'
import { Box, CheckCircle2, XCircle, Cpu } from 'lucide-react'
import clsx from 'clsx'
import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip } from 'recharts'

export default function Dashboard() {
  const { data: containersData } = useQuery({
    queryKey: ['containers'],
    queryFn: () => containersApi.list(),
    refetchInterval: 10_000,
  })

  const { data: health } = useQuery<HealthStatus>({
    queryKey: ['health'],
    queryFn: () => healthApi.check(),
    refetchInterval: 15_000,
  })

  const containers: Container[] = containersData?.containers ?? []
  const running = containers.filter((c) => c.state === 'running').length
  const stopped = containers.length - running

  return (
    <div className="p-8 flex flex-col gap-8">
      <div>
        <h1 className="text-2xl font-bold text-white">Dashboard</h1>
        <p className="text-slate-400 text-sm mt-1">Resumen del entorno de contenedores</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard title="Total Contenedores" value={containers.length} icon={<Box size={20} />} color="text-brand-400" />
        <StatCard title="En Ejecución" value={running} icon={<CheckCircle2 size={20} />} color="text-emerald-400" />
        <StatCard title="Detenidos" value={stopped} icon={<XCircle size={20} />} color="text-slate-400" />
        <StatCard title="Uso CPU Prom." value={avgCPU(containers)} suffix="%" icon={<Cpu size={20} />} color="text-yellow-400" />
      </div>

      {/* Services health */}
      {health && (
        <div className="bg-surface-800 rounded-xl border border-surface-700 p-5">
          <h2 className="text-sm font-semibold text-slate-300 mb-4">Estado de Servicios</h2>
          <div className="flex gap-6">
            <ServiceStatus name="Docker Engine" status={health.services.docker} />
            <ServiceStatus name="Ollama AI" status={health.services.ollama} />
          </div>
        </div>
      )}

      {/* CPU chart for running containers */}
      {running > 0 && (
        <div className="bg-surface-800 rounded-xl border border-surface-700 p-5">
          <h2 className="text-sm font-semibold text-slate-300 mb-4">CPU por Contenedor</h2>
          <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={containers.filter(c => c.state === 'running').map(c => ({ name: c.name, cpu: c.cpu_percent }))}>
                <XAxis dataKey="name" tick={{ fontSize: 11, fill: '#94a3b8' }} />
                <YAxis tick={{ fontSize: 11, fill: '#94a3b8' }} unit="%" />
                <Tooltip
                  contentStyle={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 8 }}
                  labelStyle={{ color: '#e2e8f0' }}
                />
                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fill="#3b82f620" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Recent containers */}
      <div className="bg-surface-800 rounded-xl border border-surface-700 p-5">
        <h2 className="text-sm font-semibold text-slate-300 mb-4">Contenedores Recientes</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-xs text-slate-500 border-b border-surface-700">
              <th className="pb-2">Nombre</th>
              <th className="pb-2">Imagen</th>
              <th className="pb-2">Estado</th>
              <th className="pb-2">CPU</th>
              <th className="pb-2">RAM</th>
            </tr>
          </thead>
          <tbody>
            {containers.slice(0, 8).map((c) => (
              <tr key={c.id} className="border-b border-surface-700/50 hover:bg-surface-700/30">
                <td className="py-2 font-mono text-xs text-slate-200">{c.name}</td>
                <td className="py-2 text-xs text-slate-400 truncate max-w-[160px]">{c.image}</td>
                <td className="py-2">
                  <span className={clsx('text-xs px-2 py-0.5 rounded-full', c.state === 'running' ? 'bg-emerald-900/50 text-emerald-300' : 'bg-slate-700 text-slate-400')}>
                    {c.state}
                  </span>
                </td>
                <td className="py-2 font-mono text-xs text-slate-300">{c.cpu_percent?.toFixed(1)}%</td>
                <td className="py-2 font-mono text-xs text-slate-300">{c.mem_usage_mb?.toFixed(0)} MB</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function StatCard({ title, value, suffix = '', icon, color }: {
  title: string; value: number; suffix?: string; icon: React.ReactNode; color: string
}) {
  return (
    <div className="bg-surface-800 rounded-xl border border-surface-700 p-5">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs text-slate-400">{title}</span>
        <span className={clsx('opacity-70', color)}>{icon}</span>
      </div>
      <p className={clsx('text-3xl font-bold', color)}>{value}{suffix}</p>
    </div>
  )
}

function ServiceStatus({ name, status }: { name: string; status: 'up' | 'down' }) {
  return (
    <div className="flex items-center gap-2">
      <span className={clsx('w-2 h-2 rounded-full', status === 'up' ? 'bg-emerald-500' : 'bg-red-500')} />
      <span className="text-sm text-slate-300">{name}</span>
      <span className={clsx('text-xs', status === 'up' ? 'text-emerald-400' : 'text-red-400')}>{status}</span>
    </div>
  )
}

function avgCPU(containers: Container[]) {
  const running = containers.filter((c) => c.state === 'running')
  if (!running.length) return 0
  return Math.round(running.reduce((s, c) => s + c.cpu_percent, 0) / running.length * 10) / 10
}
