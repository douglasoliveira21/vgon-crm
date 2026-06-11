'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

// Inbox redirects to conversations with 'open' filter
export default function InboxPage() {
  const router = useRouter()

  useEffect(() => {
    router.push('/conversations')
  }, [router])

  return null
}
