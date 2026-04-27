import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { projectsApi } from '@/services/api'
import { Project } from '@/types'
import { useState, useRef, useEffect } from 'react'
import { Plus, Play, Square, Trash2, ChevronRight, Loader2, Terminal, X } from 'lucide-react'
import clsx from 'clsx'

const STATUS_COLORS: Record<string, string> = {
  running: 'bg-emerald-900/50 text-emerald-300 border-emerald-800',
  stopped: 'bg-slate-700 text-slate-400 border-slate-600',
  error:   'bg-red-900/50 text-red-300 border-red-800',
  draft:   'bg-blue-900/50 text-blue-300 border-blue-800',
}

type LogEntry = { type: string; msg: string }

export default function Projects() {
  const qc = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({
    name: '', stack: '', db_name: '', db_user: '', db_password: 'secret', app_port: '8080', domain: '',
  })
  const [logPanel, setLogPanel] = useState<{ projectId: string; projectName: string } | null>(null)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [streaming, setStreaming] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  const logsEndRef = useRef<HTMLDivElement | null>(null)

  const { data: projectsData, isLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectsApi.list(),
  })

  const { data: stacksData } = useQuery({
    queryKey: ['stacks'],
    queryFn: () => projectsApi.listStacks(),
  })

  const projects: Project[] = projectsData?.projects ?? []
  const stacks: { name: string; description: string }[] = stacksData?.stacks ?? []

  const createMutation = useMutation({
    mutationFn: projectsApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['projects'] }); setShowCreate(false) },
  })

  const downMutation = useMutation({
    mutationFn: projectsApi.down,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['projects'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: projectsApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['projects'] }),
  })

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  function startStream(projectId: string, projectName: string) {
    setLogs([])
    setStreaming(true)
    setLogPanel({ projectId, projectName })
    const ctrl = projectsApi.upStream(projectId, (type, msg) => {
      setLogs((prev) => [...prev, { type, msg }])
      if (type === 'done' || type === 'error') {
        setStreaming(false)
        qc.invalidateQueries({ queryKey: ['projects'] })
      }
    })
    abortRef.current = ctrl
  }

  function closeLogPanel() {
    abortRef.current?.abort()
    setLogPanel(null)
    setStreaming(false)
  }

  return (
    <div className="p-8 flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Proyectos</h1>
          <p className="text-slate-400 text-sm mt-1">{projects.length} proyectos configurados</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 px-4 py-2 bg-brand-600 hover:bg-brand-700 text-white rounded-lg text-sm font-medium transition-colors"
        >
          <Plus size={15} />Nuevo Proyecto
        </button>
      </div>

      {isLoading ? (
        <div className="text-slate-500 text-sm">Cargando proyectos...</div>
      ) : projects.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-48 text-slate-500 border-2 border-dashed border-surface-700 rounded-xl gap-3">
          <p>No hay proyectos creados</p>
          <button onClick={() => setShowCreate(true)} className="text-brand-400 text-sm hover:text-brand-300">
            + Crear el primero
          </button>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {projects.map((p) => (
            <div key={p.id} className="bg-surface-800 border border-surface-700 rounded-xl p-5 flex items-center gap-4 hover:border-surface-600 transition-colors">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <h3 className="font-semibold text-white">{p.name}</h3>
                  <span className={clsx('text-xs px-2 py-0.5 rounded-full border', STATUS_COLORS[p.status] ?? STATUS_COLORS.draft)}>
                    {p.status}
                  </span>
                  <span className="text-xs bg-surface-700 text-slate-400 px-2 py-0.5 rounded font-mono">{p.stack_type}</span>
                </div>
                <p className="text-xs text-slate-500 mt-1 font-mono">{p.work_dir}</p>
              </div>
              <div className="flex items-center gap-2">
                {p.status !== 'running' ? (
                  <button
                    onClick={() => startStream(p.id, p.name)}
                    className="flex items-center gap-1 px-3 py-1.5 bg-emerald-800/50 hover:bg-emerald-700/50 text-emerald-300 rounded-lg text-xs transition-colors"
                  >
                    <Play size={13} />Up
                  </button>
                ) : (
                  <button onClick={() => downMutation.mutate(p.id)} disabled={downMutation.isPending}
                    className="flex items-center gap-1 px-3 py-1.5 bg-yellow-800/50 hover:bg-yellow-700/50 text-yellow-300 rounded-lg text-xs transition-colors">
                    <Square size={13} />Down
                  </button>
                )}
                <button onClick={() => { if (confirm(`¿Eliminar ${p.name}?`)) deleteMutation.mutate(p.id) }}
                  className="p-1.5 text-slate-500 hover:text-red-400 rounded-lg hover:bg-red-900/20 transition-colors">
                  <Trash2 size={14} />
                </button>
                <ChevronRight size={16} className="text-slate-600" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create Project Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4" onClick={() => setShowCreate(false)}>
          <div className="bg-surface-800 rounded-xl border border-surface-700 p-6 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-semibold text-white mb-4">Nuevo Proyecto</h3>
            <div className="flex flex-col gap-3">
              <Input label="Nombre" value={form.name} onChange={(v) => setForm({ ...form, name: v })} placeholder="mi-proyecto" />
              <div className="flex flex-col gap-1">
                <label className="text-xs text-slate-400">Stack</label>
                <select value={form.stack} onChange={(e) => setForm({ ...form, stack: e.target.value })}
                  className="bg-surface-900 border border-surface-600 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-brand-500">
                  <option value="">Selecciona un stack</option>
                  {stacks.map((s) => <option key={s.name} value={s.name}>{s.name} — {s.description}</option>)}
                </select>
              </div>
              <Input label="Base de datos" value={form.db_name} onChange={(v) => setForm({ ...form, db_name: v })} placeholder="appdb" />
              <Input label="Puerto" value={form.app_port} onChange={(v) => setForm({ ...form, app_port: v })} placeholder="8080" />
              <Input label="Dominio (opcional)" value={form.domain} onChange={(v) => setForm({ ...form, domain: v })} placeholder="app.midominio.com" />
            </div>
            <div className="flex gap-3 mt-5">
              <button onClick={() => setShowCreate(false)} className="flex-1 px-4 py-2 bg-surface-700 hover:bg-surface-600 text-slate-300 rounded-lg text-sm">
                Cancelar
              </button>
              <button
                onClick={() => createMutation.mutate({
                  name: form.name, stack: form.stack, db_name: form.db_name,
                  db_user: form.db_user, db_password: form.db_password,
                  app_port: form.app_port, domain: form.domain || undefined,
                })}
                disabled={!form.name || !form.stack || createMutation.isPending}
                className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium"
              >
                {createMutation.isPending ? <><Loader2 size={14} className="animate-spin" />Creando...</> : 'Crear'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Log Panel */}
      {logPanel && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="bg-gray-950 rounded-xl border border-surface-700 w-full max-w-3xl flex flex-col" style={{ height: '70vh' }}>
            <div className="flex items-center justify-between px-4 py-3 border-b border-surface-700 shrink-0">
              <div className="flex items-center gap-2 text-sm text-slate-300">
                <Terminal size={15} className="text-emerald-400" />
                <span className="font-mono font-semibold">{logPanel.projectName}</span>
                {streaming
                  ? <Loader2 size={13} className="animate-spin text-emerald-400 ml-1" />
                  : <span className="text-xs text-slate-500 ml-1">(terminado)</span>
                }
              </div>
              <button onClick={closeLogPanel} className="p-1 text-slate-500 hover:text-slate-300 rounded transition-colors">
                <X size={15} />
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-4 font-mono text-xs leading-relaxed">
              {logs.map((l, i) => (
                <div
                  key={i}
                  className={clsx('whitespace-pre-wrap break-all',
                    l.type === 'error' ? 'text-red-400' :
                    l.type === 'done'  ? 'text-emerald-400 font-semibold' :
                    'text-slate-300'
                  )}
                >
                  {l.type === 'error' && '✗ '}{l.type === 'done' && '✓ '}{l.msg}
                </div>
              ))}
              <div ref={logsEndRef} />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function Input({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-xs text-slate-400">{label}</label>
      <input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className="bg-surface-900 border border-surface-600 rounded-lg px-3 py-2 text-sm text-slate-300 placeholder-slate-600 focus:outline-none focus:border-brand-500" />
    </div>
  )
}
