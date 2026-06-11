'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, MoreVertical, DollarSign, Users } from 'lucide-react'

interface FunnelStage {
  id: string
  name: string
  color: string
  position: number
  deal_count: number
  deal_value: number
}

interface Funnel {
  id: string
  name: string
  stages: FunnelStage[]
}

interface Deal {
  id: string
  title: string
  value: number
  contact_name?: string
  assigned_to_name?: string
  stage_id: string
}

export default function FunnelsPage() {
  const [funnels, setFunnels] = useState<Funnel[]>([])
  const [selectedFunnel, setSelectedFunnel] = useState<Funnel | null>(null)
  const [deals, setDeals] = useState<Deal[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchFunnels()
  }, [])

  const fetchFunnels = async () => {
    try {
      const response = await api.get('/funnels')
      const funnelList = response.data.funnels || []
      setFunnels(funnelList)
      if (funnelList.length > 0) {
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

  const moveDeal = async (dealId: string, stageId: string) => {
    try {
      await api.put(`/deals/${dealId}/stage`, { stage_id: stageId })
      setDeals((prev) =>
        prev.map((d) => (d.id === dealId ? { ...d, stage_id: stageId } : d))
      )
    } catch {
      toast.error('Erro ao mover deal')
    }
  }

  if (loading) {
    return <div className="p-6 animate-pulse"><div className="h-8 bg-gray-200 rounded w-48"></div></div>
  }

  return (
    <div className="p-6 h-screen flex flex-col">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Funil de Vendas</h1>
          <div className="flex gap-2 mt-2">
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
          </div>
        </div>
        <button className="btn-primary">
          <Plus size={18} />
          Nova oportunidade
        </button>
      </div>

      {/* Kanban Board */}
      {selectedFunnel && (
        <div className="flex-1 overflow-x-auto">
          <div className="flex gap-4 h-full min-w-max pb-4">
            {(selectedFunnel.stages || []).map((stage) => {
              const stageDeals = deals.filter((d) => d.stage_id === stage.id)
              const totalValue = stageDeals.reduce((sum, d) => sum + d.value, 0)

              return (
                <div key={stage.id} className="w-72 flex-shrink-0 flex flex-col">
                  {/* Stage Header */}
                  <div className="bg-white rounded-t-xl border border-gray-200 p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: stage.color }}
                        />
                        <h3 className="font-medium text-gray-900 text-sm">{stage.name}</h3>
                        <span className="text-xs text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">
                          {stageDeals.length}
                        </span>
                      </div>
                      <button className="text-gray-400 hover:text-gray-600">
                        <MoreVertical size={14} />
                      </button>
                    </div>
                    {totalValue > 0 && (
                      <p className="text-xs text-gray-500 mt-1 flex items-center gap-1">
                        <DollarSign size={12} />
                        R$ {totalValue.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
                      </p>
                    )}
                  </div>

                  {/* Cards */}
                  <div className="flex-1 bg-gray-50 border-x border-b border-gray-200 rounded-b-xl p-3 space-y-2 overflow-y-auto">
                    {stageDeals.map((deal) => (
                      <div
                        key={deal.id}
                        className="bg-white rounded-lg border border-gray-100 p-3 shadow-sm hover:shadow-md transition-shadow cursor-pointer"
                      >
                        <h4 className="text-sm font-medium text-gray-900 mb-1">{deal.title}</h4>
                        {deal.contact_name && (
                          <p className="text-xs text-gray-500 flex items-center gap-1">
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
                    <button className="w-full p-2 border border-dashed border-gray-300 rounded-lg text-gray-400 text-xs hover:border-primary-400 hover:text-primary-500 transition-colors">
                      + Adicionar
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {funnels.length === 0 && (
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <p className="text-gray-500 mb-4">Nenhum funil criado</p>
            <button className="btn-primary">
              <Plus size={18} />
              Criar primeiro funil
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
