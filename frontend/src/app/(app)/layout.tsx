'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useAppearanceStore } from '@/store/appearance'
import Sidebar from '@/components/layout/sidebar'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const { isAuthenticated, checkAuth } = useAuthStore()
  const { sidebarPinned, sidebarHovered, initAppearance } = useAppearanceStore()

  useEffect(() => {
    initAppearance()
  }, [initAppearance])

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login')
    } else {
      checkAuth()
    }
  }, [isAuthenticated, router, checkAuth])

  if (!isAuthenticated) return null

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <Sidebar />
      <main className={`${sidebarPinned || sidebarHovered ? 'ml-64' : 'ml-20'} min-h-screen transition-all duration-300`}>
        {children}
      </main>
    </div>
  )
}
