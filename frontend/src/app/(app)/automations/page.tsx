'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Bot, Play, Pause, Edit2, Trash2, Zap, Ticket, Search, Building2, Link2, Eye } from 'lucide-react'

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

interface GLPIEntity {
  id: number
  name: string
  completename?: string
  complete_name?: string
}

interface GLPITicketResult {
  id: number
  title: string
  content: string
  status: number
  status_label: string
  entity_id: number
  date: string
  date_mod: string
  type: number
  priority: number
}

export default function AutomationsPage() {
  const router = useRouter()
  const [flows, setFlows] = useState<BotFlow[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'bots' | 'glpi'>('bots')

  // GLPI State
  const [glpiEntities, setGlpiEntities] = useState<GLPIEntity[]>([])
  const [glpiLoading, setGlpiLoading] = useState(false)
  const [showCreateTicket, setShowCreateTicket] = useState(false)
  const [showViewTicket, setShowViewTicket] = useState(false)
  const [ticketForm, setTicketForm] = useState({ title: '', content: '', entity_id: 0, ticket_type: 1, priority: 3 })
  const [searchTicketId, setSearchTicketId] = useState('')
  const [viewedTicket, setViewedTicket] = useState<GLPITicketResult | null>(null)
  const [createdTicketId, setCreatedTicketId] = useState<number | null>(null)

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

  // GLPI Functions
  const fetchGLPIEntities = async () => {
    setGlpiLoading(true)
    try {
      const response = await api.get('/glpi/entities')
      setGlpiEntities(response.data.entities || [])
    } catch (error: any) {
      toast.error(error?.response?.data?.error || 'Erro ao buscar entidades GLPI')
    } finally {
      setGlpiLoading(false)
    }
  }

  const createGLPITicket = async () => {
    if (!ticketForm.title || !ticketForm.content) {
      toast.error('Título e descrição são obrigatórios')
      return
    }
    try {
      const response = await api.post('/glpi/tickets', ticketForm)
      setCreatedTicketId(response.data.ticket_id)
      toast.success(`Chamado #${response.data.ticket_id} criado com sucesso!`)
      setTicketForm({ title: '', content: '', entity_id: 0, ticket_type: 1, priority: 3 })
    } catch (error: any) {
      toast.error(error?.response?.data?.error || 'Erro ao criar chamado')
    }
  }

  const searchGLPITicket = async () => {
    if (!searchTicketId) {
      toast.error('Informe o número do ticket')
      return
    }
    try {
      const response = await api.get(`/glpi/tickets/${searchTicketId}`)
      setViewedTicket(response.data)
      setShowViewTicket(true)
    } catch (error: any) {
      toast.error(error?.response?.data?.error || 'Chamado não encontrado')
    }
  }

  const verifyEntity = async (entityId: number) => {
    try {
      const response = await api.get(`/glpi/entities/${entityId}`)
      toast.success(`Entidade: ${response.data.complete_name || response.data.name}`)
    } catch (error: any) {
      toast.error(error?.response?.data?.error || 'Entidade não encontrada')
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

  const priorityLabels: Record<number, string> = {
    1: 'Muito baixa',
    2: 'Baixa',
    3: 'Média',
    4: 'Alta',
    5: 'Muito alta',
    6: 'Maior',
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Bots e Automações</h1>
          <p className="text-gray-500 mt-1">Crie fluxos automáticos e integre com GLPI</p>
        </div>
        {activeTab === 'bots' && (
          <button onClick={() => router.push('/automations/new')} className="btn-primary">
            <Plus size={18} />
            Novo fluxo
          </button>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-gray-200">
        <button
          onClick={() => setActiveTab('bots')}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'bots' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          <span className="flex items-center gap-2"><Bot size={16} /> Bots e Fluxos</span>
        </button>
        <button
          onClick={() => { setActiveTab('glpi'); fetchGLPIEntities() }}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'glpi' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          <span className="flex items-center gap-2"><Ticket size={16} /> GLPI - Chamados</span>
        </button>
      </div>

      {/* Bots Tab */}
      {activeTab === 'bots' && (
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
                <button
                  onClick={() => router.push(`/automations/${flow.id}`)}
                  className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg"
                >
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
              <button onClick={() => router.push('/automations/new')} className="btn-primary inline-flex">
                <Plus size={18} /> Criar primeiro fluxo
              </button>
            </div>
          )}
        </div>
      )}

      {/* GLPI Tab */}
      {activeTab === 'glpi' && (
        <div className="space-y-6">
          {/* Actions Row */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <button
              onClick={() => { setShowCreateTicket(true); setCreatedTicketId(null) }}
              className="card p-5 hover:border-primary-300 hover:shadow-md transition-all text-left"
            >
              <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center mb-3">
                <Plus size={20} className="text-blue-600" />
              </div>
              <h3 className="font-medium text-gray-900">Abrir Chamado</h3>
              <p className="text-xs text-gray-500 mt-1">Criar um novo ticket no GLPI</p>
            </button>

            <button
              onClick={() => setShowViewTicket(true)}
              className="card p-5 hover:border-primary-300 hover:shadow-md transition-all text-left"
            >
              <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center mb-3">
                <Eye size={20} className="text-green-600" />
              </div>
              <h3 className="font-medium text-gray-900">Visualizar Chamado</h3>
              <p className="text-xs text-gray-500 mt-1">Buscar ticket pelo número</p>
            </button>

            <button
              onClick={() => fetchGLPIEntities()}
              className="card p-5 hover:border-primary-300 hover:shadow-md transition-all text-left"
            >
              <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center mb-3">
                <Building2 size={20} className="text-purple-600" />
              </div>
              <h3 className="font-medium text-gray-900">Entidades</h3>
              <p className="text-xs text-gray-500 mt-1">Visualizar e verificar entidades</p>
            </button>
          </div>

          {/* Entities List */}
          {glpiEntities.length > 0 && (
            <div className="card p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                <Building2 size={20} className="text-purple-600" />
                Entidades GLPI
              </h3>
              <div className="space-y-2 max-h-64 overflow-y-auto">
                {glpiEntities.map((entity) => (
                  <div key={entity.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div>
                      <p className="text-sm font-medium text-gray-900">{entity.name}</p>
                      {(entity.completename || entity.complete_name) && (
                        <p className="text-xs text-gray-500">{entity.completename || entity.complete_name}</p>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-400 bg-gray-200 px-2 py-0.5 rounded">ID: {entity.id}</span>
                      <button
                        onClick={() => verifyEntity(entity.id)}
                        className="text-xs text-primary-600 hover:text-primary-700 font-medium"
                      >
                        Verificar
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {glpiLoading && (
            <div className="card p-8 text-center">
              <p className="text-gray-500">Carregando dados do GLPI...</p>
            </div>
          )}

          {!glpiLoading && glpiEntities.length === 0 && (
            <div className="card p-8 text-center">
              <Ticket size={40} className="text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500 mb-2">GLPI Integration</p>
              <p className="text-sm text-gray-400 max-w-sm mx-auto">
                Configure as variáveis GLPI_BASE_URL, GLPI_APP_TOKEN e GLPI_USER_TOKEN para conectar ao seu GLPI.
              </p>
            </div>
          )}

          {/* Create Ticket Modal */}
          {showCreateTicket && (
            <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
              <div className="bg-white rounded-2xl p-6 w-full max-w-lg">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-lg font-semibold text-gray-900">Abrir Chamado GLPI</h3>
                  <button onClick={() => setShowCreateTicket(false)} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
                </div>

                {createdTicketId ? (
                  <div className="text-center py-6">
                    <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                      <Ticket size={28} className="text-green-600" />
                    </div>
                    <h4 className="text-lg font-semibold text-gray-900 mb-1">Chamado criado!</h4>
                    <p className="text-gray-500">Número do ticket: <span className="font-bold text-primary-600">#{createdTicketId}</span></p>
                    <button onClick={() => { setShowCreateTicket(false); setCreatedTicketId(null) }} className="btn-primary mt-4">
                      Fechar
                    </button>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div>
                      <label className="text-sm font-medium text-gray-700 mb-1 block">Título *</label>
                      <input
                        type="text"
                        value={ticketForm.title}
                        onChange={(e) => setTicketForm({ ...ticketForm, title: e.target.value })}
                        className="input"
                        placeholder="Título do chamado"
                      />
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-700 mb-1 block">Descrição *</label>
                      <textarea
                        value={ticketForm.content}
                        onChange={(e) => setTicketForm({ ...ticketForm, content: e.target.value })}
                        className="input min-h-[100px]"
                        placeholder="Descreva o problema ou solicitação..."
                      />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-gray-700 mb-1 block">Tipo</label>
                        <select
                          value={ticketForm.ticket_type}
                          onChange={(e) => setTicketForm({ ...ticketForm, ticket_type: Number(e.target.value) })}
                          className="input"
                        >
                          <option value={1}>Incidente</option>
                          <option value={2}>Requisição</option>
                        </select>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-700 mb-1 block">Prioridade</label>
                        <select
                          value={ticketForm.priority}
                          onChange={(e) => setTicketForm({ ...ticketForm, priority: Number(e.target.value) })}
                          className="input"
                        >
                          {Object.entries(priorityLabels).map(([val, label]) => (
                            <option key={val} value={val}>{label}</option>
                          ))}
                        </select>
                      </div>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-700 mb-1 block">Entidade (Vinculação)</label>
                      <select
                        value={ticketForm.entity_id}
                        onChange={(e) => setTicketForm({ ...ticketForm, entity_id: Number(e.target.value) })}
                        className="input"
                      >
                        <option value={0}>Selecionar entidade...</option>
                        {glpiEntities.map((e) => (
                          <option key={e.id} value={e.id}>{e.completename || e.complete_name || e.name}</option>
                        ))}
                      </select>
                    </div>
                    <div className="flex gap-3 pt-2">
                      <button onClick={() => setShowCreateTicket(false)} className="btn-secondary flex-1">
                        Cancelar
                      </button>
                      <button onClick={createGLPITicket} className="btn-primary flex-1">
                        <Ticket size={16} /> Criar Chamado
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* View Ticket Modal */}
          {showViewTicket && (
            <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
              <div className="bg-white rounded-2xl p-6 w-full max-w-lg">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-lg font-semibold text-gray-900">Visualizar Chamado</h3>
                  <button onClick={() => { setShowViewTicket(false); setViewedTicket(null) }} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
                </div>

                {/* Search */}
                <div className="flex gap-2 mb-4">
                  <input
                    type="text"
                    value={searchTicketId}
                    onChange={(e) => setSearchTicketId(e.target.value)}
                    placeholder="Número do ticket..."
                    className="input flex-1"
                    onKeyDown={(e) => { if (e.key === 'Enter') searchGLPITicket() }}
                  />
                  <button onClick={searchGLPITicket} className="btn-primary">
                    <Search size={16} /> Buscar
                  </button>
                </div>

                {/* Ticket Details */}
                {viewedTicket && (
                  <div className="bg-gray-50 rounded-xl p-4 space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium text-gray-400 uppercase">Ticket</span>
                      <span className="text-sm font-bold text-primary-600">#{viewedTicket.id}</span>
                    </div>
                    <div>
                      <span className="text-xs font-medium text-gray-400 uppercase">Título</span>
                      <p className="text-sm text-gray-900 font-medium">{viewedTicket.title}</p>
                    </div>
                    <div>
                      <span className="text-xs font-medium text-gray-400 uppercase">Status</span>
                      <p className="text-sm">
                        <span className={`badge ${
                          viewedTicket.status <= 3 ? 'badge-blue' :
                          viewedTicket.status === 4 ? 'badge-yellow' :
                          viewedTicket.status === 5 ? 'badge-green' : 'badge-gray'
                        }`}>
                          {viewedTicket.status_label}
                        </span>
                      </p>
                    </div>
                    <div>
                      <span className="text-xs font-medium text-gray-400 uppercase">Descrição</span>
                      <p className="text-sm text-gray-700 whitespace-pre-wrap">{viewedTicket.content}</p>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <span className="text-xs font-medium text-gray-400 uppercase">Prioridade</span>
                        <p className="text-sm text-gray-900">{priorityLabels[viewedTicket.priority] || viewedTicket.priority}</p>
                      </div>
                      <div>
                        <span className="text-xs font-medium text-gray-400 uppercase">Entidade ID</span>
                        <p className="text-sm text-gray-900">{viewedTicket.entity_id}</p>
                      </div>
                    </div>
                    {viewedTicket.date && (
                      <div>
                        <span className="text-xs font-medium text-gray-400 uppercase">Data de Abertura</span>
                        <p className="text-sm text-gray-700">{viewedTicket.date}</p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
