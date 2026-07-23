'use client'

import { useEffect } from 'react'
import { useState } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useAppearanceStore } from '@/store/appearance'
import Sidebar from '@/components/layout/sidebar'

const agentDeniedPaths = ['/inbox', '/channels', '/automations', '/campaigns', '/metrics', '/audit-logs', '/widget']
const supervisorDeniedPaths = ['/inbox', '/channels', '/automations', '/campaigns', '/audit-logs', '/widget']

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, checkAuth, user } = useAuthStore()
  const { sidebarPinned, sidebarHovered, initAppearance } = useAppearanceStore()
  const [mounted, setMounted] = useState(false)
	const isAgentDeniedPath = user?.role_slug === 'agent' && agentDeniedPaths.some((path) => pathname === path || pathname.startsWith(path + '/'))
	const isSupervisorDeniedPath = user?.role_slug === 'supervisor' && supervisorDeniedPaths.some((path) => pathname === path || pathname.startsWith(path + '/'))

  useEffect(() => {
    setMounted(true)
    initAppearance()
    checkAuth()
  }, [initAppearance])

  useEffect(() => {
    if (!mounted) return

    const hasToken = !!localStorage.getItem('access_token')
    if (!hasToken) {
      router.push('/login')
    }
  }, [mounted, router])

  useEffect(() => {
	if (!mounted || user?.role_slug !== 'agent') return
	if (isAgentDeniedPath) router.replace('/dashboard')
  }, [isAgentDeniedPath, mounted, router, user?.role_slug])

	useEffect(() => {
		if (!mounted || user?.role_slug !== 'supervisor') return
		if (isSupervisorDeniedPath) router.replace('/dashboard')
	}, [isSupervisorDeniedPath, mounted, router, user?.role_slug])

	if (!mounted || !isAuthenticated || isAgentDeniedPath || isSupervisorDeniedPath) return null

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <Sidebar />
      <main className={`${sidebarPinned || sidebarHovered ? 'ml-64' : 'ml-20'} min-h-screen transition-all duration-300`}>
        {children}
      </main>
    </div>
  )
}
