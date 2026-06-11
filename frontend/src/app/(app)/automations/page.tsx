'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Bot, Play, Pause, Edit2, Trash2, Zap } from 'lucide-react'

interface BotFlow {
  id: string
  name: string
  description?: string
  trigger_type: string
  trigger_value?: string
  is_active: boolean
  nodes: any[]
  edges: any[]
  created_at: string
}

export default function AutomationsPage() {
  const [flows, setFlows] = useState<BotFlow[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchFlows()
  }, [])

  const fetchFlows = async () => {
    try {
      const response = await api.get('/bot-flows')
      setFlows(response.data.flows || [])
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const toggleFlow = async (flow: BotFlow) => {
    try {
      await api.put(`/bot-flows/${flow.id}`, {
        ...flow,
        is_active: !flow.is_active,
      })
      setFlows((prev) =>
        prev.map((f) => (f.id === flow.id ? { ...f, is_active: !f.is_active } : f))
      )
      toast.success(flow.is_active ? 'Fluxo desativado' : 'Fluxo ativado')
    } catch {
      toast.error('Erro ao atualizar')
    }
  }

  const deleteFlow = async (id: string) => {
    if (!confirm('Remover este fluxo?')) return
    try {
      await api.delete(`/bot-flows/${id}`)
      setFlows((prev) => prev.filter((f) => f.id !== id))
      toast.success('Fluxo removido')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  const triggerLabels: Record<string, string> = {
    new_conversation: 'Nova conversa',
    keyword: 'Palavra-chave',
    off_hours: 'Fora do horário',
    no_response: 'Sem resposta do cliente',
    transfer: 'Conversa transferida',
    funnel_stage: 'Entrada em etapa do funil',
    tag_added: 'Tag adicionada',
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Bots e Automações</h1>
          <p className="text-gray-500 mt-1">Crie fluxos automáticos para atendimento</p>
        </div>
        <button className="btn-primary">
          <Plus size={18} />
          Novo fluxo
        </button>
      </div>

      {/* Flows List */}
      <div className="space-y-4">
        {flows.map((flow) => (
          <div key={flow.id} className="card p-5 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                flow.is_active ? 'bg-green-100' : 'bg-gray-100'
              }`}>
                <Bot size={22} className={flow.is_active ? 'text-green-600' : 'text-gray-400'} />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-medium text-gray-900">{flow.name}</h3>
                  {flow.is_active ? (
                    <span className="badge badge-green">Ativo</span>
                  ) : (
                    <span className="badge badge-gray">Inativo</span>
                  )}
                </div>
                {flow.description && (
                  <p className="text-sm text-gray-500 mt-0.5">{flow.description}</p>
                )}
                <div className="flex items-center gap-2 mt-1">
                  <Zap size={12} className="text-yellow-500" />
                  <span className="text-xs text-gray-400">
                    Gatilho: {triggerLabels[flow.trigger_type] || flow.trigger_type}
                    {flow.trigger_value && ` - "${flow.trigger_value}"`}
                  </span>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => toggleFlow(flow)}
                className={`p-2 rounded-lg transition-colors ${
                  flow.is_active
                    ? 'text-yellow-600 hover:bg-yellow-50'
                    : 'text-green-600 hover:bg-green-50'
                }`}
                title={flow.is_active ? 'Desativar' : 'Ativar'}
              >
                {flow.is_active ? <Pause size={18} /> : <Play size={18} />}
              </button>
              <button className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg">
                <Edit2 size={18} />
              </button>
              <button
                onClick={() => deleteFlow(flow.id)}
                className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg"
              >
                <Trash2 size={18} />
              </button>
            </div>
          </div>
        ))}

        {flows.length === 0 && !loading && (
          <div className="card p-12 text-center">
            <Bot size={40} className="text-gray-300 mx-auto mb-4" />
            <p className="text-gray-500 mb-2">Nenhuma automação criada</p>
            <p className="text-sm text-gray-400 mb-6 max-w-sm mx-auto">
              Crie fluxos de bots para automatizar o atendimento, respostas e ações
            </p>
            <button className="btn-primary inline-flex">
              <Plus size={18} /> Criar primeiro fluxo
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
