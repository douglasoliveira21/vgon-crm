'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import { FileText, RefreshCw, Search } from 'lucide-react'

interface AuditLog {
  id: string
  action: string
  entity_type?: string
  entity_id?: string
  user_name?: string
  user_email?: string
  ip_address?: string
  metadata?: string
  created_at: string
}

const actionLabels: Record<string, string> = {
  'contact.create': 'Contato criado',
  'contact.update': 'Contato editado',
  'contact.delete': 'Contato removido',
  'contact.export': 'Dados exportados',
  'contact.consent.update': 'Consentimento atualizado',
  'message.send': 'Mensagem enviada',
  'message.email_campaign.send': 'E-mail de campanha enviado',
  'campaign.email.send': 'Campanha de e-mail enviada',
  'whatsapp.instance.down': 'WhatsApp desconectado',
}

export default function AuditLogsPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [action, setAction] = useState('')

  useEffect(() => {
    fetchLogs()
  }, [])

  const fetchLogs = async () => {
    setLoading(true)
    try {
      const response = await api.get('/audit-logs', {
        params: { limit: 150, action: action || undefined },
      })
      setLogs(response.data.logs || [])
    } catch {
      setLogs([])
    } finally {
      setLoading(false)
    }
  }

  const formatMetadata = (raw?: string) => {
    if (!raw) return ''
    try {
      const parsed = JSON.parse(raw)
      return Object.entries(parsed)
        .map(([key, value]) => `${key}: ${String(value)}`)
        .join(' | ')
    } catch {
      return raw
    }
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Logs administrativos</h1>
          <p className="text-gray-500 mt-1">Auditoria de ações críticas, envios, exportações e alertas operacionais.</p>
        </div>
        <button type="button" onClick={fetchLogs} className="btn-secondary">
          <RefreshCw size={16} />
          Atualizar
        </button>
      </div>

      <div className="card p-4 mb-5">
        <div className="relative">
          <Search size={17} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={action}
            onChange={(event) => setAction(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') fetchLogs()
            }}
            className="input pl-9"
            placeholder="Filtrar por ação. Ex: contact, campaign, whatsapp..."
          />
        </div>
      </div>

      <div className="card overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">Evento</th>
              <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">Usuário</th>
              <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">Origem</th>
              <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">Detalhes</th>
              <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">Data</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {logs.map((log) => (
              <tr key={log.id} className="hover:bg-gray-50">
                <td className="px-5 py-4">
                  <div className="flex items-center gap-2">
                    <FileText size={16} className="text-primary-600" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">{actionLabels[log.action] || log.action}</p>
                      <p className="text-xs text-gray-400">{log.entity_type || '-'}</p>
                    </div>
                  </div>
                </td>
                <td className="px-5 py-4">
                  <p className="text-sm text-gray-700">{log.user_name || 'Sistema'}</p>
                  {log.user_email && <p className="text-xs text-gray-400">{log.user_email}</p>}
                </td>
                <td className="px-5 py-4 text-sm text-gray-600">{log.ip_address || '-'}</td>
                <td className="px-5 py-4 text-xs text-gray-500 max-w-md truncate">{formatMetadata(log.metadata) || '-'}</td>
                <td className="px-5 py-4 text-right text-sm text-gray-500">
                  {log.created_at ? new Date(log.created_at).toLocaleString('pt-BR') : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {logs.length === 0 && !loading && (
          <div className="p-12 text-center text-gray-400">Nenhum log encontrado.</div>
        )}
        {loading && (
          <div className="p-12 text-center text-gray-400">Carregando logs...</div>
        )}
      </div>
    </div>
  )
}
