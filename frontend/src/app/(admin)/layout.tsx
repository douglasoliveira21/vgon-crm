'use client'

import { useEffect, useState } from 'react'
import { useRouter, usePathname } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import Link from 'next/link'
import {
  LayoutDashboard,
  Building2,
  Activity,
  ScrollText,
  ArrowLeft,
  Shield,
  LogOut,
  Menu,
  X,
} from 'lucide-react'

const adminNavItems = [
  { href: '/admin', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/admin/tenants', label: 'Empresas (Tenants)', icon: Building2 },
  { href: '/admin/operations', label: 'Operações', icon: Activity },
  { href: '/admin/audit', label: 'Auditoria Global', icon: ScrollText },
]

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { user, isAuthenticated, isLoading, checkAuth, logout } = useAuthStore()
  const [mounted, setMounted] = useState(false)
  const [authRequested, setAuthRequested] = useState(false)
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

  useEffect(() => {
    setMounted(true)
    setAuthRequested(true)
    checkAuth()
  }, [checkAuth])

  useEffect(() => {
    if (mounted && authRequested && !isLoading && !isAuthenticated) {
      router.replace('/login')
      return
    }
    if (mounted && user && !user.is_super_admin) {
      router.replace('/dashboard')
    }
  }, [authRequested, isAuthenticated, isLoading, mounted, user, router])

  useEffect(() => {
    setMobileNavOpen(false)
  }, [pathname])

  if (!mounted || isLoading || !isAuthenticated || !user?.is_super_admin) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-900">
      <header className="fixed inset-x-0 top-0 z-40 flex h-14 items-center border-b border-gray-800 bg-gray-950 px-4 md:hidden">
        <button onClick={() => setMobileNavOpen(true)} className="rounded-lg p-2 text-gray-300 hover:bg-gray-800" aria-label="Abrir menu">
          <Menu className="h-5 w-5" />
        </button>
        <span className="ml-3 font-semibold text-white">Super Admin</span>
      </header>
      {mobileNavOpen && <button className="fixed inset-0 z-40 bg-black/60 md:hidden" onClick={() => setMobileNavOpen(false)} aria-label="Fechar menu" />}
      {/* Sidebar */}
      <aside className={`fixed left-0 top-0 z-50 flex h-full w-64 flex-col border-r border-gray-800 bg-gray-950 transition-transform md:translate-x-0 ${mobileNavOpen ? 'translate-x-0' : '-translate-x-full'}`}>
        {/* Header */}
        <div className="p-6 border-b border-gray-800">
          <button onClick={() => setMobileNavOpen(false)} className="absolute right-3 top-3 rounded-lg p-2 text-gray-400 hover:bg-gray-800 md:hidden" aria-label="Fechar menu"><X className="h-5 w-5" /></button>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-indigo-600 flex items-center justify-center">
              <Shield className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-white font-bold text-lg">Super Admin</h1>
              <p className="text-gray-400 text-xs">Painel de Gestão</p>
            </div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4 space-y-1">
          {adminNavItems.map((item) => {
            const isActive = pathname === item.href
            const Icon = item.icon
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                  isActive
                    ? 'bg-indigo-600 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`}
              >
                <Icon className="w-5 h-5" />
                <span className="text-sm font-medium">{item.label}</span>
              </Link>
            )
          })}
        </nav>

        {/* Footer */}
        <div className="p-4 border-t border-gray-800 space-y-2">
          <Link
            href="/dashboard"
            className="flex items-center gap-3 px-4 py-3 rounded-lg text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
            <span className="text-sm font-medium">Voltar ao App</span>
          </Link>
          <button
            onClick={logout}
            className="w-full flex items-center gap-3 px-4 py-3 rounded-lg text-gray-400 hover:text-red-400 hover:bg-gray-800 transition-colors"
          >
            <LogOut className="w-5 h-5" />
            <span className="text-sm font-medium">Sair</span>
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="min-h-screen min-w-0 px-4 pb-6 pt-20 sm:px-6 md:ml-64 md:p-8">
        {children}
      </main>
    </div>
  )
}
