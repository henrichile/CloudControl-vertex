import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { aiopsApi } from '@/services/api'
import SecurityAlert from '@/components/SecurityAlert'
import { Shield, Upload, Loader2 } from 'lucide-react'
import { SecurityFinding } from '@/types'
import clsx from 'clsx'

export default function Security() {
  const [fileName, setFileName] = useState('')
  const [content, setContent] = useState('')
  const [result, setResult] = useState<{ findings: SecurityFinding[]; score: number; raw_analysis: string } | null>(null)

  const auditMutation = useMutation({
    mutationFn: ({ fileName, content }: { fileName: string; content: string }) =>
      aiopsApi.audit(fileName, content),
    onSuccess: (data) => setResult(data.result),
  })

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setFileName(file.name)
    const reader = new FileReader()
    reader.onload = (ev) => setContent(ev.target?.result as string)
    reader.readAsText(file)
  }

  const scoreColor = result
    ? result.score >= 80 ? 'text-emerald-400' : result.score >= 50 ? 'text-yellow-400' : 'text-red-400'
    : ''

  return (
    <div className="p-8 flex flex-col gap-6 max-w-4xl">
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Shield size={24} className="text-brand-400" />
          Auditoría de Seguridad
        </h1>
        <p className="text-slate-400 text-sm mt-1">
          Analiza archivos de configuración con IA para detectar vulnerabilidades
        </p>
      </div>

      {/* Upload form */}
      <div className="bg-surface-800 rounded-xl border border-surface-700 p-6 flex flex-col gap-4">
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 px-4 py-2 bg-surface-700 hover:bg-surface-600 text-slate-200 rounded-lg text-sm cursor-pointer transition-colors border border-surface-600">
            <Upload size={15} />
            Seleccionar archivo
            <input type="file" className="hidden" accept=".yml,.yaml,.env,.conf,.json,.toml,.nginx" onChange={handleFileChange} />
          </label>
          {fileName && <span className="text-sm text-slate-400 font-mono">{fileName}</span>}
        </div>

        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="O pega el contenido del archivo aquí (docker-compose.yml, .env, nginx.conf...)"
          className="w-full h-48 bg-surface-900 border border-surface-600 rounded-lg p-3 text-sm font-mono text-slate-300 placeholder-slate-600 resize-none focus:outline-none focus:border-brand-500"
        />

        {content && !fileName && (
          <input
            value={fileName}
            onChange={(e) => setFileName(e.target.value)}
            placeholder="Nombre del archivo (ej: docker-compose.yml)"
            className="bg-surface-900 border border-surface-600 rounded-lg px-3 py-2 text-sm text-slate-300 placeholder-slate-600 focus:outline-none focus:border-brand-500"
          />
        )}

        <button
          onClick={() => auditMutation.mutate({ fileName: fileName || 'config', content })}
          disabled={!content || auditMutation.isPending}
          className="flex items-center justify-center gap-2 px-6 py-2.5 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors"
        >
          {auditMutation.isPending ? (
            <><Loader2 size={15} className="animate-spin" />Analizando con IA...</>
          ) : (
            <><Shield size={15} />Auditar con IA</>
          )}
        </button>
      </div>

      {/* Results */}
      {result && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-white">Resultados</h2>
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-400">Puntuación de seguridad:</span>
              <span className={clsx('text-2xl font-bold', scoreColor)}>{result.score}/100</span>
            </div>
          </div>

          <div className="flex flex-col gap-3">
            {result.findings.length === 0 ? (
              <div className="flex items-center gap-2 text-emerald-400 bg-emerald-900/20 border border-emerald-800 rounded-lg p-4">
                <Shield size={18} />
                No se encontraron problemas de seguridad.
              </div>
            ) : (
              result.findings.map((f, i) => (
                <SecurityAlert
                  key={i}
                  log={{
                    id: String(i),
                    project_id: '',
                    severity: f.severity as 'critical' | 'high' | 'medium' | 'low' | 'info',
                    finding: f.finding,
                    suggestion: f.suggestion,
                    ai_analysis: result.raw_analysis,
                    file_path: fileName,
                    line_number: f.line_number ?? 0,
                    resolved: false,
                    created_at: new Date().toISOString(),
                  }}
                />
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
