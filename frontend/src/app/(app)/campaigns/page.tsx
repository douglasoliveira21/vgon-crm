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
  const [filterTag, setFilterTag] = useState('')
  const [saving, setSaving] = useState(false)

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
              placeholder="Ex: Promoção de Natal, Lembrete de pagamento..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Mensagem</label>
            <textarea
              value={messageContent}
              onChange={(e) => setMessageContent(e.target.value)}
              className="input resize-none"
              rows={5}
              placeholder="Olá {{nome}}, temos uma novidade para você!"
            />
            <p className="text-xs text-gray-400 mt-1">
              Variáveis: {'{{nome}}'}, {'{{telefone}}'}, {'{{empresa}}'}, {'{{email}}'}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Velocidade de envio</label>
            <select
              value={sendSpeed}
              onChange={(e) => setSendSpeed(parseInt(e.target.value))}
              className="input"
            >
              <option value={10}>10 mensagens/minuto (seguro)</option>
              <option value={20}>20 mensagens/minuto</option>
              <option value={30}>30 mensagens/minuto</option>
              <option value={60}>60 mensagens/minuto (risco de bloqueio)</option>
            </select>
            <p className="text-xs text-gray-400 mt-1">
              ⚠️ Velocidades altas podem causar bloqueio temporário do WhatsApp
            </p>
          </div>

          <div className="p-3 bg-yellow-50 rounded-lg text-xs text-yellow-700">
            📋 A campanha será enviada para todos os contatos ativos. Certifique-se de que os contatos autorizaram o recebimento (LGPD).
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
