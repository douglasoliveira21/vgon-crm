'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import {
  ChevronRight,
} from 'lucide-react'
import { clsx } from 'clsx'
import { ChannelIcon, channelSurface } from '@/components/channel-icon'

interface Channel {
  id: string
  name: string
  type: string
  status: string
  is_active: boolean
}

interface InboxStats {
  channel_id: string
  open_count: number
  unread_count: number
}

export default function InboxPage() {
  const router = useRouter()
  const [channels, setChannels] = useState<Channel[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchChannels()
  }, [])

  const fetchChannels = async () => {
    try {
      const response = await api.get('/channels')
      setChannels(response.data.channels || [])
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const getChannelIcon = (type: string) => {
    return <ChannelIcon type={type} size={20} />
  }

  const getChannelColor = (type: string) => {
    return channelSurface(type)
  }

  const getChannelLabel = (type: string) => {
    switch (type) {
      case 'whatsapp': return 'WhatsApp'
      case 'email': return 'E-mail'
      case 'webchat': return 'Chat do Site'
      default: return type
    }
  }

  return (
    <div className="mx-auto max-w-4xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Caixa de Entrada</h1>
          <p className="text-gray-500 mt-1">Todas as suas conversas organizadas por canal</p>
        </div>
      </div>

      {/* All conversations */}
      <button
        onClick={() => router.push('/conversations')}
        className="card p-5 w-full mb-4 flex items-center justify-between hover:shadow-md transition-shadow text-left"
      >
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 bg-gray-100 rounded-xl flex items-center justify-center">
            <ChannelIcon type="inbox" size={24} />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">Todas as conversas</h3>
            <p className="text-sm text-gray-500">Ver todas as conversas de todos os canais</p>
          </div>
        </div>
        <ChevronRight size={20} className="text-gray-400" />
      </button>

      {/* Channel inboxes */}
      <h2 className="text-sm font-medium text-gray-400 uppercase mb-3 mt-6">Canais ativos</h2>

      <div className="space-y-3">
        {channels.filter(c => c.status === 'connected' || c.is_active).map((channel) => (
          <button
            key={channel.id}
            onClick={() => router.push(`/conversations?channel=${channel.id}`)}
            className="card p-5 w-full flex items-center justify-between hover:shadow-md transition-shadow text-left"
          >
            <div className="flex items-center gap-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${getChannelColor(channel.type)}`}>
                {getChannelIcon(channel.type)}
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">{channel.name}</h3>
                <p className="text-sm text-gray-500">{getChannelLabel(channel.type)}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={clsx(
                'badge',
                channel.status === 'connected' ? 'badge-green' : 'badge-gray'
              )}>
                {channel.status === 'connected' ? 'Ativo' : 'Inativo'}
              </span>
              <ChevronRight size={20} className="text-gray-400" />
            </div>
          </button>
        ))}
      </div>

      {channels.length === 0 && !loading && (
        <div className="card p-12 text-center mt-4">
          <ChannelIcon type="inbox" size={40} className="mx-auto mb-4 text-gray-300" />
          <p className="text-gray-500 mb-4">Nenhum canal configurado</p>
          <button
            onClick={() => router.push('/channels')}
            className="btn-primary inline-flex"
          >
            Configurar canais
          </button>
        </div>
      )}

      {/* Quick setup */}
      {channels.length > 0 && (
        <div className="mt-8">
          <h2 className="text-sm font-medium text-gray-400 uppercase mb-3">Adicionar mais canais</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <button
              onClick={() => router.push('/channels')}
              className="card p-4 text-left hover:shadow-md transition-shadow"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
                  <ChannelIcon type="whatsapp" size={18} />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">WhatsApp</p>
                  <p className="text-xs text-gray-400">Via Evolution API</p>
                </div>
              </div>
            </button>

            <button
              onClick={() => router.push('/channels?tab=email')}
              className="card p-4 text-left hover:shadow-md transition-shadow"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
                  <ChannelIcon type="email" size={18} />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">E-mail</p>
                  <p className="text-xs text-gray-400">Gmail, Outlook, IMAP</p>
                </div>
              </div>
            </button>

            <button
              onClick={() => router.push('/widget')}
              className="card p-4 text-left hover:shadow-md transition-shadow"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center">
                  <ChannelIcon type="webchat" size={18} />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">Chat do Site</p>
                  <p className="text-xs text-gray-400">Widget embed</p>
                </div>
              </div>
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
