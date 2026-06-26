'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import Sidebar from '@/components/layout/sidebar'
import SIPAutoRegister from '@/components/telephony/SIPAutoRegister'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const { isAuthenticated, checkAuth } = useAuthStore()

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login')
    } else {
      checkAuth()
    }
  }, [isAuthenticated, router, checkAuth])

  if (!isAuthenticated) return null

  return (
    <div className="min-h-screen bg-gray-50">
      <Sidebar />
      <main className="ml-64 min-h-screen">
        {children}
      </main>
      <SIPAutoRegister />
    </div>
  )
}
