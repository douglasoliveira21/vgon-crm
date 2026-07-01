'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Send, Pause, Play, Users, Check, Eye, X, Trash2, AlertTriangle } from 'lucide-react'

interface Campaign {
  id: string
  name: string
  status: string
  message_content?: string
  message_type: string
  send_speed?: number
  total_contacts: number
  sent_count: number
  delivered_count: number
  read_count: number
  replied_count: number
  failed_count: number
  scheduled_at?: string
  created_at: string
}

export default function CampaignsPage() {
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
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Marketing em Massa</h1>
          <p className="text-gray-500 mt-1">Envie campanhas para seus contatos via WhatsApp</p>
        </div>
        <button onClick={() => { setEditingCampaign(null); setShowForm(true) }} className="btn-primary">
          <Plus size={18} />
          Nova campanha
        </button>
      </div>

      {/* Stats */}
      {campaigns.length > 0 && (
        <div className="grid grid-cols-4 gap-4 mb-6">
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
                </div>
                <p className="text-sm text-gray-500 mt-1 flex items-center gap-4">
                  <span className="flex items-center gap-1"><Users size={14} /> {campaign.total_contacts} contatos</span>
                  <span className="flex items-center gap-1"><Send size={14} /> {campaign.sent_count} enviadas</span>
                  <span className="flex items-center gap-1"><Check size={14} /> {campaign.delivered_count} entregues</span>
                  <span className="flex items-center gap-1"><Eye size={14} /> {campaign.read_count} lidas</span>
                  {campaign.failed_count > 0 && (
                    <span className="flex items-center gap-1 text-red-600"><AlertTriangle size={14} /> {campaign.failed_count} falhas</span>
                  )}
                </p>
              </div>

              <div className="flex items-center gap-2">
                {campaign.status === 'draft' && (
                  <button onClick={() => startCampaign(campaign.id)} className="btn-primary text-sm py-2">
                    <Play size={14} /> Iniciar
                  </button>
                )}
                {campaign.status === 'sending' && (
                  <button onClick={() => pauseCampaign(campaign.id)} className="btn-secondary text-sm py-2">
                    <Pause size={14} /> Pausar
                  </button>
                )}
                {campaign.status === 'paused' && (
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
  const [messageContent, setMessageContent] = useState(campaign?.message_content || '')
  const [messageType, setMessageType] = useState(campaign?.message_type || 'text')
  const [mediaBase64, setMediaBase64] = useState('')
  const [mediaFileName, setMediaFileName] = useState('')
  const [sendSpeed, setSendSpeed] = useState(campaign?.send_speed || 30)
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

  const handleMediaFile = (file?: File) => {
    if (!file) return
    if (file.type.startsWith('image/')) setMessageType('image')
    else if (file.type.startsWith('video/')) setMessageType('video')
    else if (file.type.startsWith('audio/')) setMessageType('audio')

    const reader = new FileReader()
    reader.onload = () => {
      setMediaBase64(reader.result as string)
      setMediaFileName(file.name)
    }
    reader.readAsDataURL(file)
  }

  const handleCreate = async () => {
    if (!name.trim()) { toast.error('Nome é obrigatório'); return }
    if (messageType === 'text' && !messageContent.trim()) { toast.error('Mensagem é obrigatória'); return }
    if (messageType !== 'text' && !mediaBase64 && !isEditing) { toast.error('Selecione o arquivo da campanha'); return }
    setSaving(true)
    try {
      const payload = {
        name,
        message_content: messageContent,
        message_type: messageType,
        media_base64: mediaBase64 || undefined,
        media_filename: mediaFileName || undefined,
        send_speed: sendSpeed,
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
      <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
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

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Tipo de envio</label>
            <div className="grid grid-cols-4 gap-2">
              {[
                { value: 'text', label: 'Texto' },
                { value: 'image', label: 'Imagem' },
                { value: 'video', label: 'Vídeo' },
                { value: 'audio', label: 'Áudio' },
              ].map((type) => (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => setMessageType(type.value)}
                  className={`rounded-lg border px-3 py-2 text-sm ${messageType === type.value ? 'border-primary-500 bg-primary-50 text-primary-700' : 'border-gray-200 text-gray-600 hover:bg-gray-50'}`}
                >
                  {type.label}
                </button>
              ))}
            </div>
          </div>

          {messageType !== 'text' && !isEditing && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Arquivo</label>
              <input
                type="file"
                accept={messageType === 'image' ? 'image/*' : messageType === 'video' ? 'video/*' : 'audio/*'}
                onChange={(e) => handleMediaFile(e.target.files?.[0])}
                className="input"
              />
              {mediaFileName && (
                <p className="mt-1 text-xs text-gray-500">Selecionado: {mediaFileName}</p>
              )}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {messageType === 'text' ? 'Mensagem' : messageType === 'audio' ? 'Legenda interna' : 'Legenda'}
            </label>
            <textarea
              value={messageContent}
              onChange={(e) => setMessageContent(e.target.value)}
              className="input resize-none"
              rows={4}
              placeholder={messageType === 'text' ? 'Olá {{nome}}, temos uma novidade para você!' : 'Legenda opcional para acompanhar a mídia'}
            />
            <p className="text-xs text-gray-400 mt-1">
              Variáveis: {'{{nome}}'}, {'{{telefone}}'}, {'{{empresa}}'}, {'{{email}}'}
            </p>
          </div>

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
