'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Send, Pause, Play, BarChart3, Users, Check, Eye } from 'lucide-react'

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
                  <span className="flex items-center gap-1">
                    <Users size={14} /> {campaign.total_contacts} contatos
                  </span>
                  <span className="flex items-center gap-1">
                    <Send size={14} /> {campaign.sent_count} enviadas
                  </span>
                  <span className="flex items-center gap-1">
                    <Check size={14} /> {campaign.delivered_count} entregues
                  </span>
                  <span className="flex items-center gap-1">
                    <Eye size={14} /> {campaign.read_count} lidas
                  </span>
                </p>
              </div>

              <div className="flex items-center gap-2">
                {campaign.status === 'draft' && (
                  <button
                    onClick={() => startCampaign(campaign.id)}
                    className="btn-primary text-sm py-2"
                  >
                    <Play size={14} /> Iniciar
                  </button>
                )}
                {campaign.status === 'sending' && (
                  <button
                    onClick={() => pauseCampaign(campaign.id)}
                    className="btn-secondary text-sm py-2"
                  >
                    <Pause size={14} /> Pausar
                  </button>
                )}
                {campaign.status === 'paused' && (
                  <button
                    onClick={() => startCampaign(campaign.id)}
                    className="btn-primary text-sm py-2"
                  >
                    <Play size={14} /> Retomar
                  </button>
                )}
                <button className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
                  <BarChart3 size={18} />
                </button>
              </div>
            </div>

            {/* Progress Bar */}
            {campaign.total_contacts > 0 && campaign.status !== 'draft' && (
              <div className="mt-4">
                <div className="w-full bg-gray-100 rounded-full h-2">
                  <div
                    className="bg-primary-600 h-2 rounded-full transition-all"
                    style={{
                      width: `${Math.round((campaign.sent_count / campaign.total_contacts) * 100)}%`,
                    }}
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
            <p className="text-gray-500 mb-4">Nenhuma campanha criada</p>
            <button onClick={() => setShowForm(true)} className="btn-primary inline-flex">
              <Plus size={18} /> Criar primeira campanha
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
