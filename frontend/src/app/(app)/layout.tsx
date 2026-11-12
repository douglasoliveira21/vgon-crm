'use client'

import { useEffect, useRef, useState } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useAppearanceStore } from '@/store/appearance'
import Sidebar from '@/components/layout/sidebar'
import { Menu, Search, Bell, HelpCircle, Command } from 'lucide-react'
import { SafeImage } from '@/components/safe-image'

const agentDeniedPaths = ['/inbox', '/channels', '/automations', '/campaigns', '/metrics', '/audit-logs', '/privacy', '/widget']
const supervisorDeniedPaths = ['/inbox', '/channels', '/automations', '/campaigns', '/audit-logs', '/privacy', '/widget']

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, isLoading, checkAuth, user } = useAuthStore()
  const { sidebarPinned, sidebarHovered, mobileSidebarOpen, setMobileSidebarOpen, initAppearance } = useAppearanceStore()
  const [mounted, setMounted] = useState(false)
  const [authRequested, setAuthRequested] = useState(false)
	const [globalSearch, setGlobalSearch] = useState('')
	const globalSearchRef = useRef<HTMLInputElement>(null)
	const isAgentDeniedPath = user?.role_slug === 'agent' && agentDeniedPaths.some((path) => pathname === path || pathname.startsWith(path + '/'))
	const isSupervisorDeniedPath = user?.role_slug === 'supervisor' && supervisorDeniedPaths.some((path) => pathname === path || pathname.startsWith(path + '/'))

  useEffect(() => {
    setMounted(true)
    initAppearance()
    setAuthRequested(true)
    checkAuth()
  }, [initAppearance])

  useEffect(() => {
    if (mounted && authRequested && !isLoading && !isAuthenticated && user === null) {
      router.replace('/login')
    }
  }, [authRequested, isAuthenticated, isLoading, mounted, router, user])

  useEffect(() => {
	if (!mounted || user?.role_slug !== 'agent') return
	if (isAgentDeniedPath) router.replace('/dashboard')
  }, [isAgentDeniedPath, mounted, router, user?.role_slug])

	useEffect(() => {
		if (!mounted || user?.role_slug !== 'supervisor') return
		if (isSupervisorDeniedPath) router.replace('/dashboard')
	}, [isSupervisorDeniedPath, mounted, router, user?.role_slug])

  useEffect(() => {
    const onShortcut = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        globalSearchRef.current?.focus()
      }
    }
    window.addEventListener('keydown', onShortcut)
    return () => window.removeEventListener('keydown', onShortcut)
  }, [])

	if (!mounted || isLoading || !isAuthenticated || isAgentDeniedPath || isSupervisorDeniedPath) return null

  return (
    <div className="app-shell min-h-screen bg-[#050b16] text-slate-100 dark:bg-[#050b16] dark:text-slate-100">
      <header className="fixed inset-x-0 top-0 z-40 flex h-14 items-center border-b border-slate-800 bg-[#06111f]/95 px-4 backdrop-blur md:hidden">
        <button
          type="button"
          onClick={() => setMobileSidebarOpen(!mobileSidebarOpen)}
          className="rounded-lg p-2 text-slate-300 hover:bg-white/10"
          aria-label="Abrir menu"
        >
          <Menu size={22} />
        </button>
        <SafeImage src="/logo-black.png" alt="VGON" className="ml-3 h-9 w-auto object-contain dark:hidden" />
        <SafeImage src="/assets/images/logo-vgon-negativo.png" fallbackSrc="/logo-white.png" alt="VGON" className="ml-3 hidden h-9 w-auto object-contain dark:block" />
      </header>
      <Sidebar />
      <main className={`${sidebarPinned || sidebarHovered ? 'md:ml-64' : 'md:ml-20'} app-main min-h-screen min-w-0 bg-[#050b16] pt-14 transition-all duration-300 md:pt-0`}>
        <div className="app-topbar hidden h-16 items-center gap-4 border-b border-slate-800 bg-[#050b16]/95 px-6 backdrop-blur md:flex">
          <form className="relative max-w-xl flex-1" onSubmit={(event) => { event.preventDefault(); const value = globalSearch.trim(); if (value) router.push(`/search?q=${encodeURIComponent(value)}`) }}>
            <Search size={17} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
            <input ref={globalSearchRef} value={globalSearch} onChange={(event) => setGlobalSearch(event.target.value)} placeholder="Buscar no sistema..." aria-label="Busca global" className="h-10 w-full rounded-xl border border-slate-700 bg-slate-50 dark:bg-slate-900/80 pl-10 pr-20 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/10" />
            <span className="pointer-events-none absolute right-3 top-1/2 flex -translate-y-1/2 items-center gap-1 rounded-md border border-slate-700 px-1.5 py-0.5 text-[10px] text-slate-500"><Command size={11} />K</span>
          </form>
          <div className="ml-auto flex items-center gap-2 text-slate-400"><button type="button" className="rounded-lg p-2 hover:bg-white/10 hover:text-white" aria-label="Notificações"><Bell size={18} /></button><button type="button" className="rounded-lg p-2 hover:bg-white/10 hover:text-white" aria-label="Ajuda"><HelpCircle size={18} /></button></div>
        </div>
        {children}
      </main>
    </div>
  )
}
