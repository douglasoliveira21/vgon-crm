'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Send, Pause, Play, BarChart3, Users, Check, Eye, X, Trash2 } from 'lucide-react'

interface Campaign {
  id: string
  name: string
  status: string
  message_type: string
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

  useEffect(() => {
    fetchCampaigns()
  }, [])

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
      await api.post(`/campaigns/${id}/start`)
      setCampaigns((prev) =>
        prev.map((c) => (c.id === id ? { ...c, status: 'sending' } : c))
      )
      toast.success('Campanha iniciada')
    } catch {
      toast.error('Erro ao iniciar campanha')
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
        <button onClick={() => setShowForm(true)} className="btn-primary">
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
        {campaigns.map((campaign) => (
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
              </div>
            </div>

            {/* Progress Bar */}
            {campaign.total_contacts > 0 && campaign.status !== 'draft' && (
              <div className="mt-4">
                <div className="w-full bg-gray-100 rounded-full h-2">
                  <div
                    className="bg-primary-600 h-2 rounded-full transition-all"
                    style={{ width: `${Math.round((campaign.sent_count / campaign.total_contacts) * 100)}%` }}
                  />
                </div>
                <p className="text-xs text-gray-400 mt-1">
                  {Math.round((campaign.sent_count / campaign.total_contacts) * 100)}% enviado
                </p>
              </div>
            )}
          </div>
        ))}

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
          onClose={() => setShowForm(false)}
          onCreated={() => { setShowForm(false); fetchCampaigns() }}
        />
      )}
    </div>
  )
}

function CreateCampaignModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [name, setName] = useState('')
  const [messageContent, setMessageContent] = useState('')
  const [sendSpeed, setSendSpeed] = useState(30)
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
    api.get('/contacts', { params: { limit: 200 } }).then(res => {
      setAllContacts((res.data.contacts || []).map((c: any) => ({ id: c.id, name: c.name || c.phone, phone: c.phone })))
    }).catch(() => {})
  }, [])

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

  const handleCreate = async () => {
    if (!name.trim()) { toast.error('Nome é obrigatório'); return }
    if (!messageContent.trim()) { toast.error('Mensagem é obrigatória'); return }
    setSaving(true)
    try {
      await api.post('/campaigns', {
        name,
        message_content: messageContent,
        message_type: 'text',
        send_speed: sendSpeed,
        filter_tag: targetType === 'tag' ? filterTag : undefined,
        total_contacts: targetType === 'all' ? allContacts.length : selectedContacts.length,
      })
      toast.success('Campanha criada!')
      onCreated()
    } catch {
      toast.error('Erro ao criar')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Nova campanha</h3>
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
            <label className="block text-sm font-medium text-gray-700 mb-1">Mensagem</label>
            <textarea
              value={messageContent}
              onChange={(e) => setMessageContent(e.target.value)}
              className="input resize-none"
              rows={4}
              placeholder="Olá {{nome}}, temos uma novidade para você!"
            />
            <p className="text-xs text-gray-400 mt-1">
              Variáveis: {'{{nome}}'}, {'{{telefone}}'}, {'{{empresa}}'}, {'{{email}}'}
            </p>
          </div>

          {/* Target audience */}
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

          {/* Tag filter */}
          {targetType === 'tag' && (
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
          {targetType === 'selected' && (
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
            {saving ? 'Criando...' : 'Criar campanha'}
          </button>
        </div>
      </div>
    </div>
  )
}
