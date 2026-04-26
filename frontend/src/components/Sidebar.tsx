import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Box, FolderOpen, Shield, Activity, LogOut } from 'lucide-react'
import clsx from 'clsx'

const NAV = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/containers', label: 'Contenedores', icon: Box },
  { to: '/projects', label: 'Proyectos', icon: FolderOpen },
  { to: '/security', label: 'Seguridad', icon: Shield },
  { to: '/aiops', label: 'AIOps', icon: Activity },
]

export default function Sidebar() {
  const handleLogout = () => {
    localStorage.removeItem('cc_token')
    window.location.href = '/login'
  }

  return (
    <aside className="w-60 flex-shrink-0 bg-surface-800 border-r border-surface-700 flex flex-col">
      {/* Logo */}
      <div className="px-6 py-5 border-b border-surface-700">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 bg-brand-600 rounded-lg flex items-center justify-center text-white font-bold text-sm">CC</div>
          <span className="font-bold text-white">Cloud Control</span>
        </div>
        <p className="text-xs text-slate-500 mt-1">Container Orchestration</p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 flex flex-col gap-1">
        {NAV.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) => clsx(
              'flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
              isActive
                ? 'bg-brand-600/20 text-brand-400 font-medium'
                : 'text-slate-400 hover:text-slate-200 hover:bg-surface-700'
            )}
          >
            <Icon size={16} />
            {label}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="px-3 py-4 border-t border-surface-700">
        <button
          onClick={handleLogout}
          className="flex items-center gap-3 px-3 py-2 w-full rounded-lg text-sm text-slate-400 hover:text-red-400 hover:bg-surface-700 transition-colors"
        >
          <LogOut size={16} />
          Cerrar sesión
        </button>
      </div>
    </aside>
  )
}
