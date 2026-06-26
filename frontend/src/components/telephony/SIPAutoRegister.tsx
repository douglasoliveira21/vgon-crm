'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import { useSIP } from '@/hooks/useSIP'
import { useAuthStore } from '@/store/auth'
import { Phone } from 'lucide-react'

export default function SIPAutoRegister() {
  const sip = useSIP()
  const { user } = useAuthStore()
  const [attempted, setAttempted] = useState(false)
  const [extensionInfo, setExtensionInfo] = useState<string>('')

  useEffect(() => {
    if (attempted || sip.status === 'online' || sip.status === 'registering') return

    const autoRegister = async () => {
      try {
        // Fetch provider config
        const providerRes = await api.get('/telephony/provider')
        const provider = providerRes.data.provider
        if (!provider) return

        // Fetch extensions and find one linked to the user
        const extRes = await api.get('/telephony/extensions')
        const extensions = extRes.data.extensions || []

        // Try to find an extension matching the user or use the first one
        let userExtension = extensions.find(
          (ext: any) => ext.user_id === user?.id || ext.display_name === user?.name
        )
        if (!userExtension && extensions.length > 0) {
          userExtension = extensions[0]
        }

        if (!userExtension) return

        setExtensionInfo(userExtension.extension_number)

        // Auto-register the SIP connection
        sip.register({
          server: provider.sip_host,
          port: String(provider.sip_port || '8089'),
          domain: provider.sip_domain || provider.sip_host,
          user: userExtension.extension_number,
          password: userExtension.extension_password || provider.sip_user,
          displayName: userExtension.display_name || user?.name || userExtension.extension_number,
          transport: provider.transport || 'WSS',
          stunServer: provider.stun_server || 'stun:stun.l.google.com:19302',
        })
      } catch {
        // Silently fail - user can manually register from the calls page
      } finally {
        setAttempted(true)
      }
    }

    autoRegister()
  }, [attempted, sip.status, user])

  const statusColor = sip.status === 'online'
    ? 'bg-green-500'
    : sip.status === 'registering'
    ? 'bg-yellow-500 animate-pulse'
    : sip.status === 'error'
    ? 'bg-red-500'
    : 'bg-gray-400'

  const statusLabel = sip.status === 'online'
    ? 'SIP Online'
    : sip.status === 'registering'
    ? 'Registrando...'
    : sip.status === 'error'
    ? 'SIP Erro'
    : 'SIP Offline'

  return (
    <div className="fixed bottom-4 left-4 z-40">
      <div className="flex items-center gap-2 bg-white border border-gray-200 rounded-full px-3 py-1.5 shadow-sm text-xs text-gray-600">
        <div className={`w-2.5 h-2.5 rounded-full ${statusColor}`} />
        <Phone size={12} />
        <span>{statusLabel}</span>
        {extensionInfo && sip.status === 'online' && (
          <span className="text-gray-400">({extensionInfo})</span>
        )}
      </div>
    </div>
  )
}
