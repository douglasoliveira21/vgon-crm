'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  Building2,
  Users,
  MessageSquare,
  MessagesSquare,
  TrendingUp,
  AlertTriangle,
  Wifi,
} from 'lucide-react'

interface AdminStats {
  total_tenants: number
  active_tenants: number
  total_users: number
  active_users: number
  total_conversations: number
  open_conversations: number
  total_messages: number
  new_tenants_month: number
  plan_distribution: Record<string, number>
  online_users: number
  whatsapp_issues: number
  incidents_24h: number
}

export default function AdminDashboardPage() {
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchStats()
  }, [])

  const fetchStats = async () => {
    try {
      const response = await api.get('/admin/stats')
      setStats(response.data)
    } catch (error) {
      toast.error('Erro ao carregar estatísticas')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>
    )
  }

  const statCards = [
    {
      title: 'Total de Empresas',
      value: stats?.total_tenants || 0,
      subtitle: `${stats?.active_tenants || 0} ativas`,
      icon: Building2,
      color: 'bg-blue-500',
    },
    {
      title: 'Usuários Online',
      value: stats?.online_users || 0,
      subtitle: `${stats?.total_users || 0} cadastrados`,
      icon: Users,
      color: 'bg-green-500',
    },
    {
      title: 'WhatsApp com falha',
      value: stats?.whatsapp_issues || 0,
      subtitle: 'instâncias exigindo atenção',
      icon: Wifi,
      color: 'bg-red-500',
    },
    {
      title: 'Incidentes (24h)',
      value: stats?.incidents_24h || 0,
      subtitle: 'falhas operacionais recentes',
      icon: AlertTriangle,
      color: 'bg-amber-500',
    },
    {
      title: 'Conversas',
      value: stats?.total_conversations || 0,
      subtitle: `${stats?.open_conversations || 0} abertas`,
      icon: MessageSquare,
      color: 'bg-purple-500',
    },
    {
      title: 'Mensagens',
      value: stats?.total_messages || 0,
      subtitle: 'total enviadas/recebidas',
      icon: MessagesSquare,
      color: 'bg-orange-500',
    },
    {
      title: 'Novos Tenants (30d)',
      value: stats?.new_tenants_month || 0,
      subtitle: 'últimos 30 dias',
      icon: TrendingUp,
      color: 'bg-indigo-500',
    },
  ]

  return (
    <div>
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">Dashboard Admin</h1>
        <p className="text-gray-400 mt-1">Visão geral da plataforma</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-6 mb-8">
        {statCards.map((card) => {
          const Icon = card.icon
          return (
            <div
              key={card.title}
              className="bg-gray-800 border border-gray-700 rounded-xl p-6"
            >
              <div className="flex items-center justify-between mb-4">
                <div className={`w-10 h-10 rounded-lg ${card.color} flex items-center justify-center`}>
                  <Icon className="w-5 h-5 text-white" />
                </div>
              </div>
              <p className="text-2xl font-bold text-white">
                {card.value.toLocaleString('pt-BR')}
              </p>
              <p className="text-sm text-gray-400 mt-1">{card.title}</p>
              <p className="text-xs text-gray-500 mt-1">{card.subtitle}</p>
            </div>
          )
        })}
      </div>

      <div className="rounded-lg border border-gray-700 bg-gray-800 p-6 text-sm text-gray-300">
        Use <strong className="text-white">Operações</strong> para acompanhar saúde, incidentes e sessões; e <strong className="text-white">Auditoria Global</strong> para investigar ações em todos os tenants.
      </div>
    </div>
  )
}
