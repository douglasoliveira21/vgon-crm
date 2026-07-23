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
    fetchUsers()
  }, [])

  useEffect(() => {
    fetchMetrics()
    fetchAttendanceMetrics()
  }, [selectedUser])

  const fetchMetrics = async () => {
    setLoading(true)
    try {
      const params: any = {}
      if (selectedUser) params.assigned_to = selectedUser
      const response = await api.get('/metrics', { params })
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
    <div className="animate-pulse p-4 sm:p-6">
        <div className="h-8 bg-gray-200 dark:bg-gray-800 rounded w-48 mb-6"></div>
      <div className="grid grid-cols-1 gap-3 min-[420px]:grid-cols-2 lg:grid-cols-4 lg:gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-28 bg-gray-200 dark:bg-gray-800 rounded-xl"></div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-7xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Métricas e Relatórios</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">Acompanhe o desempenho da sua equipe</p>
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
            <div className="w-10 h-10 bg-green-100 dark:bg-green-500/15 rounded-lg flex items-center justify-center">
              <CheckCircle size={20} className="text-green-600 dark:text-green-400" />
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Conversas resolvidas</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">{metrics.resolved_conversations || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-yellow-100 dark:bg-yellow-500/15 rounded-lg flex items-center justify-center">
              <AlertCircle size={20} className="text-yellow-600 dark:text-yellow-400" />
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Conversas abertas</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">{metrics.open_conversations || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-blue-100 dark:bg-blue-500/15 rounded-lg flex items-center justify-center">
              <Users size={20} className="text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Contatos</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">{metrics.total_contacts || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-emerald-100 dark:bg-emerald-500/15 rounded-lg flex items-center justify-center">
              <Headphones size={20} className="text-emerald-600 dark:text-emerald-400" />
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Atendidos dentro do SLA</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">{metrics.sla_within || 0}</p>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-red-100 dark:bg-red-500/15 rounded-lg flex items-center justify-center">
              <Timer size={20} className="text-red-600 dark:text-red-400" />
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">SLA estourados</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">{metrics.sla_breached || 0}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Deals Overview */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Vendas</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Valor ganho</span>
              <span className="font-semibold text-green-600 dark:text-green-400">
                R$ {(metrics.deals_won_value || 0).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Em negociação</span>
              <span className="font-semibold text-blue-600 dark:text-blue-400">
                R$ {(metrics.deals_open_value || 0).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Deals ganhos</span>
              <span className="font-semibold text-gray-900 dark:text-white">{metrics.deals_won_count || 0}</span>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Performance</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Total atendimentos</span>
              <span className="font-semibold text-gray-900 dark:text-white">{metrics.total_conversations || 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Conversas resolvidas</span>
              <span className="font-semibold text-gray-900 dark:text-white">{metrics.resolved_conversations || 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Taxa de resolução</span>
              <span className="font-semibold text-green-600 dark:text-green-400">
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
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Métricas de Atendimento</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">Chamados atendidos, resolvidos e tempos</p>
          </div>
          <select
            value={selectedUser}
            onChange={(e) => setSelectedUser(e.target.value)}
            className="text-sm border border-gray-200 dark:border-gray-700 rounded-lg px-3 py-2 text-gray-700 dark:text-gray-100 bg-gray-50 dark:bg-gray-800 focus:border-primary-500 outline-none min-w-[200px]"
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
              <div key={i} className="h-24 bg-gray-100 dark:bg-gray-800 rounded-xl animate-pulse"></div>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="bg-blue-50 dark:bg-blue-500/10 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-blue-100 dark:bg-blue-500/15 rounded-lg flex items-center justify-center">
                  <Headphones size={18} className="text-blue-600 dark:text-blue-400" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{attendance.attended}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Chamados Atendidos</p>
            </div>

            <div className="bg-green-50 dark:bg-green-500/10 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-green-100 dark:bg-green-500/15 rounded-lg flex items-center justify-center">
                  <CheckCircle size={18} className="text-green-600 dark:text-green-400" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{attendance.resolved}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Chamados Resolvidos</p>
            </div>

            <div className="bg-purple-50 dark:bg-purple-500/10 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-purple-100 dark:bg-purple-500/15 rounded-lg flex items-center justify-center">
                  <Clock size={18} className="text-purple-600 dark:text-purple-400" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{formatTime(attendance.total_time_minutes)}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Tempo Total de Atendimento</p>
            </div>

            <div className="bg-orange-50 dark:bg-orange-500/10 rounded-xl p-4">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-9 h-9 bg-orange-100 dark:bg-orange-500/15 rounded-lg flex items-center justify-center">
                  <Timer size={18} className="text-orange-600 dark:text-orange-400" />
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{formatTime(attendance.avg_time_minutes)}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Média de Tempo de Atendimento</p>
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
    blue: 'bg-blue-50 text-blue-600 dark:bg-blue-500/15 dark:text-blue-400',
    green: 'bg-green-50 text-green-600 dark:bg-green-500/15 dark:text-green-400',
    purple: 'bg-purple-50 text-purple-600 dark:bg-purple-500/15 dark:text-purple-400',
    orange: 'bg-orange-50 text-orange-600 dark:bg-orange-500/15 dark:text-orange-400',
  }

  return (
    <div className="card p-5">
      <div className="flex items-center justify-between mb-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colors[color]}`}>
          {icon}
        </div>
        {change && (
          <span className="text-xs font-medium text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-500/15 px-2 py-0.5 rounded-full">
            {change}
          </span>
        )}
      </div>
      <p className="text-2xl font-bold text-gray-900 dark:text-white">{value}</p>
      <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{label}</p>
    </div>
  )
}
