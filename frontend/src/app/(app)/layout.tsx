'use client'

import { useEffect } from 'react'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useAppearanceStore } from '@/store/appearance'
import Sidebar from '@/components/layout/sidebar'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const { isAuthenticated, checkAuth } = useAuthStore()
  const { sidebarPinned, sidebarHovered, initAppearance } = useAppearanceStore()
  const [mounted, setMounted] = useState(false)

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

  if (!mounted || !isAuthenticated) return null

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <Sidebar />
      <main className={`${sidebarPinned || sidebarHovered ? 'ml-64' : 'ml-20'} min-h-screen transition-all duration-300`}>
        {children}
      </main>
    </div>
  )
}
