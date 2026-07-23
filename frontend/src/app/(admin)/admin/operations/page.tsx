'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Activity, AlertTriangle, CheckCircle2, LogOut, RefreshCw, ShieldAlert, Wifi } from 'lucide-react'

type Health = {
  id: string
  name: string
  status: 'healthy' | 'attention' | 'suspended'
  online_users: number
  whatsapp_total: number
  whatsapp_connected: number
  whatsapp_issues: number
  last_activity?: string
}

type Incident = { id: string; action: string; tenant_name: string; metadata?: Record<string, unknown>; created_at: string }
type Session = { id: string; user_name: string; email: string; tenant_name: string; ip_address: string; user_agent: string; last_used_at: string; expires_at: string }
type Security = { failed_logins_24h: number; super_admins: number; super_admins_without_2fa: number; active_sessions: number }

export default function OperationsPage() {
  const [health, setHealth] = useState<Health[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [sessions, setSessions] = useState<Session[]>([])
  const [security, setSecurity] = useState<Security | null>(null)
  const [loading, setLoading] = useState(true)

  const load = async () => {
    setLoading(true)
    try {
      const [healthRes, incidentsRes, sessionsRes, securityRes] = await Promise.all([
        api.get('/admin/health'),
        api.get('/admin/incidents'),
        api.get('/admin/sessions'),
        api.get('/admin/security'),
      ])
      setHealth(healthRes.data.tenants || [])
      setIncidents(incidentsRes.data.incidents || [])
      setSessions(sessionsRes.data.sessions || [])
      setSecurity(securityRes.data)
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao carregar operações')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  const revoke = async (id: string) => {
    if (!confirm('Encerrar esta sessão?')) return
    try {
      await api.delete(`/admin/sessions/${id}`)
      setSessions((current) => current.filter((session) => session.id !== id))
      toast.success('Sessão revogada')
    } catch {
      toast.error('Erro ao revogar sessão')
    }
  }

  const retryIncident = async (id: string) => {
    try {
      const response = await api.post(`/admin/incidents/${id}/retry`)
      toast.success(`Verificação concluída: ${response.data.status}`)
      load()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Não foi possível repetir esta operação')
    }
  }

  const date = (value?: string) => value ? new Date(value).toLocaleString('pt-BR') : 'Sem atividade'

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Operações e Segurança</h1>
          <p className="mt-1 text-sm text-gray-400">Saúde dos tenants, incidentes e sessões ativas</p>
        </div>
        <button onClick={load} disabled={loading} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50">
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} /> Atualizar
        </button>
      </div>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {[
          ['Falhas de login (24h)', security?.failed_logins_24h || 0, ShieldAlert],
          ['Sessões ativas', security?.active_sessions || 0, Activity],
          ['Super administradores', security?.super_admins || 0, CheckCircle2],
          ['Super admins sem 2FA', security?.super_admins_without_2fa || 0, AlertTriangle],
        ].map(([label, value, Icon]: any) => (
          <div key={label} className="rounded-lg border border-gray-700 bg-gray-800 p-4">
            <Icon className="mb-3 h-5 w-5 text-indigo-400" />
            <p className="text-2xl font-bold text-white">{value}</p>
            <p className="text-xs text-gray-400">{label}</p>
          </div>
        ))}
      </div>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-white">Saúde dos tenants</h2>
        <div className="overflow-hidden rounded-lg border border-gray-700 bg-gray-800">
          <table className="w-full text-sm">
            <thead className="bg-gray-900 text-left text-xs uppercase text-gray-400"><tr><th className="px-4 py-3">Tenant</th><th>Status</th><th>WhatsApp</th><th>Usuários online</th><th>Última atividade</th></tr></thead>
            <tbody className="divide-y divide-gray-700">
              {health.map((item) => (
                <tr key={item.id}>
                  <td className="px-4 py-3 font-medium text-white">{item.name}</td>
                  <td><span className={`rounded px-2 py-1 text-xs ${item.status === 'healthy' ? 'bg-green-500/15 text-green-300' : item.status === 'attention' ? 'bg-amber-500/15 text-amber-300' : 'bg-red-500/15 text-red-300'}`}>{item.status === 'healthy' ? 'Saudável' : item.status === 'attention' ? 'Atenção' : 'Suspenso'}</span></td>
                  <td className="text-gray-300"><span className="inline-flex items-center gap-1"><Wifi className="h-3.5 w-3.5" /> {item.whatsapp_connected}/{item.whatsapp_total} conectados</span></td>
                  <td className="text-gray-300">{item.online_users}</td>
                  <td className="text-gray-400">{date(item.last_activity)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-white">Incidentes recentes</h2>
        <div className="space-y-2">
          {incidents.slice(0, 30).map((incident) => (
            <div key={incident.id} className="flex items-center gap-3 rounded-lg border border-gray-700 bg-gray-800 px-4 py-3">
              <AlertTriangle className="h-4 w-4 shrink-0 text-amber-400" />
              <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-white">{incident.action}</p><p className="text-xs text-gray-400">{incident.tenant_name || 'Plataforma'}</p></div>
              <time className="text-xs text-gray-500">{date(incident.created_at)}</time>
              <button onClick={() => retryIncident(incident.id)} className="rounded border border-gray-600 px-2.5 py-1.5 text-xs text-gray-300 hover:border-indigo-500 hover:text-white">Tentar novamente</button>
            </div>
          ))}
          {!incidents.length && <div className="rounded-lg border border-gray-700 bg-gray-800 p-6 text-center text-sm text-gray-400">Nenhum incidente registrado.</div>}
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-white">Sessões ativas</h2>
        <div className="overflow-hidden rounded-lg border border-gray-700 bg-gray-800">
          <table className="w-full text-sm">
            <thead className="bg-gray-900 text-left text-xs uppercase text-gray-400"><tr><th className="px-4 py-3">Usuário</th><th>Tenant</th><th>IP</th><th>Último uso</th><th className="text-right">Ação</th></tr></thead>
            <tbody className="divide-y divide-gray-700">
              {sessions.map((session) => (
                <tr key={session.id}>
                  <td className="px-4 py-3"><p className="text-white">{session.user_name}</p><p className="text-xs text-gray-500">{session.email}</p></td>
                  <td className="text-gray-300">{session.tenant_name}</td><td className="text-gray-400">{session.ip_address || '-'}</td><td className="text-gray-400">{date(session.last_used_at)}</td>
                  <td className="pr-4 text-right"><button onClick={() => revoke(session.id)} title="Encerrar sessão" className="rounded p-2 text-gray-400 hover:bg-red-500/10 hover:text-red-400"><LogOut className="h-4 w-4" /></button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
