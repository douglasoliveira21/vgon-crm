'use client'

import { useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { ArrowLeft, CheckCircle2, Loader2, Mail, Search, Send, Users, XCircle } from 'lucide-react'
import { ChannelIcon } from '@/components/channel-icon'

interface EmailChannel {
  id: string
  name: string
  type: string
  status: string
  settings?: {
    username?: string
    email_address?: string
    provider?: string
  }
}

interface EmailContact {
  id: string
  name?: string
  email?: string
  phone?: string
  company_name?: string
}

interface SendResult {
  total: number
  sent_count: number
  queued_count?: number
  failed_count: number
  failures?: Array<{ contact_id: string; name: string; email: string; error: string }>
}

export default function EmailCampaignPage() {
  const [channels, setChannels] = useState<EmailChannel[]>([])
  const [contacts, setContacts] = useState<EmailContact[]>([])
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [targetType, setTargetType] = useState<'all' | 'selected'>('all')
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [search, setSearch] = useState('')
  const [result, setResult] = useState<SendResult | null>(null)
  const [form, setForm] = useState({
    channel_id: '',
    subject: '',
    content: '',
  })

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [channelsResponse, contactsResponse] = await Promise.all([
        api.get('/channels'),
        loadAllContacts(),
      ])
      const emailChannels = (channelsResponse.data.channels || []).filter(
        (channel: EmailChannel) => channel.type === 'email' && channel.status === 'connected'
      )
      setChannels(emailChannels)
      setContacts(contactsResponse)
      if (emailChannels.length > 0) {
        setForm((current) => ({ ...current, channel_id: current.channel_id || emailChannels[0].id }))
      }
    } catch (error) {
      console.error('Error:', error)
      toast.error('Erro ao carregar campanha por e-mail')
    } finally {
      setLoading(false)
    }
  }

  const loadAllContacts = async () => {
    const pageSize = 500
    let offset = 0
    let total = 0
    const allContacts: EmailContact[] = []

    do {
      const response = await api.get('/contacts', { params: { limit: pageSize, offset } })
      const pageContacts = response.data.contacts || []
      total = response.data.total || pageContacts.length
      allContacts.push(
        ...pageContacts
          .filter((contact: EmailContact) => contact.email?.trim())
          .map((contact: EmailContact) => ({
            ...contact,
            name: contact.name || contact.email,
            email: contact.email?.trim(),
          }))
      )
      offset += pageSize
    } while (offset < total)

    return allContacts.sort((a, b) => (a.name || a.email || '').localeCompare(b.name || b.email || ''))
  }

  const filteredContacts = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (!query) return contacts
    return contacts.filter((contact) =>
      contact.name?.toLowerCase().includes(query) ||
      contact.email?.toLowerCase().includes(query) ||
      contact.company_name?.toLowerCase().includes(query)
    )
  }, [contacts, search])

  const selectedContacts = useMemo(
    () => contacts.filter((contact) => selectedIds.includes(contact.id)),
    [contacts, selectedIds]
  )

  const totalRecipients = targetType === 'all' ? contacts.length : selectedContacts.length

  const toggleContact = (id: string) => {
    setSelectedIds((current) =>
      current.includes(id) ? current.filter((contactID) => contactID !== id) : [...current, id]
    )
  }

  const selectFilteredContacts = () => {
    setSelectedIds((current) => Array.from(new Set([...current, ...filteredContacts.map((contact) => contact.id)])))
  }

  const clearSelection = () => {
    setSelectedIds([])
  }

  const sendCampaign = async () => {
    if (!form.channel_id) {
      toast.error('Selecione uma caixa de entrada')
      return
    }
    if (!form.subject.trim()) {
      toast.error('Informe o assunto')
      return
    }
    if (!form.content.trim()) {
      toast.error('Escreva a mensagem')
      return
    }
    if (totalRecipients === 0) {
      toast.error('Nenhum contato com e-mail para enviar')
      return
    }
    if (!confirm(`Enviar e-mail para ${totalRecipients} contato(s)?`)) return

    setSending(true)
    setResult(null)
    try {
      const response = await api.post('/campaigns/email/send', {
        channel_id: form.channel_id,
        subject: form.subject,
        content: form.content,
        send_to_all: targetType === 'all',
        contact_ids: targetType === 'selected' ? selectedIds : undefined,
      })
      setResult(response.data)
      toast.success(`${response.data.queued_count || 0} e-mails adicionados à fila de envio`)
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao enviar campanha')
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="mx-auto max-w-6xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <Link href="/campaigns" className="inline-flex items-center gap-2 text-sm text-gray-500 hover:text-gray-900 mb-3">
            <ArrowLeft size={16} />
            Voltar para Marketing
          </Link>
          <h1 className="text-2xl font-bold text-gray-900">Campanha por e-mail</h1>
          <p className="text-gray-500 mt-1">Dispare e-mails usando uma caixa de entrada conectada.</p>
        </div>
        <button onClick={sendCampaign} disabled={sending || loading} className="btn-primary">
          {sending ? <Loader2 size={18} className="animate-spin" /> : <Send size={18} />}
          {sending ? 'Enviando...' : 'Enviar campanha'}
        </button>
      </div>

      {loading ? (
        <div className="card p-10 text-center text-gray-500">
          <Loader2 size={28} className="animate-spin mx-auto mb-3" />
          Carregando dados da campanha...
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6">
          <div className="space-y-6">
            <div className="card p-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Caixa de entrada remetente</label>
                <select
                  value={form.channel_id}
                  onChange={(event) => setForm({ ...form, channel_id: event.target.value })}
                  className="input"
                >
                  {channels.length === 0 && <option value="">Nenhuma caixa de e-mail conectada</option>}
                  {channels.map((channel) => (
                    <option key={channel.id} value={channel.id}>
                      {channel.name} {channel.settings?.email_address || channel.settings?.username ? `- ${channel.settings?.email_address || channel.settings?.username}` : ''}
                    </option>
                  ))}
                </select>
                {channels.length === 0 && (
                  <p className="text-xs text-red-500 mt-1">Cadastre uma caixa em Canais antes de enviar e-mail marketing.</p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Assunto</label>
                <input
                  type="text"
                  value={form.subject}
                  onChange={(event) => setForm({ ...form, subject: event.target.value })}
                  className="input"
                  placeholder="Ex: Olá {{nome}}, temos uma novidade"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Mensagem</label>
                <textarea
                  value={form.content}
                  onChange={(event) => setForm({ ...form, content: event.target.value })}
                  className="input resize-none"
                  rows={10}
                  placeholder={'Olá {{nome}},\n\nEscreva aqui o conteúdo da campanha.\n\nAtenciosamente,'}
                />
                <p className="text-xs text-gray-400 mt-1">Variáveis: {'{{nome}}'}, {'{{email}}'}, {'{{telefone}}'}, {'{{empresa}}'}</p>
              </div>
            </div>

            <div className="card p-5">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h2 className="font-semibold text-gray-900">Destinatários</h2>
                  <p className="text-sm text-gray-500">{contacts.length} contatos com e-mail salvo</p>
                </div>
                <Users size={22} className="text-gray-400" />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
                <label className="flex items-center gap-3 rounded-lg border border-gray-200 p-3 cursor-pointer hover:bg-gray-50">
                  <input type="radio" checked={targetType === 'all'} onChange={() => setTargetType('all')} />
                  <div>
                    <p className="text-sm font-medium text-gray-900">Todos com e-mail</p>
                    <p className="text-xs text-gray-400">{contacts.length} contatos elegíveis</p>
                  </div>
                </label>
                <label className="flex items-center gap-3 rounded-lg border border-gray-200 p-3 cursor-pointer hover:bg-gray-50">
                  <input type="radio" checked={targetType === 'selected'} onChange={() => setTargetType('selected')} />
                  <div>
                    <p className="text-sm font-medium text-gray-900">Selecionar manualmente</p>
                    <p className="text-xs text-gray-400">{selectedContacts.length} selecionados</p>
                  </div>
                </label>
              </div>

              {targetType === 'selected' && (
                <div className="space-y-3">
                  <div className="relative">
                    <Search size={16} className="absolute left-3 top-3 text-gray-400" />
                    <input
                      type="text"
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                      className="input pl-9"
                      placeholder="Buscar por nome, e-mail ou empresa..."
                    />
                  </div>
                  <div className="flex items-center gap-3">
                    <button type="button" onClick={selectFilteredContacts} className="text-xs text-primary-600 hover:underline">
                      Selecionar lista filtrada ({filteredContacts.length})
                    </button>
                    {selectedIds.length > 0 && (
                      <button type="button" onClick={clearSelection} className="text-xs text-red-500 hover:underline">
                        Limpar seleção
                      </button>
                    )}
                  </div>
                  <div className="max-h-80 overflow-y-auto rounded-lg border border-gray-100 divide-y divide-gray-100">
                    {filteredContacts.slice(0, 200).map((contact) => (
                      <label key={contact.id} className="flex items-center gap-3 p-3 hover:bg-gray-50 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={selectedIds.includes(contact.id)}
                          onChange={() => toggleContact(contact.id)}
                        />
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-gray-900 truncate">{contact.name || contact.email}</p>
                          <p className="text-xs text-gray-500 truncate">{contact.email}</p>
                        </div>
                      </label>
                    ))}
                    {filteredContacts.length === 0 && (
                      <div className="p-6 text-center text-sm text-gray-500">Nenhum contato encontrado.</div>
                    )}
                    {filteredContacts.length > 200 && (
                      <div className="p-3 text-center text-xs text-gray-400">Mostrando 200 contatos. Use a busca para refinar.</div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="space-y-4">
            <div className="card p-5">
              <div className="flex items-center gap-2 mb-3">
                <ChannelIcon type="email" size={18} />
                <h2 className="font-semibold text-gray-900">Resumo do envio</h2>
              </div>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-500">Caixas conectadas</span>
                  <span className="font-medium text-gray-900">{channels.length}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Contatos com e-mail</span>
                  <span className="font-medium text-gray-900">{contacts.length}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Destinatários</span>
                  <span className="font-medium text-primary-600">{totalRecipients}</span>
                </div>
              </div>
              <div className="mt-4 rounded-lg bg-yellow-50 p-3 text-xs text-yellow-700">
                Envie apenas para contatos que autorizaram receber comunicação por e-mail.
              </div>
            </div>

            {result && (
              <div className="card p-5">
                <h2 className="font-semibold text-gray-900 mb-3">Resultado</h2>
                <div className="space-y-2 text-sm">
                  <p className="flex items-center gap-2 text-green-700">
                    <CheckCircle2 size={16} /> {result.queued_count || result.sent_count} enfileirados
                  </p>
                  <p className="flex items-center gap-2 text-red-600">
                    <XCircle size={16} /> {result.failed_count} falhas
                  </p>
                </div>
                {result.failures && result.failures.length > 0 && (
                  <div className="mt-4 max-h-64 overflow-y-auto space-y-2">
                    {result.failures.slice(0, 20).map((failure) => (
                      <div key={`${failure.contact_id}-${failure.email}`} className="rounded-lg bg-red-50 p-2 text-xs text-red-700">
                        <p className="font-medium">{failure.name || failure.email}</p>
                        <p>{failure.error}</p>
                      </div>
                    ))}
                    {result.failures.length > 20 && (
                      <p className="text-xs text-gray-400">+ {result.failures.length - 20} falhas ocultas</p>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
