'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  MessageSquare,
  TrendingUp,
  DollarSign,
  Users,
  Clock,
  CheckCircle,
  AlertCircle,
  Headphones,
  Timer,
} from 'lucide-react'

interface UserItem {
  id: string
  name: string
  email: string
  role_name?: string
}

interface AttendanceMetrics {
  attended: number
  resolved: number
  total_time_minutes: number
  avg_time_minutes: number
}

export default function MetricsPage() {
  const [metrics, setMetrics] = useState<any>({})
  const [loading, setLoading] = useState(true)
  const [users, setUsers] = useState<UserItem[]>([])
  const [selectedUser, setSelectedUser] = useState('')
  const [attendance, setAttendance] = useState<AttendanceMetrics>({ attended: 0, resolved: 0, total_time_minutes: 0, avg_time_minutes: 0 })
  const [loadingAttendance, setLoadingAttendance] = useState(true)

  useEffect(() => {
    fetchMetrics()
    fetchUsers()
  }, [])

  useEffect(() => {
    fetchAttendanceMetrics()
  }, [selectedUser])

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

  const fetchUsers = async () => {
    try {
      const response = await api.get('/users')
      setUsers(response.data.users || [])
    } catch {}
  }

  const fetchAttendanceMetrics = async () => {
    setLoadingAttendance(true)
    try {
      const params: any = {}
      if (selectedUser) params.assigned_to = selectedUser
      const response = await api.get('/metrics/attendance', { params })
      setAttendance(response.data || { attended: 0, resolved: 0, total_time_minutes: 0, avg_time_minutes: 0 })
    } catch {
      setAttendance({ attended: 0, resolved: 0, total_time_minutes: 0, avg_time_minutes: 0 })
    } finally {
      setLoadingAttendance(false)
    }
  }

  const formatTime = (minutes: number) => {
    if (minutes < 60) return `${Math.round(minutes)} min`
    const hours = Math.floor(minutes / 60)
    const mins = Math.round(minutes % 60)
    return `${hours}h ${mins}min`
  }

  const exportSLAReport = async () => {
    try {
      const response = await api.get('/customer-companies/sla-report.csv', { responseType: 'blob' })
      const url = window.URL.createObjectURL(new Blob([response.data], { type: 'text/csv;charset=utf-8;' }))
      const link = document.createElement('a')
      link.href = url
      link.download = `relatorio-sla-empresas-${new Date().toISOString().slice(0, 10)}.csv`
      link.click()
      window.URL.revokeObjectURL(url)
    } catch {
      toast.error('Erro ao exportar relatório de SLA')
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
        <button onClick={exportSLAReport} className="btn-secondary">
          Exportar relatório SLA
        </button>
      </div>

      {/* Main Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-4 mb-8">
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
      </div>

      {/* Secondary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-4 mb-8">
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

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-emerald-100 rounded-lg flex items-center justify-center">
              <Headphones size={20} className="text-emerald-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Atendidos dentro do SLA</p>
              <p className="text-xl font-bold text-gray-900">{metrics.sla_within || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-red-100 rounded-lg flex items-center justify-center">
              <Timer size={20} className="text-red-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">SLA estourados</p>
              <p className="text-xl font-bold text-gray-900">{metrics.sla_breached || 0}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Deals Overview */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
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

      {/* Attendance Metrics */}
      <div className="card p-6">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Métricas de Atendimento</h3>
            <p className="text-sm text-gray-500 mt-0.5">Chamados atendidos, resolvidos e tempos</p>
          </div>
          <select
            value={selectedUser}
            onChange={(e) => setSelectedUser(e.target.value)}
            className="text-sm border border-gray-200 rounded-lg px-3 py-2 text-gray-700 bg-gray-50 focus:border-primary-500 outline-none min-w-[200px]"
          >
            <option value="">Todos os atendentes</option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>{u.name}</option>
            ))}
          </select>
        </div>

        {loadingAttendance ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="h-24 bg-gray-100 rounded-xl animate-pulse"></div>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="bg-blue-50 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-blue-100 rounded-lg flex items-center justify-center">
                  <Headphones size={18} className="text-blue-600" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900">{attendance.attended}</p>
              <p className="text-xs text-gray-500 mt-0.5">Chamados Atendidos</p>
            </div>

            <div className="bg-green-50 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-green-100 rounded-lg flex items-center justify-center">
                  <CheckCircle size={18} className="text-green-600" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900">{attendance.resolved}</p>
              <p className="text-xs text-gray-500 mt-0.5">Chamados Resolvidos</p>
            </div>

            <div className="bg-purple-50 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-purple-100 rounded-lg flex items-center justify-center">
                  <Clock size={18} className="text-purple-600" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900">{formatTime(attendance.total_time_minutes)}</p>
              <p className="text-xs text-gray-500 mt-0.5">Tempo Total de Atendimento</p>
            </div>

            <div className="bg-orange-50 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-orange-100 rounded-lg flex items-center justify-center">
                  <Timer size={18} className="text-orange-600" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900">{formatTime(attendance.avg_time_minutes)}</p>
              <p className="text-xs text-gray-500 mt-0.5">Média de Tempo de Atendimento</p>
            </div>
          </div>
        )}
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
