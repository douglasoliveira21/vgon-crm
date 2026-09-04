'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'

type Audit = { id: string; action: string; entity_type: string; tenant_name: string; user_name: string; user_email: string; ip_address: string; metadata?: Record<string, unknown>; created_at: string }

export default function GlobalAuditPage() {
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 250)
    return () => clearTimeout(timer)
  }, [search])

  const { data: logs = [], isLoading: loading } = useQuery({
    queryKey: ['admin-audit-logs', debouncedSearch],
    queryFn: async () => {
      const response = await api.get('/admin/audit-logs', { params: { search: debouncedSearch, limit: 300 } })
      return (response.data.logs || []) as Audit[]
    },
  })

  return (
    <div>
      <div className="mb-6"><h1 className="text-2xl font-bold text-white">Auditoria Global</h1><p className="mt-1 text-sm text-gray-400">Ações realizadas em todos os tenants</p></div>
      <div className="relative mb-5 max-w-xl"><Search className="absolute left-3 top-3 h-4 w-4 text-gray-500" /><input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Buscar ação, tenant ou usuário" className="w-full rounded-lg border border-gray-700 bg-gray-800 py-2.5 pl-10 pr-4 text-sm text-white outline-none focus:border-indigo-500" /></div>
      <div className="overflow-x-auto rounded-lg border border-gray-700 bg-gray-800">
        <table className="w-full min-w-[760px] text-sm">
          <thead className="bg-gray-900 text-left text-xs uppercase text-gray-400"><tr><th className="px-4 py-3">Data</th><th>Ação</th><th>Tenant</th><th>Responsável</th><th>IP</th></tr></thead>
          <tbody className="divide-y divide-gray-700">
            {logs.map((log) => <tr key={log.id}><td className="px-4 py-3 text-gray-400">{new Date(log.created_at).toLocaleString('pt-BR')}</td><td><p className="font-medium text-white">{log.action}</p><p className="text-xs text-gray-500">{log.entity_type}</p></td><td className="text-gray-300">{log.tenant_name || '-'}</td><td><p className="text-gray-300">{log.user_name}</p><p className="text-xs text-gray-500">{log.user_email}</p></td><td className="text-gray-400">{log.ip_address || '-'}</td></tr>)}
          </tbody>
        </table>
        {!loading && !logs.length && <div className="p-8 text-center text-sm text-gray-400">Nenhum registro encontrado.</div>}
        {loading && <div className="p-8 text-center text-sm text-gray-400">Carregando...</div>}
      </div>
    </div>
  )
}
