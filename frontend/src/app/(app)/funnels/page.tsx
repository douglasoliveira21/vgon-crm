'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, MoreVertical, DollarSign, Users, Edit2, Trash2, X, GripVertical } from 'lucide-react'

interface FunnelStage {
  id: string
  name: string
  color: string
  position: number
  deal_count: number
  deal_value: number
  is_won: boolean
  is_lost: boolean
}

interface Funnel {
  id: string
  name: string
  description?: string
  is_default: boolean
  is_active: boolean
  stages: FunnelStage[]
}

interface Deal {
  id: string
  title: string
  value: number
  status: string
  stage_id: string
  contact_name?: string
  assigned_to_name?: string
}

export default function FunnelsPage() {
  const [funnels, setFunnels] = useState<Funnel[]>([])
  const [selectedFunnel, setSelectedFunnel] = useState<Funnel | null>(null)
  const [deals, setDeals] = useState<Deal[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateFunnel, setShowCreateFunnel] = useState(false)
  const [showCreateDeal, setShowCreateDeal] = useState(false)
  const [targetStageId, setTargetStageId] = useState('')
  const [dragDealId, setDragDealId] = useState<string | null>(null)

  useEffect(() => {
    fetchFunnels()
  }, [])

  const fetchFunnels = async () => {
    try {
      const response = await api.get('/funnels')
      const funnelList = response.data.funnels || []
      setFunnels(funnelList)
      if (funnelList.length > 0 && !selectedFunnel) {
        setSelectedFunnel(funnelList[0])
        fetchDeals(funnelList[0].id)
      }
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchDeals = async (funnelId: string) => {
    try {
      const response = await api.get('/deals', { params: { funnel_id: funnelId } })
      setDeals(response.data.deals || [])
    } catch (error) {
      console.error('Error:', error)
    }
  }

  const createDeal = async (data: { title: string; value: number; contact_name: string }) => {
    if (!selectedFunnel || !targetStageId) return
    try {
      await api.post('/deals', {
        funnel_id: selectedFunnel.id,
        stage_id: targetStageId,
        title: data.title,
        value: data.value,
      })
      toast.success('Oportunidade criada')
      setShowCreateDeal(false)
      fetchDeals(selectedFunnel.id)
      fetchFunnels()
    } catch {
      toast.error('Erro ao criar')
    }
  }

  const moveDeal = async (dealId: string, newStageId: string) => {
    try {
      await api.put(`/deals/${dealId}/stage`, { stage_id: newStageId })
      setDeals((prev) =>
        prev.map((d) => (d.id === dealId ? { ...d, stage_id: newStageId } : d))
      )
      fetchFunnels() // Update counts
    } catch {
      toast.error('Erro ao mover')
    }
  }

  const deleteDeal = async (dealId: string) => {
    if (!confirm('Remover esta oportunidade?')) return
    try {
      await api.put(`/deals/${dealId}`, { status: 'lost', loss_reason: 'Removido' })
      setDeals((prev) => prev.filter((d) => d.id !== dealId))
      toast.success('Oportunidade removida')
      fetchFunnels()
    } catch {
      toast.error('Erro ao remover')
    }
  }

  const deleteFunnel = async (funnelId: string) => {
    if (!confirm('Remover este funil e todas as oportunidades?')) return
    try {
      await api.delete(`/funnels/${funnelId}`)
      setFunnels((prev) => prev.filter((f) => f.id !== funnelId))
      if (selectedFunnel?.id === funnelId) {
        setSelectedFunnel(null)
        setDeals([])
      }
      toast.success('Funil removido')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  // Drag and drop
  const handleDragStart = (dealId: string) => {
    setDragDealId(dealId)
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
  }

  const handleDrop = (e: React.DragEvent, stageId: string) => {
    e.preventDefault()
    if (dragDealId) {
      moveDeal(dragDealId, stageId)
      setDragDealId(null)
    }
  }

  const totalValue = deals.filter(d => d.status === 'open').reduce((sum, d) => sum + d.value, 0)

  return (
    <div className="p-6 h-screen flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Funil de Vendas</h1>
          <div className="flex items-center gap-4 mt-2">
            {funnels.map((funnel) => (
              <button
                key={funnel.id}
                onClick={() => { setSelectedFunnel(funnel); fetchDeals(funnel.id) }}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  selectedFunnel?.id === funnel.id
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-500 hover:bg-gray-100'
                }`}
              >
                {funnel.name}
              </button>
            ))}
            <button
              onClick={() => setShowCreateFunnel(true)}
              className="px-3 py-1.5 rounded-lg text-sm font-medium text-gray-400 hover:text-gray-600 hover:bg-gray-100"
            >
              + Novo funil
            </button>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-xs text-gray-400">Total em negociação</p>
            <p className="text-lg font-bold text-green-600">
              R$ {totalValue.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
            </p>
          </div>
          {selectedFunnel && (
            <button
              onClick={() => deleteFunnel(selectedFunnel.id)}
              className="p-2 text-gray-400 hover:text-red-500 rounded-lg hover:bg-red-50"
              title="Remover funil"
            >
              <Trash2 size={18} />
            </button>
          )}
        </div>
      </div>

      {/* Kanban Board */}
      {selectedFunnel && (
        <div className="flex-1 overflow-x-auto">
          <div className="flex gap-4 h-full min-w-max pb-4">
            {(selectedFunnel.stages || []).map((stage) => {
              const stageDeals = deals.filter((d) => d.stage_id === stage.id && d.status === 'open')
              const stageValue = stageDeals.reduce((sum, d) => sum + d.value, 0)

              return (
                <div
                  key={stage.id}
                  className="w-72 flex-shrink-0 flex flex-col"
                  onDragOver={handleDragOver}
                  onDrop={(e) => handleDrop(e, stage.id)}
                >
                  {/* Stage Header */}
                  <div className="bg-white rounded-t-xl border border-gray-200 p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-full" style={{ backgroundColor: stage.color }} />
                        <h3 className="font-medium text-gray-900 text-sm">{stage.name}</h3>
                        <span className="text-xs text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">
                          {stageDeals.length}
                        </span>
                      </div>
                    </div>
                    {stageValue > 0 && (
                      <p className="text-xs text-gray-500 mt-1 flex items-center gap-1">
                        <DollarSign size={12} />
                        R$ {stageValue.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
                      </p>
                    )}
                  </div>

                  {/* Cards */}
                  <div className="flex-1 bg-gray-50 border-x border-b border-gray-200 rounded-b-xl p-3 space-y-2 overflow-y-auto">
                    {stageDeals.map((deal) => (
                      <div
                        key={deal.id}
                        draggable
                        onDragStart={() => handleDragStart(deal.id)}
                        className="bg-white rounded-lg border border-gray-100 p-3 shadow-sm hover:shadow-md transition-shadow cursor-grab active:cursor-grabbing"
                      >
                        <div className="flex items-start justify-between">
                          <h4 className="text-sm font-medium text-gray-900 flex-1">{deal.title}</h4>
                          <button
                            onClick={() => deleteDeal(deal.id)}
                            className="p-1 text-gray-300 hover:text-red-500"
                          >
                            <X size={12} />
                          </button>
                        </div>
                        {deal.contact_name && (
                          <p className="text-xs text-gray-500 flex items-center gap-1 mt-1">
                            <Users size={10} /> {deal.contact_name}
                          </p>
                        )}
                        {deal.value > 0 && (
                          <p className="text-sm font-semibold text-green-600 mt-2">
                            R$ {deal.value.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
                          </p>
                        )}
                        {deal.assigned_to_name && (
                          <div className="mt-2 flex items-center gap-1">
                            <div className="w-5 h-5 bg-primary-100 rounded-full flex items-center justify-center">
                              <span className="text-primary-700 text-[10px] font-medium">
                                {deal.assigned_to_name.charAt(0)}
                              </span>
                            </div>
                            <span className="text-xs text-gray-400">{deal.assigned_to_name}</span>
                          </div>
                        )}
                      </div>
                    ))}

                    {/* Add card button */}
                    <button
                      onClick={() => { setTargetStageId(stage.id); setShowCreateDeal(true) }}
                      className="w-full p-2 border border-dashed border-gray-300 rounded-lg text-gray-400 text-xs hover:border-primary-400 hover:text-primary-500 transition-colors"
                    >
                      + Adicionar
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {funnels.length === 0 && !loading && (
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <DollarSign size={40} className="text-gray-300 mx-auto mb-4" />
            <p className="text-gray-500 mb-4">Nenhum funil criado</p>
            <button onClick={() => setShowCreateFunnel(true)} className="btn-primary inline-flex">
              <Plus size={18} /> Criar primeiro funil
            </button>
          </div>
        </div>
      )}

      {/* Create Funnel Modal */}
      {showCreateFunnel && (
        <CreateFunnelModal
          onClose={() => setShowCreateFunnel(false)}
          onCreated={() => { setShowCreateFunnel(false); fetchFunnels() }}
        />
      )}

      {/* Create Deal Modal */}
      {showCreateDeal && (
        <CreateDealModal
          onClose={() => setShowCreateDeal(false)}
          onCreated={createDeal}
        />
      )}
    </div>
  )
}

function CreateFunnelModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [name, setName] = useState('')
  const [stages, setStages] = useState([
    { name: 'Novo Lead', color: '#3B82F6' },
    { name: 'Qualificação', color: '#F59E0B' },
    { name: 'Proposta', color: '#8B5CF6' },
    { name: 'Fechamento', color: '#10B981' },
  ])
  const [saving, setSaving] = useState(false)

  const addStage = () => setStages([...stages, { name: '', color: '#6B7280' }])
  const removeStage = (i: number) => setStages(stages.filter((_, idx) => idx !== i))

  const handleCreate = async () => {
    if (!name.trim()) { toast.error('Nome é obrigatório'); return }
    if (stages.some(s => !s.name.trim())) { toast.error('Todas as etapas precisam de nome'); return }
    setSaving(true)
    try {
      await api.post('/funnels', { name, stages })
      toast.success('Funil criado!')
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
          <h3 className="text-lg font-semibold text-gray-900">Criar funil de vendas</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nome do funil</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="input"
              placeholder="Ex: Vendas B2B, Imóveis, Serviços..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Etapas</label>
            <div className="space-y-2">
              {stages.map((stage, i) => (
                <div key={i} className="flex items-center gap-2">
                  <input
                    type="color"
                    value={stage.color}
                    onChange={(e) => {
                      const newStages = [...stages]
                      newStages[i].color = e.target.value
                      setStages(newStages)
                    }}
                    className="w-8 h-8 rounded border border-gray-200 cursor-pointer"
                  />
                  <input
                    type="text"
                    value={stage.name}
                    onChange={(e) => {
                      const newStages = [...stages]
                      newStages[i].name = e.target.value
                      setStages(newStages)
                    }}
                    className="input flex-1"
                    placeholder={`Etapa ${i + 1}`}
                  />
                  {stages.length > 2 && (
                    <button onClick={() => removeStage(i)} className="text-red-400 hover:text-red-600">
                      <X size={16} />
                    </button>
                  )}
                </div>
              ))}
            </div>
            <button onClick={addStage} className="text-xs text-primary-600 mt-2 hover:text-primary-700">
              + Adicionar etapa
            </button>
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button onClick={onClose} className="btn-secondary flex-1">Cancelar</button>
          <button onClick={handleCreate} disabled={saving} className="btn-primary flex-1">
            {saving ? 'Criando...' : 'Criar funil'}
          </button>
        </div>
      </div>
    </div>
  )
}

function CreateDealModal({ onClose, onCreated }: { onClose: () => void; onCreated: (data: any) => void }) {
  const [title, setTitle] = useState('')
  const [value, setValue] = useState('')
  const [contactName, setContactName] = useState('')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Nova oportunidade</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Título</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="input"
              placeholder="Ex: Proposta comercial - João"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Valor (R$)</label>
            <input
              type="number"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className="input"
              placeholder="0,00"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Contato</label>
            <input
              type="text"
              value={contactName}
              onChange={(e) => setContactName(e.target.value)}
              className="input"
              placeholder="Nome do contato"
            />
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button onClick={onClose} className="btn-secondary flex-1">Cancelar</button>
          <button
            onClick={() => onCreated({ title, value: parseFloat(value) || 0, contact_name: contactName })}
            disabled={!title.trim()}
            className="btn-primary flex-1"
          >
            Criar
          </button>
        </div>
      </div>
    </div>
  )
}
