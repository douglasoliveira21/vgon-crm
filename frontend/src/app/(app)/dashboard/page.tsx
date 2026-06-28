'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/auth'
import api from '@/lib/api'
import {
  MessageSquare,
  Clock,
  UserCheck,
  TrendingUp,
  Wifi,
  WifiOff,
  Plus,
  Pin,
} from 'lucide-react'
import Link from 'next/link'

interface DashboardData {
  stats: {
    open_conversations: number
    pending_conversations: number
    my_conversations: number
    active_deals: number
    total_deals_value: number
    connected_channels: number
    avg_response_time: number
  }
  announcements: Array<{
    id: string
    title: string
    content: string
    priority: string
    is_pinned: boolean
    author_name: string
    published_at: string
    is_read: boolean
  }>
  channels: Array<{
    id: string
    name: string
    type: string
    status: string
  }>
}

export default function DashboardPage() {
  const { user } = useAuthStore()
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchDashboard()
  }, [])

  const fetchDashboard = async () => {
    try {
      const response = await api.get('/dashboard')
      setData(response.data)
    } catch (error) {
      console.error('Failed to fetch dashboard:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-8 bg-gray-200 rounded w-64 mb-6"></div>
        <div className="grid grid-cols-4 gap-4 mb-6">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-24 bg-gray-200 rounded-xl"></div>
          ))}
        </div>
      </div>
    )
  }

  const stats = data?.stats

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">
          Bem-vindo, {user?.name?.split(' ')[0]}! 👋
        </h1>
        <p className="text-gray-500 mt-1">
          Aqui está o resumo do seu dia
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          label="Conversas abertas"
          value={stats?.open_conversations || 0}
          icon={<MessageSquare size={20} />}
          color="blue"
        />
        <StatCard
          label="Aguardando atendimento"
          value={stats?.pending_conversations || 0}
          icon={<Clock size={20} />}
          color="yellow"
        />
        <StatCard
          label="Atribuídas a mim"
          value={stats?.my_conversations || 0}
          icon={<UserCheck size={20} />}
          color="green"
        />
        <StatCard
          label="Vendas em andamento"
          value={stats?.active_deals || 0}
          subtitle={`R$ ${(stats?.total_deals_value || 0).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}`}
          icon={<TrendingUp size={20} />}
          color="purple"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Channels Status */}
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Canais</h2>
            <Link
              href="/channels"
              className="text-primary-600 hover:text-primary-700 text-sm font-medium"
            >
              Ver todos
            </Link>
          </div>

          {data?.channels && data.channels.length > 0 ? (
            <div className="space-y-3">
              {data.channels.map((channel) => (
                <div key={channel.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-green-100 rounded-full flex items-center justify-center">
                      <MessageSquare size={16} className="text-green-600" />
                    </div>
                    <span className="text-sm font-medium text-gray-700">{channel.name}</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {channel.status === 'connected' ? (
                      <>
                        <Wifi size={14} className="text-green-500" />
                        <span className="text-xs text-green-600 font-medium">Conectado</span>
                      </>
                    ) : (
                      <>
                        <WifiOff size={14} className="text-red-500" />
                        <span className="text-xs text-red-600 font-medium">Desconectado</span>
                      </>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <div className="w-12 h-12 bg-primary-100 rounded-full flex items-center justify-center mx-auto mb-3">
                <Plus size={24} className="text-primary-600" />
              </div>
              <p className="text-gray-500 text-sm mb-3">Nenhum canal conectado</p>
              <Link
                href="/channels"
                className="btn-primary inline-flex text-sm"
              >
                Conectar WhatsApp
              </Link>
            </div>
          )}
        </div>

        {/* Announcements */}
        <div className="card p-6 lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Recados internos</h2>
          </div>

          {data?.announcements && data.announcements.length > 0 ? (
            <div className="space-y-3">
              {data.announcements.map((announcement) => (
                <div
                  key={announcement.id}
                  className={`p-4 rounded-lg border ${
                    announcement.is_pinned
                      ? 'border-primary-200 bg-primary-50'
                      : 'border-gray-100 bg-gray-50'
                  } ${!announcement.is_read ? 'border-l-4 border-l-primary-500' : ''}`}
                >
                  <div className="flex items-start gap-2">
                    {announcement.is_pinned && (
                      <Pin size={14} className="text-primary-500 mt-0.5 flex-shrink-0" />
                    )}
                    <div className="flex-1">
                      <h3 className="text-sm font-semibold text-gray-900">{announcement.title}</h3>
                      <p className="text-sm text-gray-600 mt-1 line-clamp-2">{announcement.content}</p>
                      <p className="text-xs text-gray-400 mt-2">
                        por {announcement.author_name}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-400 text-sm text-center py-8">Nenhum recado no momento</p>
          )}
        </div>
      </div>

      {/* Quick Actions */}
      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-4">
        <Link href="/conversations" className="card p-4 hover:shadow-md transition-shadow flex items-center gap-4">
          <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
            <MessageSquare size={20} className="text-blue-600" />
          </div>
          <div>
            <p className="font-medium text-gray-900">Iniciar atendimento</p>
            <p className="text-sm text-gray-500">Ver conversas abertas</p>
          </div>
        </Link>

        <Link href="/channels" className="card p-4 hover:shadow-md transition-shadow flex items-center gap-4">
          <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
            <Wifi size={20} className="text-green-600" />
          </div>
          <div>
            <p className="font-medium text-gray-900">Conectar WhatsApp</p>
            <p className="text-sm text-gray-500">Adicionar canal</p>
          </div>
        </Link>

      </div>
    </div>
  )
}

function StatCard({
  label,
  value,
  subtitle,
  icon,
  color,
}: {
  label: string
  value: number
  subtitle?: string
  icon: React.ReactNode
  color: 'blue' | 'green' | 'yellow' | 'purple'
}) {
  const colors = {
    blue: 'bg-blue-50 text-blue-600',
    green: 'bg-green-50 text-green-600',
    yellow: 'bg-yellow-50 text-yellow-600',
    purple: 'bg-purple-50 text-purple-600',
  }

  return (
    <div className="card p-5">
      <div className="flex items-center justify-between mb-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colors[color]}`}>
          {icon}
        </div>
      </div>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-sm text-gray-500 mt-0.5">{label}</p>
      {subtitle && (
        <p className="text-sm font-medium text-green-600 mt-1">{subtitle}</p>
      )}
    </div>
  )
}
