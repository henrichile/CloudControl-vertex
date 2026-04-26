import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { containersApi, aiopsApi } from '@/services/api'
import { Container } from '@/types'
import ContainerCard from '@/components/ContainerCard'
import { RefreshCw } from 'lucide-react'
import { useState } from 'react'

export default function Containers() {
  const qc = useQueryClient()
  const [filter, setFilter] = useState<'all' | 'running' | 'stopped'>('all')
  const [analysisResult, setAnalysisResult] = useState<{ container: string; result: unknown } | null>(null)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['containers'],
    queryFn: () => containersApi.list(),
    refetchInterval: 15_000,
  })

  const containers: Container[] = data?.containers ?? []
  const filtered = containers.filter((c) => {
    if (filter === 'running') return c.state === 'running'
    if (filter === 'stopped') return c.state !== 'running'
    return true
  })

  const startMutation = useMutation({
    mutationFn: containersApi.start,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['containers'] }),
  })

  const stopMutation = useMutation({
    mutationFn: (id: string) => containersApi.stop(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['containers'] }),
  })

  const removeMutation = useMutation({
    mutationFn: (id: string) => containersApi.remove(id, true),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['containers'] }),
  })

  const analyzeMutation = useMutation({
    mutationFn: aiopsApi.analyze,
    onSuccess: (data, containerId) => setAnalysisResult({ container: containerId, result: data }),
  })

  return (
    <div className="p-8 flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Contenedores</h1>
          <p className="text-slate-400 text-sm mt-1">{containers.length} contenedores encontrados</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex bg-surface-800 rounded-lg border border-surface-700 overflow-hidden">
            {(['all', 'running', 'stopped'] as const).map((f) => (
              <button
                key={f}
                onClick={() => setFilter(f)}
                className={`px-3 py-1.5 text-xs capitalize transition-colors ${filter === f ? 'bg-brand-600 text-white' : 'text-slate-400 hover:text-slate-200'}`}
              >
                {f === 'all' ? 'Todos' : f === 'running' ? 'En ejecución' : 'Detenidos'}
              </button>
            ))}
          </div>
          <button onClick={() => refetch()} className="p-2 rounded-lg bg-surface-800 border border-surface-700 text-slate-400 hover:text-slate-200 transition-colors">
            <RefreshCw size={15} />
          </button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-48 text-slate-500">Cargando contenedores...</div>
      ) : filtered.length === 0 ? (
        <div className="flex items-center justify-center h-48 text-slate-500 border-2 border-dashed border-surface-700 rounded-xl">
          No hay contenedores {filter !== 'all' ? filter === 'running' ? 'en ejecución' : 'detenidos' : ''}
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filtered.map((c) => (
            <ContainerCard
              key={c.id}
              container={c}
              onStart={startMutation.mutate}
              onStop={stopMutation.mutate}
              onRemove={(id) => { if (confirm(`¿Eliminar ${c.name}?`)) removeMutation.mutate(id) }}
              onAnalyze={analyzeMutation.mutate}
            />
          ))}
        </div>
      )}

      {/* AI Analysis Modal */}
      {analysisResult && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4" onClick={() => setAnalysisResult(null)}>
          <div className="bg-surface-800 rounded-xl border border-surface-700 p-6 max-w-lg w-full" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-semibold text-white mb-2">Análisis IA — {analysisResult.container}</h3>
            <pre className="text-xs text-slate-300 bg-surface-900 p-4 rounded-lg overflow-auto max-h-80 font-mono whitespace-pre-wrap">
              {JSON.stringify(analysisResult.result, null, 2)}
            </pre>
            <button onClick={() => setAnalysisResult(null)} className="mt-4 px-4 py-2 bg-surface-700 hover:bg-surface-600 text-slate-200 rounded-lg text-sm transition-colors w-full">
              Cerrar
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
