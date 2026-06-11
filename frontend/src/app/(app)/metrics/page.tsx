'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import {
  MessageSquare,
  TrendingUp,
  DollarSign,
  Phone,
  Users,
  Clock,
  CheckCircle,
  AlertCircle,
} from 'lucide-react'

export default function MetricsPage() {
  const [metrics, setMetrics] = useState<any>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchMetrics()
  }, [])

  const fetchMetrics = async () => {
    try {
      const response = await api.get('/metrics')
      setMetrics(response.data.metrics || {})
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="p-6 animate-pulse">
        <div className="h-8 bg-gray-200 rounded w-48 mb-6"></div>
        <div className="grid grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-28 bg-gray-200 rounded-xl"></div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Métricas e Relatórios</h1>
          <p className="text-gray-500 mt-1">Acompanhe o desempenho da sua equipe</p>
        </div>
        <button className="btn-secondary">
          Exportar CSV
        </button>
      </div>

      {/* Main Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <MetricCard
          label="Atendimentos"
          value={metrics.total_conversations || 0}
          icon={<MessageSquare size={20} />}
          color="blue"
          change="+16%"
        />
        <MetricCard
          label="Mensagens"
          value={metrics.total_messages || 0}
          icon={<TrendingUp size={20} />}
          color="green"
          change="+22%"
        />
        <MetricCard
          label="Vendas"
          value={`R$ ${(metrics.deals_won_value || 0).toLocaleString('pt-BR')}`}
          icon={<DollarSign size={20} />}
          color="purple"
          change="+23%"
        />
        <MetricCard
          label="Chamadas"
          value={metrics.total_calls || 0}
          icon={<Phone size={20} />}
          color="orange"
          change="+12%"
        />
      </div>

      {/* Secondary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
              <CheckCircle size={20} className="text-green-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Conversas resolvidas</p>
              <p className="text-xl font-bold text-gray-900">{metrics.resolved_conversations || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-yellow-100 rounded-lg flex items-center justify-center">
              <AlertCircle size={20} className="text-yellow-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Conversas abertas</p>
              <p className="text-xl font-bold text-gray-900">{metrics.open_conversations || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
              <Users size={20} className="text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Contatos</p>
              <p className="text-xl font-bold text-gray-900">{metrics.total_contacts || 0}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Deals Overview */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Vendas</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Valor ganho</span>
              <span className="font-semibold text-green-600">
                R$ {(metrics.deals_won_value || 0).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Em negociação</span>
              <span className="font-semibold text-blue-600">
                R$ {(metrics.deals_open_value || 0).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Deals ganhos</span>
              <span className="font-semibold text-gray-900">{metrics.deals_won_count || 0}</span>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Performance</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Total atendimentos</span>
              <span className="font-semibold text-gray-900">{metrics.total_conversations || 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Conversas resolvidas</span>
              <span className="font-semibold text-gray-900">{metrics.resolved_conversations || 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Taxa de resolução</span>
              <span className="font-semibold text-green-600">
                {metrics.total_conversations > 0
                  ? Math.round((metrics.resolved_conversations / metrics.total_conversations) * 100)
                  : 0}%
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function MetricCard({
  label,
  value,
  icon,
  color,
  change,
}: {
  label: string
  value: number | string
  icon: React.ReactNode
  color: string
  change?: string
}) {
  const colors: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-600',
    green: 'bg-green-50 text-green-600',
    purple: 'bg-purple-50 text-purple-600',
    orange: 'bg-orange-50 text-orange-600',
  }

  return (
    <div className="card p-5">
      <div className="flex items-center justify-between mb-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colors[color]}`}>
          {icon}
        </div>
        {change && (
          <span className="text-xs font-medium text-green-600 bg-green-50 px-2 py-0.5 rounded-full">
            {change}
          </span>
        )}
      </div>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-sm text-gray-500 mt-0.5">{label}</p>
    </div>
  )
}
