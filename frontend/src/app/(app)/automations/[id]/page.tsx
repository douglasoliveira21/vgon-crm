'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  Plus,
  MessageSquare,
  HelpCircle,
  GitBranch,
  ArrowRightLeft,
  Tag,
  Globe,
  Clock,
  Trash2,
  Save,
  ArrowLeft,
  Play,
  Pause,
  Zap,
  Send,
} from 'lucide-react'

interface FlowNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: Record<string, any>
  next?: string[]
}

interface FlowEdge {
  id: string
  source: string
  target: string
  label?: string
}

const NODE_TYPES = [
  { type: 'send_message', label: 'Enviar mensagem', icon: MessageSquare, color: 'bg-blue-100 text-blue-600' },
  { type: 'ask_question', label: 'Fazer pergunta', icon: HelpCircle, color: 'bg-green-100 text-green-600' },
  { type: 'condition', label: 'Condição', icon: GitBranch, color: 'bg-yellow-100 text-yellow-600' },
  { type: 'transfer_team', label: 'Transferir para time', icon: ArrowRightLeft, color: 'bg-orange-100 text-orange-600' },
  { type: 'transfer_agent', label: 'Transferir para agente', icon: ArrowRightLeft, color: 'bg-orange-100 text-orange-600' },
  { type: 'add_tag', label: 'Adicionar tag', icon: Tag, color: 'bg-purple-100 text-purple-600' },
  { type: 'call_webhook', label: 'Chamar webhook', icon: Globe, color: 'bg-gray-100 text-gray-600' },
  { type: 'delay', label: 'Aguardar', icon: Clock, color: 'bg-pink-100 text-pink-600' },
  { type: 'end', label: 'Encerrar', icon: Pause, color: 'bg-red-100 text-red-600' },
]

export default function FlowEditorPage() {
  const params = useParams()
  const router = useRouter()
  const flowId = params.id as string
  const isNew = flowId === 'new'

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [triggerType, setTriggerType] = useState('new_conversation')
  const [triggerValue, setTriggerValue] = useState('')
  const [isActive, setIsActive] = useState(false)
  const [nodes, setNodes] = useState<FlowNode[]>([])
  const [edges, setEdges] = useState<FlowEdge[]>([])
  const [selectedNode, setSelectedNode] = useState<FlowNode | null>(null)
  const [showAddNode, setShowAddNode] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!isNew) {
      fetchFlow()
    }
  }, [flowId])

  const fetchFlow = async () => {
    try {
      const response = await api.get('/bot-flows')
      const flows = response.data.flows || []
      const flow = flows.find((f: any) => f.id === flowId)
      if (flow) {
        setName(flow.name)
        setDescription(flow.description || '')
        setTriggerType(flow.trigger_type)
        setTriggerValue(flow.trigger_value || '')
        setIsActive(flow.is_active)
        setNodes(flow.nodes || [])
        setEdges(flow.edges || [])
      }
    } catch {
      toast.error('Erro ao carregar fluxo')
    }
  }

  const saveFlow = async () => {
    if (!name.trim()) {
      toast.error('Nome do fluxo é obrigatório')
      return
    }
    setSaving(true)
    try {
      const payload = {
        name,
        description,
        trigger_type: triggerType,
        trigger_value: triggerValue,
        is_active: isActive,
        nodes: JSON.stringify(nodes),
        edges: JSON.stringify(edges),
      }

      if (isNew) {
        await api.post('/bot-flows', payload)
        toast.success('Fluxo criado')
      } else {
        await api.put(`/bot-flows/${flowId}`, payload)
        toast.success('Fluxo salvo')
      }
      router.push('/automations')
    } catch {
      toast.error('Erro ao salvar fluxo')
    } finally {
      setSaving(false)
    }
  }

  const addNode = (type: string) => {
    const newNode: FlowNode = {
      id: `node_${Date.now()}`,
      type,
      position: { x: 250, y: (nodes.length + 1) * 120 },
      data: getDefaultNodeData(type),
    }
    setNodes([...nodes, newNode])
    setSelectedNode(newNode)
    setShowAddNode(false)

    // Auto-connect to previous node
    if (nodes.length > 0) {
      const lastNode = nodes[nodes.length - 1]
      const newEdge: FlowEdge = {
        id: `edge_${Date.now()}`,
        source: lastNode.id,
        target: newNode.id,
      }
      setEdges([...edges, newEdge])
    }
  }

  const updateNodeData = (nodeId: string, data: Record<string, any>) => {
    setNodes(nodes.map(n => n.id === nodeId ? { ...n, data: { ...n.data, ...data } } : n))
    if (selectedNode?.id === nodeId) {
      setSelectedNode({ ...selectedNode, data: { ...selectedNode.data, ...data } })
    }
  }

  const deleteNode = (nodeId: string) => {
    setNodes(nodes.filter(n => n.id !== nodeId))
    setEdges(edges.filter(e => e.source !== nodeId && e.target !== nodeId))
    if (selectedNode?.id === nodeId) setSelectedNode(null)
  }

  const getDefaultNodeData = (type: string): Record<string, any> => {
    switch (type) {
      case 'send_message': return { message: '' }
      case 'ask_question': return { question: '', options: [''] }
      case 'condition': return { field: 'message', operator: 'contains', value: '' }
      case 'transfer_team': return { team_id: '' }
      case 'transfer_agent': return { agent_id: '' }
      case 'add_tag': return { tag_name: '' }
      case 'call_webhook': return { url: '', method: 'POST' }
      case 'delay': return { seconds: 5 }
      case 'end': return { close_conversation: false }
      default: return {}
    }
  }

  const getNodeTypeInfo = (type: string) => NODE_TYPES.find(t => t.type === type)

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between flex-shrink-0">
        <div className="flex items-center gap-4">
          <button onClick={() => router.push('/automations')} className="text-gray-400 hover:text-gray-600">
            <ArrowLeft size={20} />
          </button>
          <div>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Nome do fluxo..."
              className="text-lg font-semibold text-gray-900 border-none outline-none bg-transparent"
            />
          </div>
        </div>
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="rounded border-gray-300"
            />
            <span className={isActive ? 'text-green-600 font-medium' : 'text-gray-500'}>
              {isActive ? 'Ativo' : 'Inativo'}
            </span>
          </label>
          <button onClick={saveFlow} disabled={saving} className="btn-primary">
            <Save size={16} /> {saving ? 'Salvando...' : 'Salvar'}
          </button>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Flow canvas */}
        <div className="flex-1 bg-gray-50 p-6 overflow-y-auto">
          {/* Trigger */}
          <div className="mb-4">
            <div className="inline-flex items-center gap-2 bg-indigo-100 text-indigo-700 px-4 py-2 rounded-lg text-sm font-medium">
              <Zap size={16} />
              Gatilho: {triggerType === 'new_conversation' ? 'Nova conversa' : 
                       triggerType === 'keyword' ? `Palavra-chave: "${triggerValue}"` :
                       triggerType === 'off_hours' ? 'Fora do horário' :
                       triggerType}
            </div>
          </div>

          {/* Nodes */}
          <div className="space-y-3">
            {nodes.map((node, index) => {
              const info = getNodeTypeInfo(node.type)
              const Icon = info?.icon || MessageSquare

              return (
                <div key={node.id}>
                  {/* Connection line */}
                  {index > 0 && (
                    <div className="flex justify-center py-1">
                      <div className="w-0.5 h-6 bg-gray-300"></div>
                    </div>
                  )}

                  <div
                    onClick={() => setSelectedNode(node)}
                    className={`max-w-md mx-auto p-4 rounded-xl border-2 cursor-pointer transition-all ${
                      selectedNode?.id === node.id
                        ? 'border-primary-500 shadow-lg'
                        : 'border-gray-200 hover:border-gray-300 shadow-sm'
                    } bg-white`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${info?.color}`}>
                          <Icon size={16} />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-gray-900">{info?.label}</p>
                          <p className="text-xs text-gray-500 truncate max-w-[200px]">
                            {node.type === 'send_message' && (node.data.message || 'Sem mensagem')}
                            {node.type === 'ask_question' && (node.data.question || 'Sem pergunta')}
                            {node.type === 'condition' && `Se ${node.data.field} ${node.data.operator} "${node.data.value}"`}
                            {node.type === 'delay' && `Aguardar ${node.data.seconds}s`}
                            {node.type === 'add_tag' && (node.data.tag_name || 'Sem tag')}
                            {node.type === 'transfer_team' && 'Transferir'}
                            {node.type === 'call_webhook' && (node.data.url || 'Sem URL')}
                            {node.type === 'end' && 'Fim do fluxo'}
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={(e) => { e.stopPropagation(); deleteNode(node.id) }}
                        className="p-1 text-gray-400 hover:text-red-500"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>

          {/* Add node button */}
          <div className="flex justify-center mt-4">
            {nodes.length > 0 && (
              <div className="w-0.5 h-6 bg-gray-300 mx-auto mb-2"></div>
            )}
          </div>
          <div className="flex justify-center">
            <button
              onClick={() => setShowAddNode(true)}
              className="flex items-center gap-2 px-4 py-2 border-2 border-dashed border-gray-300 rounded-xl text-gray-500 hover:border-primary-400 hover:text-primary-600 transition-colors"
            >
              <Plus size={16} /> Adicionar bloco
            </button>
          </div>
        </div>

        {/* Right panel - Node config */}
        <div className="w-80 bg-white border-l border-gray-200 p-4 overflow-y-auto">
          {selectedNode ? (
            <NodeEditor
              node={selectedNode}
              onUpdate={(data) => updateNodeData(selectedNode.id, data)}
              onDelete={() => deleteNode(selectedNode.id)}
            />
          ) : (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">Configuração do Fluxo</h3>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Gatilho</label>
                <select
                  value={triggerType}
                  onChange={(e) => setTriggerType(e.target.value)}
                  className="input"
                >
                  <option value="new_conversation">Nova conversa</option>
                  <option value="keyword">Palavra-chave</option>
                  <option value="off_hours">Fora do horário</option>
                  <option value="no_response">Sem resposta do cliente</option>
                  <option value="tag_added">Tag adicionada</option>
                </select>
              </div>

              {triggerType === 'keyword' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Palavra-chave</label>
                  <input
                    type="text"
                    value={triggerValue}
                    onChange={(e) => setTriggerValue(e.target.value)}
                    className="input"
                    placeholder="Ex: oi, olá, preço..."
                  />
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Descrição</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="input resize-none"
                  rows={3}
                  placeholder="Descrição do fluxo..."
                />
              </div>

              <div className="pt-4 border-t">
                <p className="text-xs text-gray-400">
                  💡 Clique em um bloco para editá-lo. Clique em "Adicionar bloco" para expandir o fluxo.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Add Node Modal */}
      {showAddNode && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-sm">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Adicionar bloco</h3>
            <div className="space-y-2">
              {NODE_TYPES.map((nodeType) => {
                const Icon = nodeType.icon
                return (
                  <button
                    key={nodeType.type}
                    onClick={() => addNode(nodeType.type)}
                    className="w-full flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 transition-colors text-left"
                  >
                    <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${nodeType.color}`}>
                      <Icon size={16} />
                    </div>
                    <span className="text-sm font-medium text-gray-700">{nodeType.label}</span>
                  </button>
                )
              })}
            </div>
            <button onClick={() => setShowAddNode(false)} className="btn-secondary w-full mt-4">
              Cancelar
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

// Node Editor Component
function NodeEditor({ node, onUpdate, onDelete }: { node: FlowNode; onUpdate: (data: Record<string, any>) => void; onDelete: () => void }) {
  const info = NODE_TYPES.find(t => t.type === node.type)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-semibold text-gray-900">{info?.label}</h3>
        <button onClick={onDelete} className="text-red-500 hover:text-red-700 text-xs">
          <Trash2 size={14} />
        </button>
      </div>

      {node.type === 'send_message' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Mensagem</label>
          <textarea
            value={node.data.message || ''}
            onChange={(e) => onUpdate({ message: e.target.value })}
            className="input resize-none"
            rows={4}
            placeholder="Digite a mensagem que será enviada..."
          />
          <p className="text-xs text-gray-400 mt-1">
            Variáveis: {'{{nome}}'}, {'{{telefone}}'}, {'{{empresa}}'}
          </p>
        </div>
      )}

      {node.type === 'ask_question' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Pergunta</label>
            <textarea
              value={node.data.question || ''}
              onChange={(e) => onUpdate({ question: e.target.value })}
              className="input resize-none"
              rows={3}
              placeholder="Ex: Qual seu nome?"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Opções de resposta</label>
            {(node.data.options || ['']).map((opt: string, i: number) => (
              <div key={i} className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={opt}
                  onChange={(e) => {
                    const newOptions = [...(node.data.options || [''])]
                    newOptions[i] = e.target.value
                    onUpdate({ options: newOptions })
                  }}
                  className="input"
                  placeholder={`Opção ${i + 1}`}
                />
              </div>
            ))}
            <button
              onClick={() => onUpdate({ options: [...(node.data.options || ['']), ''] })}
              className="text-xs text-primary-600 hover:text-primary-700"
            >
              + Adicionar opção
            </button>
          </div>
        </>
      )}

      {node.type === 'condition' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Campo</label>
            <select
              value={node.data.field || 'message'}
              onChange={(e) => onUpdate({ field: e.target.value })}
              className="input"
            >
              <option value="message">Mensagem</option>
              <option value="contact_name">Nome do contato</option>
              <option value="contact_tag">Tag do contato</option>
              <option value="channel">Canal</option>
              <option value="time">Horário</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Operador</label>
            <select
              value={node.data.operator || 'contains'}
              onChange={(e) => onUpdate({ operator: e.target.value })}
              className="input"
            >
              <option value="contains">Contém</option>
              <option value="equals">É igual a</option>
              <option value="starts_with">Começa com</option>
              <option value="not_contains">Não contém</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Valor</label>
            <input
              type="text"
              value={node.data.value || ''}
              onChange={(e) => onUpdate({ value: e.target.value })}
              className="input"
              placeholder="Valor para comparar..."
            />
          </div>
        </>
      )}

      {node.type === 'delay' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Tempo de espera (segundos)</label>
          <input
            type="number"
            value={node.data.seconds || 5}
            onChange={(e) => onUpdate({ seconds: parseInt(e.target.value) || 5 })}
            className="input"
            min={1}
            max={3600}
          />
        </div>
      )}

      {node.type === 'add_tag' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Nome da tag</label>
          <input
            type="text"
            value={node.data.tag_name || ''}
            onChange={(e) => onUpdate({ tag_name: e.target.value })}
            className="input"
            placeholder="Ex: lead-quente, suporte..."
          />
        </div>
      )}

      {node.type === 'call_webhook' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">URL</label>
            <input
              type="text"
              value={node.data.url || ''}
              onChange={(e) => onUpdate({ url: e.target.value })}
              className="input"
              placeholder="https://..."
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Método</label>
            <select
              value={node.data.method || 'POST'}
              onChange={(e) => onUpdate({ method: e.target.value })}
              className="input"
            >
              <option value="POST">POST</option>
              <option value="GET">GET</option>
            </select>
          </div>
        </>
      )}

      {node.type === 'transfer_team' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Time destino</label>
          <input
            type="text"
            value={node.data.team_name || ''}
            onChange={(e) => onUpdate({ team_name: e.target.value })}
            className="input"
            placeholder="Nome do time..."
          />
        </div>
      )}

      {node.type === 'end' && (
        <div>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={node.data.close_conversation || false}
              onChange={(e) => onUpdate({ close_conversation: e.target.checked })}
              className="rounded border-gray-300"
            />
            <span className="text-sm text-gray-700">Resolver conversa ao encerrar</span>
          </label>
        </div>
      )}
    </div>
  )
}
