'use client'

import { useEffect, useState } from 'react'

export function AppLoader() {
  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (document.readyState === 'complete') {
      const timeout = setTimeout(() => setReady(true), 250)
      return () => clearTimeout(timeout)
    }

    const handleLoad = () => setTimeout(() => setReady(true), 250)
    window.addEventListener('load', handleLoad)
    return () => window.removeEventListener('load', handleLoad)
  }, [])

  if (ready) return null

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-white dark:bg-gray-950">
      <div className="flex flex-col items-center gap-4">
        <div className="h-12 w-12 animate-spin rounded-full border-4 border-gray-200 border-t-primary-600 dark:border-gray-800 dark:border-t-primary-400" />
        <div className="text-center">
          <p className="text-sm font-semibold text-gray-900 dark:text-white">Carregando CRM</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">Preparando scripts, plugins e interface...</p>
        </div>
      </div>
    </div>
  )
}
