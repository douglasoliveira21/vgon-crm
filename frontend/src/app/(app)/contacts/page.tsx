'use client'

import { useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Search, Plus, Edit2, Trash2, MapPin, MessageSquare, X, ShieldCheck, Download, History } from 'lucide-react'

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
  is_opted_out?: boolean
  opted_out_at?: string
  opt_out_reason?: string
  opt_out_source?: string
  consent_status?: string
  consent_source?: string
  consent_text?: string
  consent_given_at?: string
  consent_revoked_at?: string
  tags?: Array<{ id: string; name: string; color: string }>
  created_at: string
}

interface CustomerCompany {
  id: string
  name: string
  cnpj?: string
}

interface Channel {
  id: string
  name: string
  type: string
  status: string
  is_active: boolean
}

export default function ContactsPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [contacts, setContacts] = useState<Contact[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [showForm, setShowForm] = useState(false)
  const [editingContact, setEditingContact] = useState<Contact | null>(null)
  const [conversationContact, setConversationContact] = useState<Contact | null>(null)
  const [privacyContact, setPrivacyContact] = useState<Contact | null>(null)
  const [channels, setChannels] = useState<Channel[]>([])
  const [loadingChannels, setLoadingChannels] = useState(false)
  const [startingConversation, setStartingConversation] = useState(false)
  const [companies, setCompanies] = useState<CustomerCompany[]>([])
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  useEffect(() => {
    fetchContacts()
  }, [search, page])

  useEffect(() => {
    const urlSearch = searchParams.get('search') || ''
    if (urlSearch) setSearch(urlSearch)
    fetchCompanies()
    fetchChannels()
  }, [searchParams])

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

  const fetchChannels = async () => {
    setLoadingChannels(true)
    try {
      const response = await api.get('/channels')
      const availableChannels = (response.data.channels || []).filter((channel: Channel) =>
        channel.type === 'whatsapp' && channel.status === 'connected' && channel.is_active
      )
      setChannels(availableChannels)
    } catch {
      setChannels([])
    } finally {
      setLoadingChannels(false)
    }
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
    setConversationContact(contact)
  }

  const confirmStartConversation = async (channelId: string) => {
    if (!conversationContact?.phone) return

    setStartingConversation(true)
    try {
      const response = await api.post('/conversations/start', {
        phone: conversationContact.phone,
        channel_id: channelId,
      })
      const conversationID = response.data.conversation_id
      toast.success('Conversa iniciada')
      setConversationContact(null)
      router.push(conversationID ? `/conversations?conversation=${conversationID}` : '/conversations')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao iniciar conversa')
    } finally {
      setStartingConversation(false)
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
                      title="Editar contato"
                    >
                      <Edit2 size={14} />
                    </button>
                    <button
                      onClick={() => setPrivacyContact(contact)}
                      className="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded"
                      title="LGPD e auditoria"
                    >
                      <ShieldCheck size={14} />
                    </button>
                    <button
                      onClick={() => deleteContact(contact.id)}
                      className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                      title="Remover contato"
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

      {conversationContact && (
        <StartConversationModal
          contact={conversationContact}
          channels={channels}
          loadingChannels={loadingChannels}
          starting={startingConversation}
          onRefreshChannels={fetchChannels}
          onClose={() => setConversationContact(null)}
          onStart={confirmStartConversation}
        />
      )}

      {privacyContact && (
        <ContactPrivacyModal
          contact={privacyContact}
          onClose={() => setPrivacyContact(null)}
          onUpdated={() => {
            fetchContacts()
          }}
        />
      )}
    </div>
  )
}

function getConsentLabel(status?: string) {
  const labels: Record<string, string> = {
    granted: 'Autorizado',
    revoked: 'Revogado',
    opted_out: 'Opt-out',
    unknown: 'Não informado',
  }
  return labels[status || 'unknown'] || 'Não informado'
}

function ContactPrivacyModal({
  contact,
  onClose,
  onUpdated,
}: {
  contact: Contact
  onClose: () => void
  onUpdated: () => void
}) {
  const [status, setStatus] = useState(contact.consent_status || (contact.is_opted_out ? 'opted_out' : 'unknown'))
  const [source, setSource] = useState(contact.consent_source || contact.opt_out_source || 'manual')
  const [reason, setReason] = useState(contact.opt_out_reason || '')
  const [consentText, setConsentText] = useState(contact.consent_text || '')
  const [saving, setSaving] = useState(false)
  const [loadingAudit, setLoadingAudit] = useState(true)
  const [consents, setConsents] = useState<Array<any>>([])
  const [audit, setAudit] = useState<Array<any>>([])

  useEffect(() => {
    loadAudit()
  }, [])

  const loadAudit = async () => {
    setLoadingAudit(true)
    try {
      const response = await api.get(`/contacts/${contact.id}/audit`)
      setConsents(response.data.consents || [])
      setAudit(response.data.audit || [])
    } catch {
      toast.error('Erro ao carregar histórico LGPD')
    } finally {
      setLoadingAudit(false)
    }
  }

  const saveConsent = async (nextStatus = status) => {
    setSaving(true)
    try {
      await api.post(`/contacts/${contact.id}/consent`, {
        status: nextStatus,
        source,
        reason,
        consent_text: consentText,
      })
      setStatus(nextStatus)
      toast.success('Consentimento atualizado')
      await loadAudit()
      onUpdated()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao atualizar consentimento')
    } finally {
      setSaving(false)
    }
  }

  const exportContact = async () => {
    try {
      const response = await api.get(`/contacts/${contact.id}/export`, { responseType: 'blob' })
      const url = window.URL.createObjectURL(new Blob([response.data], { type: 'application/json' }))
      const link = document.createElement('a')
      link.href = url
      link.download = `contato-${contact.id}-lgpd.json`
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)
      toast.success('Exportação gerada')
      await loadAudit()
    } catch {
      toast.error('Erro ao exportar dados do contato')
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-3xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-4 mb-5">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">LGPD e auditoria</h3>
            <p className="text-sm text-gray-500 mt-1">{contact.name || contact.email || contact.phone}</p>
          </div>
          <button type="button" onClick={onClose} className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg">
            <X size={18} />
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-5">
          <div className="space-y-4">
            <div className="rounded-xl border border-gray-200 p-4">
              <div className="flex items-center gap-2 mb-3">
                <ShieldCheck size={18} className="text-primary-600" />
                <h4 className="font-medium text-gray-900">Consentimento e opt-out</h4>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                  <select value={status} onChange={(e) => setStatus(e.target.value)} className="input">
                    <option value="unknown">Não informado</option>
                    <option value="granted">Autorizado</option>
                    <option value="revoked">Revogado</option>
                    <option value="opted_out">Opt-out</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Origem</label>
                  <input value={source} onChange={(e) => setSource(e.target.value)} className="input" placeholder="manual, site, whatsapp..." />
                </div>
              </div>
              <div className="mt-3">
                <label className="block text-sm font-medium text-gray-700 mb-1">Texto/base da autorização</label>
                <textarea value={consentText} onChange={(e) => setConsentText(e.target.value)} className="input resize-none" rows={3} />
              </div>
              {(status === 'revoked' || status === 'opted_out') && (
                <div className="mt-3">
                  <label className="block text-sm font-medium text-gray-700 mb-1">Motivo</label>
                  <input value={reason} onChange={(e) => setReason(e.target.value)} className="input" placeholder="Solicitado pelo cliente, descadastro..." />
                </div>
              )}
              <div className="flex flex-wrap gap-2 mt-4">
                <button type="button" disabled={saving} onClick={() => saveConsent()} className="btn-primary">
                  {saving ? 'Salvando...' : 'Salvar status'}
                </button>
                <button type="button" disabled={saving} onClick={() => saveConsent('granted')} className="btn-secondary">
                  Marcar autorizado
                </button>
                <button type="button" disabled={saving} onClick={() => saveConsent('opted_out')} className="btn-secondary text-red-600">
                  Registrar opt-out
                </button>
              </div>
            </div>

            <div className="rounded-xl border border-gray-200 p-4">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <History size={18} className="text-gray-500" />
                  <h4 className="font-medium text-gray-900">Histórico</h4>
                </div>
                <button type="button" onClick={loadAudit} className="text-xs text-primary-600 hover:underline">Atualizar</button>
              </div>
              {loadingAudit ? (
                <p className="text-sm text-gray-500">Carregando histórico...</p>
              ) : (
                <div className="space-y-3">
                  {[...consents, ...audit].slice(0, 30).map((item, index) => (
                    <div key={`${item.id}-${index}`} className="rounded-lg bg-gray-50 p-3 text-sm">
                      <p className="font-medium text-gray-900">{item.action || `Consentimento: ${getConsentLabel(item.status)}`}</p>
                      <p className="text-xs text-gray-500">
                        {item.user_name || 'Sistema'} · {item.created_at ? new Date(item.created_at).toLocaleString('pt-BR') : ''}
                      </p>
                      {(item.reason || item.source) && (
                        <p className="text-xs text-gray-500 mt-1">{item.source} {item.reason ? `- ${item.reason}` : ''}</p>
                      )}
                    </div>
                  ))}
                  {consents.length === 0 && audit.length === 0 && (
                    <p className="text-sm text-gray-500">Nenhum histórico registrado.</p>
                  )}
                </div>
              )}
            </div>
          </div>

          <div className="space-y-4">
            <div className="rounded-xl border border-gray-200 p-4">
              <p className="text-xs text-gray-500">Status atual</p>
              <p className="text-lg font-semibold text-gray-900">{getConsentLabel(status)}</p>
              <p className="text-xs text-gray-500 mt-2">
                Opt-out impede campanhas e registra a solicitação do titular.
              </p>
            </div>
            <button type="button" onClick={exportContact} className="btn-secondary w-full justify-center">
              <Download size={16} />
              Exportar dados LGPD
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function StartConversationModal({
  contact,
  channels,
  loadingChannels,
  starting,
  onRefreshChannels,
  onClose,
  onStart,
}: {
  contact: Contact
  channels: Channel[]
  loadingChannels: boolean
  starting: boolean
  onRefreshChannels: () => void
  onClose: () => void
  onStart: (channelId: string) => void
}) {
  const [selectedChannelId, setSelectedChannelId] = useState(channels[0]?.id || '')

  useEffect(() => {
    if (!selectedChannelId && channels[0]?.id) {
      setSelectedChannelId(channels[0].id)
    }
  }, [channels, selectedChannelId])

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-md">
        <div className="flex items-start justify-between gap-4 mb-5">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Iniciar conversa</h3>
            <p className="text-sm text-gray-500 mt-1">
              Escolha por qual canal chamar {contact.name || contact.phone}.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
            aria-label="Fechar"
          >
            <X size={18} />
          </button>
        </div>

        {loadingChannels ? (
          <div className="py-8 text-center text-sm text-gray-500">Carregando canais...</div>
        ) : channels.length === 0 ? (
          <div className="space-y-4">
            <div className="rounded-xl border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-800">
              Nenhum canal de WhatsApp conectado foi encontrado.
            </div>
            <div className="flex gap-3">
              <button type="button" onClick={onClose} className="btn-secondary flex-1">
                Cancelar
              </button>
              <button type="button" onClick={onRefreshChannels} className="btn-primary flex-1">
                Atualizar canais
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              {channels.map((channel) => (
                <label
                  key={channel.id}
                  className={`flex cursor-pointer items-center gap-3 rounded-xl border p-3 transition-colors ${
                    selectedChannelId === channel.id ? 'border-primary-500 bg-primary-50' : 'border-gray-200 hover:bg-gray-50'
                  }`}
                >
                  <input
                    type="radio"
                    name="channel"
                    value={channel.id}
                    checked={selectedChannelId === channel.id}
                    onChange={() => setSelectedChannelId(channel.id)}
                    className="h-4 w-4 text-primary-600"
                  />
                  <div>
                    <p className="text-sm font-medium text-gray-900">{channel.name}</p>
                    <p className="text-xs text-gray-500">WhatsApp conectado</p>
                  </div>
                </label>
              ))}
            </div>

            <div className="flex gap-3 pt-2">
              <button type="button" onClick={onClose} className="btn-secondary flex-1">
                Cancelar
              </button>
              <button
                type="button"
                disabled={!selectedChannelId || starting}
                onClick={() => onStart(selectedChannelId)}
                className="btn-primary flex-1 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {starting ? 'Iniciando...' : 'Iniciar conversa'}
              </button>
            </div>
          </div>
        )}
      </div>
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
    consent_status: contact?.consent_status || (contact?.is_opted_out ? 'opted_out' : 'unknown'),
    consent_source: contact?.consent_source || '',
    consent_text: contact?.consent_text || '',
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

          <div className="rounded-xl border border-gray-200 p-4">
            <div className="flex items-center gap-2 mb-3">
              <ShieldCheck size={16} className="text-primary-600" />
              <p className="text-sm font-medium text-gray-900">LGPD e consentimento</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                <select
                  value={form.consent_status}
                  onChange={(e) => setForm({ ...form, consent_status: e.target.value })}
                  className="input"
                >
                  <option value="unknown">Não informado</option>
                  <option value="granted">Autorizado</option>
                  <option value="revoked">Revogado</option>
                  <option value="opted_out">Opt-out</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Origem da autorização</label>
                <input
                  type="text"
                  value={form.consent_source}
                  onChange={(e) => setForm({ ...form, consent_source: e.target.value })}
                  className="input"
                  placeholder="site, whatsapp, contrato..."
                />
              </div>
            </div>
            <div className="mt-3">
              <label className="block text-sm font-medium text-gray-700 mb-1">Texto/base da autorização</label>
              <textarea
                value={form.consent_text}
                onChange={(e) => setForm({ ...form, consent_text: e.target.value })}
                className="input resize-none"
                rows={2}
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
