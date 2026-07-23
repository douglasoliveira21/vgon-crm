'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/auth'
import { Plus, Send, Pause, Play, Users, Check, Eye, X, Trash2, AlertTriangle, FileText, Image, Video, Music, ArrowUp, ArrowDown, Mail, CalendarClock, ShieldCheck } from 'lucide-react'
import { ChannelIcon } from '@/components/channel-icon'

type CampaignContentType = 'text' | 'image' | 'video' | 'audio'

interface CampaignContentItem {
  client_id?: string
  type: CampaignContentType
  content: string
  media_url?: string
  media_base64?: string
  media_filename?: string
}

interface Campaign {
  id: string
  name: string
  status: string
  message_content?: string
  message_type: string
  media_url?: string
  content_items?: CampaignContentItem[]
  send_speed?: number
  total_contacts: number
  sent_count: number
  delivered_count: number
  read_count: number
  replied_count: number
  failed_count: number
  scheduled_at?: string
  timezone?: string
  approval_status?: string
  frequency_cap_days?: number
  created_at: string
}

export default function CampaignsPage() {
  const { user } = useAuthStore()
  const [campaigns, setCampaigns] = useState<Campaign[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [editingCampaign, setEditingCampaign] = useState<Campaign | null>(null)

  useEffect(() => {
    fetchCampaigns()
  }, [])

  useEffect(() => {
    if (!campaigns.some((campaign) => campaign.status === 'sending')) return
    const interval = setInterval(fetchCampaigns, 5000)
    return () => clearInterval(interval)
  }, [campaigns])

  const fetchCampaigns = async () => {
    try {
      const response = await api.get('/campaigns')
      setCampaigns(response.data.campaigns || [])
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const startCampaign = async (id: string) => {
    try {
      const response = await api.post(`/campaigns/${id}/start`)
      const nextStatus = response.data?.status || 'sending'
      setCampaigns((prev) =>
        prev.map((c) => (c.id === id ? { ...c, status: nextStatus } : c))
      )
      toast.success(nextStatus === 'completed' ? 'Campanha concluída' : 'Campanha iniciada')
      fetchCampaigns()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao iniciar campanha')
    }
  }

  const deleteCampaign = async (id: string) => {
    if (!confirm('Excluir esta campanha?')) return
    try {
      await api.delete(`/campaigns/${id}`)
      toast.success('Campanha excluída')
      fetchCampaigns()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao excluir campanha')
    }
  }

  const pauseCampaign = async (id: string) => {
    try {
      await api.post(`/campaigns/${id}/pause`)
      setCampaigns((prev) =>
        prev.map((c) => (c.id === id ? { ...c, status: 'paused' } : c))
      )
      toast.success('Campanha pausada')
    } catch {
      toast.error('Erro ao pausar')
    }
  }

  const approveCampaign = async (campaign: Campaign) => {
    try {
      await api.post(`/campaigns/${campaign.id}/approve`)
      if (campaign.scheduled_at && new Date(campaign.scheduled_at).getTime() > Date.now()) {
        await api.post(`/campaigns/${campaign.id}/schedule`, {
          scheduled_at: campaign.scheduled_at,
          timezone: campaign.timezone || 'America/Sao_Paulo',
        })
        toast.success('Campanha aprovada e agendada')
      } else {
        toast.success('Campanha aprovada para envio')
      }
      fetchCampaigns()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao aprovar campanha')
    }
  }

  const getStatusBadge = (status: string) => {
    const map: Record<string, { label: string; class: string }> = {
      draft: { label: 'Rascunho', class: 'badge-gray' },
      scheduled: { label: 'Agendada', class: 'badge-blue' },
      sending: { label: 'Enviando', class: 'badge-yellow' },
      completed: { label: 'Concluída', class: 'badge-green' },
      paused: { label: 'Pausada', class: 'badge-red' },
    }
    const s = map[status] || map.draft
    return <span className={`badge ${s.class}`}>{s.label}</span>
  }

  return (
    <div className="mx-auto max-w-6xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Marketing em Massa</h1>
          <p className="text-gray-500 mt-1">Envie campanhas para seus contatos via WhatsApp</p>
        </div>
        <div className="flex items-center gap-2">
          <Link href="/campaigns/email" className="btn-secondary">
            <ChannelIcon type="email" size={18} />
            Campanha por e-mail
          </Link>
          <button onClick={() => { setEditingCampaign(null); setShowForm(true) }} className="btn-primary">
            <Plus size={18} />
            Nova campanha
          </button>
        </div>
      </div>

      {/* Stats */}
      {campaigns.length > 0 && (
      <div className="mb-6 grid grid-cols-1 gap-3 min-[420px]:grid-cols-2 lg:grid-cols-4 lg:gap-4">
          <div className="card p-4">
            <p className="text-xs text-gray-400">Total enviadas</p>
            <p className="text-xl font-bold text-gray-900">{campaigns.reduce((s, c) => s + c.sent_count, 0)}</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-gray-400">Entregues</p>
            <p className="text-xl font-bold text-green-600">{campaigns.reduce((s, c) => s + c.delivered_count, 0)}</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-gray-400">Lidas</p>
            <p className="text-xl font-bold text-blue-600">{campaigns.reduce((s, c) => s + c.read_count, 0)}</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-gray-400">Respondidas</p>
            <p className="text-xl font-bold text-purple-600">{campaigns.reduce((s, c) => s + c.replied_count, 0)}</p>
          </div>
        </div>
      )}

      {/* Campaigns List */}
      <div className="space-y-4">
        {campaigns.map((campaign) => {
          const processedCount = campaign.sent_count + campaign.failed_count
          const progress = campaign.total_contacts > 0
            ? Math.min(100, Math.round((processedCount / campaign.total_contacts) * 100))
            : 0

          return (
          <div key={campaign.id} className="card p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3">
                  <h3 className="font-semibold text-gray-900">{campaign.name}</h3>
                  {getStatusBadge(campaign.status)}
                  <span className={`badge ${campaign.approval_status === 'approved' ? 'badge-green' : 'badge-yellow'}`}>
                    {campaign.approval_status === 'approved' ? 'Aprovada' : 'Aguardando aprovação'}
                  </span>
                </div>
                <p className="text-sm text-gray-500 mt-1 flex items-center gap-4">
                  <span className="flex items-center gap-1"><Users size={14} /> {campaign.total_contacts} contatos</span>
                  <span className="flex items-center gap-1"><FileText size={14} /> {campaign.content_items?.length || 1} conteúdos</span>
                  <span className="flex items-center gap-1"><Send size={14} /> {campaign.sent_count} enviadas</span>
                  <span className="flex items-center gap-1"><Check size={14} /> {campaign.delivered_count} entregues</span>
                  {campaign.scheduled_at && <span className="flex items-center gap-1"><CalendarClock size={14} /> {new Date(campaign.scheduled_at).toLocaleString('pt-BR')}</span>}
                  <span className="flex items-center gap-1"><Eye size={14} /> {campaign.read_count} lidas</span>
                  {campaign.failed_count > 0 && (
                    <span className="flex items-center gap-1 text-red-600"><AlertTriangle size={14} /> {campaign.failed_count} falhas</span>
                  )}
                </p>
              </div>

              <div className="flex items-center gap-2">
                {(campaign.status === 'draft' || campaign.status === 'paused') && campaign.approval_status !== 'approved' && user?.role_slug !== 'marketing' && (
                  <button onClick={() => approveCampaign(campaign)} className="btn-primary text-sm py-2">
                    <ShieldCheck size={14} /> {campaign.scheduled_at ? 'Aprovar e agendar' : 'Aprovar'}
                  </button>
                )}
                {(campaign.status === 'draft' || campaign.status === 'paused') && campaign.approval_status !== 'approved' && user?.role_slug === 'marketing' && (
                  <span className="text-xs text-amber-600">Aguardando aprovação do administrador</span>
                )}
                {campaign.status === 'draft' && campaign.approval_status === 'approved' && (
                  <button onClick={() => startCampaign(campaign.id)} className="btn-primary text-sm py-2">
                    <Play size={14} /> Iniciar
                  </button>
                )}
                {campaign.status === 'sending' && (
                  <button onClick={() => pauseCampaign(campaign.id)} className="btn-secondary text-sm py-2">
                    <Pause size={14} /> Pausar
                  </button>
                )}
                {campaign.status === 'paused' && campaign.approval_status === 'approved' && (
                  <button onClick={() => startCampaign(campaign.id)} className="btn-primary text-sm py-2">
                    <Play size={14} /> Retomar
                  </button>
                )}
                {(campaign.status === 'draft' || campaign.status === 'paused') && (
                  <button onClick={() => { setEditingCampaign(campaign); setShowForm(true) }} className="btn-secondary text-sm py-2">
                    Editar
                  </button>
                )}
                {campaign.status !== 'sending' && (
                  <button onClick={() => deleteCampaign(campaign.id)} className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg" title="Excluir">
                    <Trash2 size={16} />
                  </button>
                )}
              </div>
            </div>

            {/* Progress Bar */}
            {campaign.total_contacts > 0 && campaign.status !== 'draft' && (
              <div className="mt-4">
                <div className="w-full bg-gray-100 rounded-full h-2">
                  <div
                    className="bg-primary-600 h-2 rounded-full transition-all"
                    style={{ width: `${progress}%` }}
                  />
                </div>
                <p className="text-xs text-gray-400 mt-1">
                  {progress}% processado
                </p>
              </div>
            )}
          </div>
          )
        })}

        {campaigns.length === 0 && !loading && (
          <div className="card p-12 text-center">
            <Send size={40} className="text-gray-300 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">Nenhuma campanha criada</h3>
            <p className="text-gray-500 text-sm mb-6 max-w-sm mx-auto">
              Crie campanhas de marketing para enviar mensagens em massa para seus contatos via WhatsApp.
            </p>
            <button onClick={() => setShowForm(true)} className="btn-primary inline-flex">
              <Plus size={18} /> Criar primeira campanha
            </button>
          </div>
        )}
      </div>

      {/* Create Campaign Modal */}
      {showForm && (
        <CreateCampaignModal
          campaign={editingCampaign}
          onClose={() => { setShowForm(false); setEditingCampaign(null) }}
          onCreated={() => { setShowForm(false); setEditingCampaign(null); fetchCampaigns() }}
        />
      )}
    </div>
  )
}

function CreateCampaignModal({ campaign, onClose, onCreated }: { campaign: Campaign | null; onClose: () => void; onCreated: () => void }) {
  const isEditing = Boolean(campaign)
  const [name, setName] = useState(campaign?.name || '')
  const initialItems: CampaignContentItem[] = campaign?.content_items?.length
    ? campaign.content_items
    : [{ type: (campaign?.message_type as CampaignContentType) || 'text', content: campaign?.message_content || '', media_url: campaign?.media_url }]
  const [contentItems, setContentItems] = useState<CampaignContentItem[]>(
    initialItems.map((item, index) => ({ ...item, client_id: `${Date.now()}-${index}` }))
  )
  const [sendSpeed, setSendSpeed] = useState(campaign?.send_speed || 30)
  const [frequencyCapDays, setFrequencyCapDays] = useState(campaign?.frequency_cap_days || 0)
  const [scheduledAt, setScheduledAt] = useState(
    campaign?.scheduled_at ? new Date(campaign.scheduled_at).toISOString().slice(0, 16) : ''
  )
  const [targetType, setTargetType] = useState('all') // all, tag, selected
  const [filterTag, setFilterTag] = useState('')
  const [contactSearch, setContactSearch] = useState('')
  const [allContacts, setAllContacts] = useState<Array<{id: string; name: string; phone: string}>>([])
  const [selectedContacts, setSelectedContacts] = useState<Array<{id: string; name: string; phone: string}>>([])
  const [searchResults, setSearchResults] = useState<Array<{id: string; name: string; phone: string}>>([])
  const [tags, setTags] = useState<Array<{id: string; name: string; color: string}>>([])
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.get('/tags').then(res => setTags(res.data.tags || [])).catch(() => {})
    loadAllContacts()
  }, [])

  const loadAllContacts = async () => {
    const pageSize = 500
    let offset = 0
    let total = 0
    const contacts: Array<{id: string; name: string; phone: string}> = []

    try {
      do {
        const response = await api.get('/contacts', { params: { limit: pageSize, offset } })
        const pageContacts = response.data.contacts || []
        total = response.data.total || pageContacts.length
        contacts.push(
          ...pageContacts
            .filter((c: any) => c.phone)
            .map((c: any) => ({ id: c.id, name: c.name || c.phone, phone: c.phone }))
        )
        offset += pageSize
      } while (offset < total)

      setAllContacts(contacts)
    } catch {
      setAllContacts(contacts)
    }
  }

  const searchContact = (query: string) => {
    setContactSearch(query)
    if (query.length < 2) { setSearchResults([]); return }
    const results = allContacts.filter(c =>
      c.name?.toLowerCase().includes(query.toLowerCase()) || c.phone?.includes(query)
    ).slice(0, 10)
    setSearchResults(results)
  }

  const addContact = (contact: {id: string; name: string; phone: string}) => {
    if (!selectedContacts.find(c => c.id === contact.id)) {
      setSelectedContacts([...selectedContacts, contact])
    }
    setContactSearch('')
    setSearchResults([])
  }

  const removeContact = (id: string) => {
    setSelectedContacts(selectedContacts.filter(c => c.id !== id))
  }

  const selectAll = () => {
    setSelectedContacts(allContacts)
  }

  const addContentItem = (type: CampaignContentType) => {
    setContentItems((items) => [
      ...items,
      { client_id: `${Date.now()}-${type}`, type, content: '' },
    ])
  }

  const updateContentItem = (index: number, patch: Partial<CampaignContentItem>) => {
    setContentItems((items) => items.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item))
  }

  const removeContentItem = (index: number) => {
    setContentItems((items) => items.filter((_, itemIndex) => itemIndex !== index))
  }

  const moveContentItem = (index: number, direction: -1 | 1) => {
    const nextIndex = index + direction
    if (nextIndex < 0 || nextIndex >= contentItems.length) return
    const nextItems = [...contentItems]
    const current = nextItems[index]
    nextItems[index] = nextItems[nextIndex]
    nextItems[nextIndex] = current
    setContentItems(nextItems)
  }

  const handleMediaFile = (index: number, file?: File) => {
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => {
      updateContentItem(index, {
        media_base64: reader.result as string,
        media_filename: file.name,
        media_url: undefined,
      })
    }
    reader.readAsDataURL(file)
  }

  const handleCreate = async () => {
    if (!name.trim()) { toast.error('Nome é obrigatório'); return }
    const validItems = contentItems.filter((item) =>
      item.type === 'text'
        ? item.content.trim()
        : item.media_base64 || item.media_url
    )
    if (validItems.length === 0) { toast.error('Adicione pelo menos um texto ou arquivo'); return }
    setSaving(true)
    try {
      const firstItem = validItems[0]
      const payload = {
        name,
        message_content: firstItem.content,
        message_type: firstItem.type,
        media_url: firstItem.media_url,
        content_items: validItems.map(({ client_id, ...item }) => item),
        send_speed: sendSpeed,
        frequency_cap_days: frequencyCapDays,
        scheduled_at: !campaign && scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'America/Sao_Paulo',
        filter_tag: targetType === 'tag' ? filterTag : undefined,
        contact_ids: targetType === 'selected' ? selectedContacts.map((contact) => contact.id) : undefined,
        total_contacts: targetType === 'all' ? allContacts.length : selectedContacts.length,
      }
      if (campaign) {
        await api.put(`/campaigns/${campaign.id}`, payload)
        toast.success('Campanha atualizada!')
      } else {
        await api.post('/campaigns', payload)
        toast.success('Campanha criada!')
      }
      onCreated()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao criar')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">{isEditing ? 'Editar campanha' : 'Nova campanha'}</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nome da campanha</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="input"
              placeholder="Ex: Promoção de Natal, Lembrete..."
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Agendar envio (opcional)</label>
              <input
                type="datetime-local"
                value={scheduledAt}
                min={new Date(Date.now() + 60000).toISOString().slice(0, 16)}
                onChange={(event) => setScheduledAt(event.target.value)}
                disabled={isEditing}
                className="input"
              />
              <p className="mt-1 text-xs text-gray-400">O envio só começa após aprovação.</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Limite de frequência</label>
              <select value={frequencyCapDays} onChange={(event) => setFrequencyCapDays(Number(event.target.value))} className="input">
                <option value={0}>Sem limite</option>
                <option value={1}>Não repetir em 1 dia</option>
                <option value={7}>Não repetir em 7 dias</option>
                <option value={15}>Não repetir em 15 dias</option>
                <option value={30}>Não repetir em 30 dias</option>
              </select>
              <p className="mt-1 text-xs text-gray-400">Evita excesso de mensagens ao mesmo contato.</p>
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <div>
                <label className="block text-sm font-medium text-gray-700">Conteúdo da campanha</label>
                <p className="text-xs text-gray-400">Monte a ordem exata do que o contato vai receber.</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <button type="button" onClick={() => addContentItem('text')} className="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"><FileText size={14} className="inline mr-1" /> Texto</button>
              <button type="button" onClick={() => addContentItem('image')} className="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"><Image size={14} className="inline mr-1" /> Imagem</button>
              <button type="button" onClick={() => addContentItem('video')} className="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"><Video size={14} className="inline mr-1" /> Vídeo</button>
              <button type="button" onClick={() => addContentItem('audio')} className="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"><Music size={14} className="inline mr-1" /> Áudio</button>
            </div>
          </div>

          <div className="space-y-3">
            {contentItems.map((item, index) => (
              <div key={item.client_id || index} className="rounded-lg border border-gray-200 p-3">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <p className="text-sm font-medium text-gray-900">
                      {index + 1}. {item.type === 'text' ? 'Texto' : item.type === 'image' ? 'Imagem' : item.type === 'video' ? 'Vídeo' : 'Áudio'}
                    </p>
                    <p className="text-xs text-gray-400">
                      {item.type === 'text' ? 'Mensagem escrita' : item.type === 'audio' ? 'Arquivo de áudio' : 'Arquivo com legenda opcional'}
                    </p>
                  </div>
                  <div className="flex items-center gap-1">
                    <button type="button" onClick={() => moveContentItem(index, -1)} disabled={index === 0} className="p-1 text-gray-400 hover:text-gray-700 disabled:opacity-30"><ArrowUp size={14} /></button>
                    <button type="button" onClick={() => moveContentItem(index, 1)} disabled={index === contentItems.length - 1} className="p-1 text-gray-400 hover:text-gray-700 disabled:opacity-30"><ArrowDown size={14} /></button>
                    <button type="button" onClick={() => removeContentItem(index)} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                  </div>
                </div>

                {item.type !== 'text' && (
                  <div className="mb-3">
                    <input
                      type="file"
                      accept={item.type === 'image' ? 'image/*' : item.type === 'video' ? 'video/*' : 'audio/*'}
                      onChange={(e) => handleMediaFile(index, e.target.files?.[0])}
                      className="input"
                    />
                    {(item.media_filename || item.media_url) && (
                      <p className="mt-1 text-xs text-gray-500">
                        {item.media_filename ? `Selecionado: ${item.media_filename}` : 'Arquivo atual mantido'}
                      </p>
                    )}
                  </div>
                )}

                {(item.type === 'text' || item.type === 'image' || item.type === 'video') && (
                  <textarea
                    value={item.content}
                    onChange={(e) => updateContentItem(index, { content: e.target.value })}
                    className="input resize-none"
                    rows={item.type === 'text' ? 4 : 2}
                    placeholder={item.type === 'text' ? 'Olá {{nome}}, temos uma novidade para você!' : 'Legenda opcional'}
                  />
                )}
                {item.type === 'audio' && (
                  <p className="text-xs text-gray-400">Para enviar texto junto com áudio, adicione um bloco de texto antes ou depois do áudio.</p>
                )}
                <p className="text-xs text-gray-400 mt-1">
                  Variáveis: {'{{nome}}'}, {'{{telefone}}'}, {'{{empresa}}'}, {'{{email}}'}
                </p>
              </div>
            ))}

            {contentItems.length === 0 && (
              <div className="rounded-lg border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500">
                Adicione texto, imagem, vídeo ou áudio para montar a campanha.
              </div>
            )}
          </div>

          {isEditing && (
            <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
              Ao editar uma campanha pausada, os contatos já enviados não recebem novamente. A nova sequência vale para os pendentes.
            </div>
          )}

          {/* Target audience */}
          {!isEditing && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Enviar para</label>
            <div className="space-y-2">
              <label className="flex items-center gap-2 p-3 border rounded-lg cursor-pointer hover:bg-gray-50" onClick={() => setTargetType('all')}>
                <input type="radio" name="target" checked={targetType === 'all'} onChange={() => setTargetType('all')} />
                <div>
                  <p className="text-sm font-medium text-gray-900">Todos os contatos ({allContacts.length})</p>
                  <p className="text-xs text-gray-400">Envia para toda a base de contatos</p>
                </div>
              </label>
              <label className="flex items-center gap-2 p-3 border rounded-lg cursor-pointer hover:bg-gray-50" onClick={() => setTargetType('tag')}>
                <input type="radio" name="target" checked={targetType === 'tag'} onChange={() => setTargetType('tag')} />
                <div>
                  <p className="text-sm font-medium text-gray-900">Por tag</p>
                  <p className="text-xs text-gray-400">Envia apenas para contatos com uma tag específica</p>
                </div>
              </label>
              <label className="flex items-center gap-2 p-3 border rounded-lg cursor-pointer hover:bg-gray-50" onClick={() => setTargetType('selected')}>
                <input type="radio" name="target" checked={targetType === 'selected'} onChange={() => setTargetType('selected')} />
                <div>
                  <p className="text-sm font-medium text-gray-900">Selecionar contatos</p>
                  <p className="text-xs text-gray-400">Escolha manualmente quem vai receber</p>
                </div>
              </label>
            </div>
          </div>
          )}

          {/* Tag filter */}
          {!isEditing && targetType === 'tag' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Selecionar tag</label>
              <select
                value={filterTag}
                onChange={(e) => setFilterTag(e.target.value)}
                className="input"
              >
                <option value="">Selecione uma tag...</option>
                {tags.map(tag => (
                  <option key={tag.id} value={tag.id}>{tag.name}</option>
                ))}
              </select>
            </div>
          )}

          {/* Contact selection */}
          {!isEditing && targetType === 'selected' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Buscar contatos ({selectedContacts.length} selecionados)
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={contactSearch}
                  onChange={(e) => searchContact(e.target.value)}
                  className="input"
                  placeholder="Buscar por nome ou telefone..."
                />
                {searchResults.length > 0 && (
                  <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-32 overflow-y-auto">
                    {searchResults.map(c => (
                      <button
                        key={c.id}
                        onClick={() => addContact(c)}
                        className="w-full text-left px-3 py-2 hover:bg-gray-50 text-sm border-b border-gray-50"
                      >
                        {c.name} <span className="text-gray-400">({c.phone})</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>

              <button onClick={selectAll} className="text-xs text-primary-600 mt-2 hover:underline">
                Selecionar todos ({allContacts.length})
              </button>

              {/* Selected contacts list */}
              {selectedContacts.length > 0 && (
                <div className="mt-2 max-h-32 overflow-y-auto space-y-1">
                  {selectedContacts.slice(0, 20).map(c => (
                    <div key={c.id} className="flex items-center justify-between px-2 py-1 bg-gray-50 rounded text-xs">
                      <span>{c.name}</span>
                      <button onClick={() => removeContact(c.id)} className="text-red-400 hover:text-red-600">
                        <X size={12} />
                      </button>
                    </div>
                  ))}
                  {selectedContacts.length > 20 && (
                    <p className="text-xs text-gray-400 px-2">+ {selectedContacts.length - 20} mais...</p>
                  )}
                </div>
              )}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Velocidade de envio</label>
            <select
              value={sendSpeed}
              onChange={(e) => setSendSpeed(parseInt(e.target.value))}
              className="input"
            >
              <option value={10}>10 msg/min (seguro)</option>
              <option value={20}>20 msg/min</option>
              <option value={30}>30 msg/min</option>
              <option value={60}>60 msg/min (risco)</option>
            </select>
          </div>

          <div className="p-3 bg-yellow-50 rounded-lg text-xs text-yellow-700">
            ⚠️ Respeite a LGPD. Envie apenas para contatos que autorizaram o recebimento.
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button onClick={onClose} className="btn-secondary flex-1">Cancelar</button>
          <button onClick={handleCreate} disabled={saving} className="btn-primary flex-1">
            {saving ? (isEditing ? 'Salvando...' : 'Criando...') : (isEditing ? 'Salvar campanha' : 'Criar campanha')}
          </button>
        </div>
      </div>
    </div>
  )
}
