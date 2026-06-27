'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import { useSIP } from '@/hooks/useSIP'
import { useAuthStore } from '@/store/auth'
import { Phone, PhoneIncoming, PhoneOff } from 'lucide-react'

export default function SIPAutoRegister() {
  const sip = useSIP()
  const { user } = useAuthStore()
  const [attempted, setAttempted] = useState(false)
  const [extensionInfo, setExtensionInfo] = useState<string>('')
  const [incomingContact, setIncomingContact] = useState<{ name: string; phone: string } | null>(null)

  useEffect(() => {
    if (attempted || sip.status === 'online' || sip.status === 'registering') return

    const autoRegister = async () => {
      try {
        const res = await api.get('/telephony/webrtc/config')
        const cfg = res.data
        if (!cfg?.username || !cfg?.password) return

        setExtensionInfo(cfg.extension_number)

        // Auto-register the SIP connection
        sip.register({
          server: cfg.sip_host,
          port: String(cfg.sip_port || '5060'),
          domain: cfg.webrtc_domain || cfg.sip_domain,
          websocketUrl: cfg.webrtc_ws_url,
          user: cfg.username,
          password: cfg.password,
          displayName: cfg.display_name || user?.name || cfg.username,
          transport: cfg.transport || 'WSS',
          stunServer: cfg.stun_server || 'stun:stun.l.google.com:19302',
        })
      } catch {
        // Silently fail - user can manually register from the calls page
      } finally {
        setAttempted(true)
      }
    }

    autoRegister()
  }, [attempted, sip.status, user])

  useEffect(() => {
    if (!sip.isIncoming || !sip.incomingNumber) {
      setIncomingContact(null)
      return
    }

    const audio = new Audio('/sounds/notification-1.wav')
    audio.loop = true
    audio.play().catch(() => {})

    api.get('/contacts', { params: { search: sip.incomingNumber, limit: 1 } })
      .then(res => {
        const contact = (res.data.contacts || [])[0]
        setIncomingContact(contact ? { name: contact.name, phone: contact.phone } : null)
      })
      .catch(() => setIncomingContact(null))

    return () => {
      audio.pause()
      audio.currentTime = 0
    }
  }, [sip.isIncoming, sip.incomingNumber])

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
    <>
      {sip.isIncoming && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[70] p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-sm text-center shadow-xl">
            <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4 animate-pulse">
              <PhoneIncoming size={30} className="text-green-600" />
            </div>
            <h3 className="text-lg font-semibold text-gray-900">Chamada recebida</h3>
            <p className="text-xl font-medium text-gray-800 mt-2">{sip.incomingNumber}</p>
            {incomingContact && (
              <p className="text-sm text-primary-700 mt-2">{incomingContact.name}</p>
            )}
            <div className="flex justify-center gap-4 mt-6">
              <button onClick={() => sip.endCall()} className="p-4 bg-red-600 text-white rounded-full hover:bg-red-700" title="Recusar">
                <PhoneOff size={24} />
              </button>
              <button onClick={() => sip.answerCall()} className="p-4 bg-green-600 text-white rounded-full hover:bg-green-700" title="Atender">
                <Phone size={24} />
              </button>
            </div>
          </div>
        </div>
      )}

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
    </>
  )
}
