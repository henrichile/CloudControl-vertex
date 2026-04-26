import { useState } from 'react'
import { authApi } from '@/services/api'

interface Props {
  onLogin: () => void
}

export default function Login({ onLogin }: Props) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isRegister, setIsRegister] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const data = isRegister
        ? await authApi.register(email, password)
        : await authApi.login(email, password)
      localStorage.setItem('cc_token', data.token)
      onLogin()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error desconocido'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-surface-900 flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="w-12 h-12 bg-brand-600 rounded-2xl flex items-center justify-center text-white font-bold text-xl mx-auto mb-4">CC</div>
          <h1 className="text-2xl font-bold text-white">Cloud Control</h1>
          <p className="text-slate-400 text-sm mt-1">{isRegister ? 'Crea tu cuenta' : 'Inicia sesión para continuar'}</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-surface-800 rounded-xl border border-surface-700 p-6 flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label className="text-xs text-slate-400">Email</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required
              className="bg-surface-900 border border-surface-600 rounded-lg px-3 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-brand-500"
              placeholder="admin@empresa.com" />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-slate-400">Contraseña</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8}
              className="bg-surface-900 border border-surface-600 rounded-lg px-3 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-brand-500"
              placeholder="••••••••" />
          </div>

          {error && <p className="text-red-400 text-xs bg-red-900/20 border border-red-800 rounded-lg px-3 py-2">{error}</p>}

          <button type="submit" disabled={loading}
            className="w-full py-2.5 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition-colors mt-1">
            {loading ? 'Cargando...' : isRegister ? 'Crear cuenta' : 'Iniciar sesión'}
          </button>

          <button type="button" onClick={() => setIsRegister(!isRegister)}
            className="text-xs text-slate-500 hover:text-slate-300 text-center transition-colors">
            {isRegister ? '¿Ya tienes cuenta? Inicia sesión' : '¿Primera vez? Crea tu cuenta'}
          </button>
        </form>
      </div>
    </div>
  )
}
