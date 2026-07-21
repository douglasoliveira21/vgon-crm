'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Ban, Loader2, Search, Unlock } from 'lucide-react'

const PAGE_SIZE = 25
type BlockedContact = { id: string; name?: string; phone?: string; email?: string; avatar_url?: string }

export default function BlockedContactsPage() {
  const [contacts, setContacts] = useState<BlockedContact[]>([])
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  const loadContacts = async () => {
    setLoading(true)
    try {
      const response = await api.get('/contacts', { params: { blocked: true, search, limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE } })
      setContacts(response.data.contacts || [])
      setTotal(response.data.total || 0)
    } catch { toast.error('Erro ao carregar contatos bloqueados') }
    finally { setLoading(false) }
  }

  useEffect(() => { loadContacts() }, [search, page])

  const unblock = async (contact: BlockedContact) => {
    if (!confirm(`Desbloquear ${contact.name || contact.phone || 'este contato'}?`)) return
    try {
      await api.post(`/contacts/${contact.id}/unblock`)
      toast.success('Contato desbloqueado')
      loadContacts()
    } catch (error: any) { toast.error(error.response?.data?.error || 'Erro ao desbloquear contato') }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="mx-auto max-w-6xl p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-red-50 text-red-600"><Ban size={20} /></div>
        <div><h1 className="text-2xl font-bold text-gray-900 dark:text-white">Contatos bloqueados</h1><p className="text-sm text-gray-500">{total} contatos bloqueados</p></div>
      </div>
      <div className="relative mb-6">
        <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input value={search} onChange={(event) => { setSearch(event.target.value); setPage(1) }} className="input pl-10" placeholder="Buscar por nome, telefone ou e-mail..." />
      </div>
      <div className="card overflow-hidden">
        {loading ? <div className="flex h-48 items-center justify-center"><Loader2 className="animate-spin text-primary-600" /></div> : contacts.length === 0 ? <div className="p-12 text-center text-gray-400">Nenhum contato bloqueado</div> : (
          <div className="divide-y divide-gray-100 dark:divide-gray-800">{contacts.map((contact) => (
            <div key={contact.id} className="flex items-center gap-4 px-5 py-4">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-gray-100 font-semibold text-gray-600">{contact.avatar_url ? <img src={contact.avatar_url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${contact.avatar_url}` : contact.avatar_url} alt="" className="h-full w-full object-cover" /> : (contact.name || '?').charAt(0).toUpperCase()}</div>
              <div className="min-w-0 flex-1"><p className="truncate font-medium text-gray-900 dark:text-white">{contact.name || 'Sem nome'}</p><p className="truncate text-sm text-gray-500">{contact.phone || contact.email || 'Sem telefone ou e-mail'}</p></div>
              <button type="button" onClick={() => unblock(contact)} className="btn-secondary flex items-center gap-2 text-sm"><Unlock size={15} /> Desbloquear</button>
            </div>
          ))}</div>
        )}
      </div>
      {totalPages > 1 && <div className="mt-4 flex items-center justify-between text-sm text-gray-600"><span>Página {page} de {totalPages}</span><div className="flex gap-2"><button className="btn-secondary" disabled={page === 1 || loading} onClick={() => setPage((value) => Math.max(1, value - 1))}>Anterior</button><button className="btn-secondary" disabled={page >= totalPages || loading} onClick={() => setPage((value) => Math.min(totalPages, value + 1))}>Próxima</button></div></div>}
    </div>
  )
}
