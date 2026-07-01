'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Search, Plus, Edit2, Trash2, Phone, Mail, MapPin, Building, Tag, MessageSquare } from 'lucide-react'

const PAGE_SIZE = 25

interface Contact {
  id: string
  name?: string
  phone?: string
  email?: string
  customer_company_id?: string
  customer_company_name?: string
  company_name?: string
  position?: string
  city?: string
  origin?: string
  avatar_url?: string
  tags?: Array<{ id: string; name: string; color: string }>
  created_at: string
}

interface CustomerCompany {
  id: string
  name: string
  cnpj?: string
}

export default function ContactsPage() {
  const router = useRouter()
  const [contacts, setContacts] = useState<Contact[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [showForm, setShowForm] = useState(false)
  const [editingContact, setEditingContact] = useState<Contact | null>(null)
  const [companies, setCompanies] = useState<CustomerCompany[]>([])
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  useEffect(() => {
    fetchContacts()
  }, [search, page])

  useEffect(() => {
    fetchCompanies()
  }, [])

  const fetchContacts = async () => {
    setLoading(true)
    try {
      const response = await api.get('/contacts', {
        params: { search, limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE },
      })
      const sortedContacts = [...(response.data.contacts || [])].sort((a, b) =>
        (a.name || a.phone || a.email || '').localeCompare(b.name || b.phone || b.email || '', 'pt-BR', {
          sensitivity: 'base',
        })
      )
      setContacts(sortedContacts)
      setTotal(response.data.total || 0)
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchCompanies = async () => {
    try {
      const response = await api.get('/customer-companies')
      setCompanies(response.data.companies || [])
    } catch {}
  }

  const deleteContact = async (id: string) => {
    if (!confirm('Remover este contato?')) return
    try {
      await api.delete(`/contacts/${id}`)
      setContacts((prev) => prev.filter((c) => c.id !== id))
      toast.success('Contato removido')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  const startConversation = async (contact: Contact) => {
    if (!contact.phone) {
      toast.error('Contato não tem telefone')
      return
    }
    try {
      // Send initial message to create/reopen conversation
      await api.post('/conversations/start', { phone: contact.phone })
      toast.success('Conversa iniciada')
      router.push('/conversations')
    } catch {
      // If endpoint doesn't exist, just go to conversations
      router.push('/conversations')
    }
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Contatos</h1>
          <p className="text-gray-500 mt-1">{total} contatos registrados</p>
        </div>
        <button onClick={() => setShowForm(true)} className="btn-primary">
          <Plus size={18} />
          Novo contato
        </button>
      </div>

      {/* Search */}
      <div className="mb-6 relative">
        <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(1)
          }}
          placeholder="Buscar por nome, telefone, e-mail ou empresa..."
          className="input pl-10"
        />
      </div>

      {/* Contacts Table */}
      <div className="card overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">Contato</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">Telefone</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">E-mail</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">Empresa</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">Origem</th>
              <th className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase">Ações</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {contacts.map((contact) => (
              <tr key={contact.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <div className="w-9 h-9 bg-primary-100 rounded-full flex items-center justify-center overflow-hidden flex-shrink-0">
                      {contact.avatar_url ? (
                        <img
                          src={contact.avatar_url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${contact.avatar_url}` : contact.avatar_url}
                          alt=""
                          className="w-full h-full object-cover"
                          onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
                        />
                      ) : (
                        <span className="text-primary-700 text-sm font-medium">
                          {contact.name?.charAt(0)?.toUpperCase() || '?'}
                        </span>
                      )}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">{contact.name || '-'}</p>
                      {contact.city && (
                        <p className="text-xs text-gray-400 flex items-center gap-1">
                          <MapPin size={10} /> {contact.city}
                        </p>
                      )}
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-sm text-gray-600">{contact.phone || '-'}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{contact.email || '-'}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{contact.customer_company_name || contact.company_name || '-'}</td>
                <td className="px-6 py-4">
                  {contact.origin && (
                    <span className="badge badge-blue">{contact.origin}</span>
                  )}
                </td>
                <td className="px-6 py-4 text-right">
                  <div className="flex items-center justify-end gap-1">
                    <button
                      onClick={() => startConversation(contact)}
                      className="p-1.5 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded"
                      title="Enviar mensagem"
                    >
                      <MessageSquare size={14} />
                    </button>
                    <button
                      onClick={() => { setEditingContact(contact); setShowForm(true) }}
                      className="p-1.5 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded"
                    >
                      <Edit2 size={14} />
                    </button>
                    <button
                      onClick={() => deleteContact(contact.id)}
                      className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {contacts.length === 0 && !loading && (
          <div className="p-12 text-center">
            <p className="text-gray-400">Nenhum contato encontrado</p>
          </div>
        )}
      </div>

      {total > PAGE_SIZE && (
        <div className="flex items-center justify-between mt-4 text-sm text-gray-600">
          <span>
            Página {page} de {totalPages} · {total} contatos
          </span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              disabled={page === 1 || loading}
              className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Anterior
            </button>
            <button
              type="button"
              onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
              disabled={page >= totalPages || loading}
              className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Próxima
            </button>
          </div>
        </div>
      )}

      {/* Contact Form Modal */}
      {showForm && (
        <ContactFormModal
          contact={editingContact}
          companies={companies}
          onClose={() => { setShowForm(false); setEditingContact(null) }}
          onSaved={() => { setShowForm(false); setEditingContact(null); fetchContacts() }}
        />
      )}
    </div>
  )
}

function ContactFormModal({
  contact,
  companies,
  onClose,
  onSaved,
}: {
  contact: Contact | null
  companies: CustomerCompany[]
  onClose: () => void
  onSaved: () => void
}) {
  const [form, setForm] = useState({
    name: contact?.name || '',
    phone: contact?.phone || '',
    email: contact?.email || '',
    customer_company_id: contact?.customer_company_id || '',
    company_name: contact?.company_name || '',
    position: contact?.position || '',
    city: contact?.city || '',
    origin: contact?.origin || '',
  })
  const [saving, setSaving] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)

    try {
      if (contact) {
        await api.put(`/contacts/${contact.id}`, form)
        toast.success('Contato atualizado')
      } else {
        await api.post('/contacts', form)
        toast.success('Contato criado')
      }
      onSaved()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao salvar')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          {contact ? 'Editar contato' : 'Novo contato'}
        </h3>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nome</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="input"
                placeholder="João Silva"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Telefone</label>
              <input
                type="text"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
                className="input"
                placeholder="+55 11 99999-9999"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">E-mail</label>
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="input"
              placeholder="joao@empresa.com"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Empresa vinculada</label>
              <select
                value={form.customer_company_id}
                onChange={(e) => {
                  const selected = companies.find((company) => company.id === e.target.value)
                  setForm({ ...form, customer_company_id: e.target.value, company_name: selected?.name || form.company_name })
                }}
                className="input"
              >
                <option value="">Sem empresa</option>
                {companies.map((company) => (
                  <option key={company.id} value={company.id}>{company.name}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Cargo</label>
              <input
                type="text"
                value={form.position}
                onChange={(e) => setForm({ ...form, position: e.target.value })}
                className="input"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Cidade</label>
              <input
                type="text"
                value={form.city}
                onChange={(e) => setForm({ ...form, city: e.target.value })}
                className="input"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Origem</label>
              <input
                type="text"
                value={form.origin}
                onChange={(e) => setForm({ ...form, origin: e.target.value })}
                className="input"
                placeholder="WhatsApp, site, indicação..."
              />
            </div>
          </div>

          <div className="flex gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn-secondary flex-1">
              Cancelar
            </button>
            <button type="submit" disabled={saving} className="btn-primary flex-1">
              {saving ? 'Salvando...' : 'Salvar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
