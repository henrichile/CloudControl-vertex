import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import Sidebar from '@/components/Sidebar'
import Dashboard from '@/pages/Dashboard'
import Containers from '@/pages/Containers'
import Projects from '@/pages/Projects'
import Security from '@/pages/Security'
import AIOps from '@/pages/AIOps'
import Login from '@/pages/Login'

const qc = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      retry: 1,
    },
  },
})

function Layout() {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/containers" element={<Containers />} />
          <Route path="/projects" element={<Projects />} />
          <Route path="/security" element={<Security />} />
          <Route path="/aiops" element={<AIOps />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  )
}

export default function App() {
  const [authed, setAuthed] = useState(() => !!localStorage.getItem('cc_token'))

  return (
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        {authed ? (
          <Layout />
        ) : (
          <Routes>
            <Route path="*" element={<Login onLogin={() => setAuthed(true)} />} />
          </Routes>
        )}
      </BrowserRouter>
    </QueryClientProvider>
  )
}
