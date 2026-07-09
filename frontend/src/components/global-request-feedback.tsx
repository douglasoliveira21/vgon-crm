'use client'

import { useEffect, useRef, useState } from 'react'
import { Loader2 } from 'lucide-react'

export function GlobalRequestFeedback() {
  const [pending, setPending] = useState(0)
  const [visible, setVisible] = useState(false)
  const timerRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    const handleActivity = (event: Event) => {
      const detail = (event as CustomEvent<{ active: boolean }>).detail
      setPending((current) => Math.max(0, current + (detail?.active ? 1 : -1)))
    }

    window.addEventListener('api:activity', handleActivity)
    return () => window.removeEventListener('api:activity', handleActivity)
  }, [])

  useEffect(() => {
    if (pending > 0) {
      if (timerRef.current) clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => setVisible(true), 450)
      return
    }

    if (timerRef.current) clearTimeout(timerRef.current)
    setVisible(false)
  }, [pending])

  if (!visible) return null

  return (
    <div className="fixed right-5 bottom-5 z-[9999] flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm text-white shadow-lg">
      <Loader2 size={16} className="animate-spin" />
      Processando...
    </div>
  )
}
