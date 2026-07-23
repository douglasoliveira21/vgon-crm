'use client'

import { useEffect } from 'react'
import { useState } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useAppearanceStore } from '@/store/appearance'
import Sidebar from '@/components/layout/sidebar'
import { Menu } from 'lucide-react'
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

	if (!mounted || isLoading || !isAuthenticated || isAgentDeniedPath || isSupervisorDeniedPath) return null

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <header className="fixed inset-x-0 top-0 z-40 flex h-14 items-center border-b border-gray-200 bg-white px-4 md:hidden dark:border-gray-800 dark:bg-gray-900">
        <button
          type="button"
          onClick={() => setMobileSidebarOpen(!mobileSidebarOpen)}
          className="rounded-lg p-2 text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800"
          aria-label="Abrir menu"
        >
          <Menu size={22} />
        </button>
        <SafeImage src="/assets/images/logo-vgon-negativo.png" fallbackSrc="/logo-white.png" alt="VGON" className="ml-3 h-9 w-auto object-contain" />
      </header>
      <Sidebar />
      <main className={`${sidebarPinned || sidebarHovered ? 'md:ml-64' : 'md:ml-20'} min-h-screen min-w-0 pt-14 transition-all duration-300 md:pt-0`}>
        {children}
      </main>
    </div>
  )
}
