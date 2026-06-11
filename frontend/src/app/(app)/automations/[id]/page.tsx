'use client'

import { useEffect, useState, useCallback, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  MiniMap,
  addEdge,
  Connection,
  useNodesState,
  useEdgesState,
  MarkerType,
  Panel,
} from 'reactflow'
import 'reactflow/dist/style.css'
import {
  Save,
  ArrowLeft,
  Play,
  Zap,
  MessageSquare,
  HelpCircle,
  GitBranch,
  ArrowRightLeft,
  Tag,
  Globe,
  Clock,
  X,
  Send,
  Image,
  FileText,
  Mic,
  Users,
  UserPlus,
  CheckCircle,
  Bell,
  Trash2,
  Copy,
  Settings,
} from 'lucide-react'

// Node categories and types
const NODE_CATEGORIES = [
  {
    name: 'Gatilhos',
    color: 'blue',
    nodes: [
      { type: 'trigger_new_conversation', label: 'Nova conversa', icon: '💬' },
      { type: 'trigger_inbox_message', label: 'Mensagem na caixa de entrada', icon: '📥' },
      { type: 'trigger_keyword', label: 'Palavra-chave', icon: '🔑' },
      { type: 'trigger_off_hours', label: 'Fora do horário', icon: '🌙' },
      { type: 'trigger_tag_added', label: 'Tag adicionada', icon: '🏷️' },
      { type: 'trigger_funnel_stage', label: 'Etapa do funil', icon: '📊' },
      { type: 'trigger_no_response', label: 'Sem resposta', icon: '⏰' },
      { type: 'trigger_contact_created', label: 'Contato criado', icon: '👤' },
      { type: 'trigger_campaign_replied', label: 'Campanha respondida', icon: '📢' },
    ],
  },
  {
    name: 'Mensagens',
    color: 'green',
    nodes: [
      { type: 'send_text', label: 'Enviar texto', icon: '💬' },
      { type: 'send_image', label: 'Enviar imagem', icon: '🖼️' },
      { type: 'send_audio', label: 'Enviar áudio', icon: '🎵' },
      { type: 'send_document', label: 'Enviar documento', icon: '📄' },
      { type: 'send_video', label: 'Enviar vídeo', icon: '🎬' },
      { type: 'send_buttons', label: 'Botões de resposta', icon: '🔘' },
      { type: 'send_list', label: 'Lista de opções', icon: '📋' },
    ],
  },
  {
    name: 'Perguntas',
    color: 'teal',
    nodes: [
      { type: 'ask_text', label: 'Pergunta aberta', icon: '❓' },
      { type: 'ask_options', label: 'Pergunta com opções', icon: '🔢' },
      { type: 'ask_email', label: 'Pedir e-mail', icon: '📧' },
      { type: 'ask_phone', label: 'Pedir telefone', icon: '📱' },
    ],
  },
  {
    name: 'Condições',
    color: 'yellow',
    nodes: [
      { type: 'condition', label: 'SE / ENTÃO / SENÃO', icon: '🔀' },
      { type: 'condition_business_hours', label: 'Horário comercial', icon: '🕐' },
      { type: 'condition_contact_field', label: 'Campo do contato', icon: '👤' },
      { type: 'condition_tag', label: 'Tem tag?', icon: '🏷️' },
    ],
  },
  {
    name: 'Ações',
    color: 'purple',
    nodes: [
      { type: 'action_add_tag', label: 'Adicionar tag', icon: '🏷️' },
      { type: 'action_remove_tag', label: 'Remover tag', icon: '❌' },
      { type: 'action_assign_agent', label: 'Atribuir atendente', icon: '👤' },
      { type: 'action_transfer_team', label: 'Transferir p/ time', icon: '👥' },
      { type: 'action_update_contact', label: 'Atualizar contato', icon: '✏️' },
      { type: 'action_change_funnel', label: 'Alterar etapa funil', icon: '📊' },
      { type: 'action_create_deal', label: 'Criar oportunidade', icon: '💰' },
      { type: 'action_send_notification', label: 'Notificação interna', icon: '🔔' },
      { type: 'action_close_conversation', label: 'Encerrar conversa', icon: '✅' },
      { type: 'action_webhook', label: 'Webhook externo', icon: '🌐' },
    ],
  },
  {
    name: 'Esperas',
    color: 'gray',
    nodes: [
      { type: 'wait_seconds', label: 'Esperar segundos', icon: '⏱️' },
      { type: 'wait_minutes', label: 'Esperar minutos', icon: '⏳' },
      { type: 'wait_hours', label: 'Esperar horas', icon: '🕐' },
      { type: 'wait_response', label: 'Esperar resposta', icon: '💬' },
      { type: 'wait_business_hours', label: 'Esperar horário', icon: '🏢' },
    ],
  },
]

const CATEGORY_COLORS: Record<string, string> = {
  blue: 'border-blue-500 bg-blue-50',
  green: 'border-green-500 bg-green-50',
  teal: 'border-teal-500 bg-teal-50',
  yellow: 'border-yellow-500 bg-yellow-50',
  purple: 'border-purple-500 bg-purple-50',
  gray: 'border-gray-500 bg-gray-50',
}

const NODE_COLORS: Record<string, string> = {
  trigger: '#3B82F6',
  send: '#10B981',
  ask: '#14B8A6',
  condition: '#F59E0B',
  action: '#8B5CF6',
  wait: '#6B7280',
}

function getNodeColor(type: string): string {
  if (type.startsWith('trigger')) return NODE_COLORS.trigger
  if (type.startsWith('send')) return NODE_COLORS.send
  if (type.startsWith('ask')) return NODE_COLORS.ask
  if (type.startsWith('condition')) return NODE_COLORS.condition
  if (type.startsWith('action')) return NODE_COLORS.action
  if (type.startsWith('wait')) return NODE_COLORS.wait
  return '#6B7280'
}

function getNodeLabel(type: string): string {
  for (const cat of NODE_CATEGORIES) {
    const found = cat.nodes.find(n => n.type === type)
    if (found) return found.label
  }
  return type
}

function getNodeIcon(type: string): string {
  for (const cat of NODE_CATEGORIES) {
    const found = cat.nodes.find(n => n.type === type)
    if (found) return found.icon
  }
  return '⚡'
}

export default function FlowEditorPage() {
  const params = useParams()
  const router = useRouter()
  const flowId = params.id as string
  const isNew = flowId === 'new'

  const [name, setName] = useState('Nova automação')
  const [description, setDescription] = useState('')
  const [botName, setBotName] = useState('Assistente')
  const [isActive, setIsActive] = useState(false)
  const [saving, setSaving] = useState(false)
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [showBlockPanel, setShowBlockPanel] = useState(true)

  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  useEffect(() => {
    if (!isNew) fetchFlow()
  }, [flowId])

  const fetchFlow = async () => {
    try {
      const response = await api.get(`/automation-flows/${flowId}`)
      const flow = response.data
      if (flow) {
        setName(flow.name)
        setDescription(flow.description || '')
        setIsActive(flow.is_active)
        if (flow.nodes) {
          const parsedNodes = typeof flow.nodes === 'string' ? JSON.parse(flow.nodes) : flow.nodes
          setNodes(parsedNodes.map((n: any) => ({
            ...n,
            style: { borderColor: getNodeColor(n.data?.nodeType || n.type || '') },
          })))
        }
        if (flow.edges) {
          const parsedEdges = typeof flow.edges === 'string' ? JSON.parse(flow.edges) : flow.edges
          setEdges(parsedEdges)
        }
      }
    } catch {
      // Try old format
      try {
        const response = await api.get('/bot-flows')
        const flows = response.data.flows || []
        const flow = flows.find((f: any) => f.id === flowId)
        if (flow) {
          setName(flow.name)
          setDescription(flow.description || '')
          setIsActive(flow.is_active)
        }
      } catch {}
    }
  }

  const onConnect = useCallback((connection: Connection) => {
    const edge: Edge = {
      ...connection,
      id: `edge_${Date.now()}`,
      type: 'smoothstep',
      animated: true,
      markerEnd: { type: MarkerType.ArrowClosed },
    } as Edge
    setEdges((eds) => addEdge(edge, eds))
  }, [setEdges])

  const onNodeClick = useCallback((_: any, node: Node) => {
    setSelectedNode(node)
    setShowBlockPanel(false)
  }, [])

  const addNode = (nodeType: string) => {
    const newNode: Node = {
      id: `node_${Date.now()}_${Math.random().toString(36).substr(2, 5)}`,
      type: 'default',
      position: { x: 300, y: (nodes.length + 1) * 150 },
      data: {
        label: `${getNodeIcon(nodeType)} ${getNodeLabel(nodeType)}`,
        nodeType,
        config: {},
      },
      style: {
        borderColor: getNodeColor(nodeType),
        borderWidth: 2,
        borderRadius: 12,
        padding: 12,
        fontSize: 13,
        minWidth: 180,
      },
    }
    setNodes((nds) => [...nds, newNode])
    setSelectedNode(newNode)
    setShowBlockPanel(false)
  }

  const updateNodeConfig = (nodeId: string, config: Record<string, any>) => {
    setNodes((nds) => nds.map(n => {
      if (n.id === nodeId) {
        return { ...n, data: { ...n.data, config: { ...n.data.config, ...config } } }
      }
      return n
    }))
  }

  const deleteSelectedNode = () => {
    if (!selectedNode) return
    setNodes((nds) => nds.filter(n => n.id !== selectedNode.id))
    setEdges((eds) => eds.filter(e => e.source !== selectedNode.id && e.target !== selectedNode.id))
    setSelectedNode(null)
  }

  const saveFlow = async () => {
    if (!name.trim()) {
      toast.error('Nome é obrigatório')
      return
    }
    setSaving(true)
    try {
      const payload = {
        name,
        description,
        trigger_type: nodes.find(n => n.data?.nodeType?.startsWith('trigger'))?.data?.nodeType || 'trigger_new_conversation',
        trigger_value: '',
        is_active: isActive,
        nodes: JSON.stringify(nodes),
        edges: JSON.stringify(edges),
      }

      if (isNew) {
        await api.post('/bot-flows', payload)
      } else {
        await api.put(`/bot-flows/${flowId}`, payload)
      }
      toast.success('Automação salva!')
      router.push('/automations')
    } catch {
      toast.error('Erro ao salvar')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Top bar */}
      <div className="bg-white border-b border-gray-200 px-4 py-2 flex items-center justify-between flex-shrink-0 z-10">
        <div className="flex items-center gap-3">
          <button onClick={() => router.push('/automations')} className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
            <ArrowLeft size={18} />
          </button>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="text-base font-semibold text-gray-900 border-none outline-none bg-transparent w-48"
            placeholder="Nome da automação..."
          />
          <span className="text-gray-300">|</span>
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-gray-400">🤖 Bot:</span>
            <input
              type="text"
              value={botName}
              onChange={(e) => setBotName(e.target.value)}
              className="text-sm text-purple-600 font-medium border-none outline-none bg-transparent w-32"
              placeholder="Nome do bot..."
            />
          </div>
        </div>
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="rounded border-gray-300"
            />
            <span className={isActive ? 'text-green-600 font-medium' : 'text-gray-500'}>
              {isActive ? '🟢 Ativo' : '⚪ Inativo'}
            </span>
          </label>
          <button onClick={saveFlow} disabled={saving} className="btn-primary text-sm py-2">
            <Save size={14} /> {saving ? 'Salvando...' : 'Salvar'}
          </button>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Left panel - Block types */}
        {showBlockPanel && (
          <div className="w-64 bg-white border-r border-gray-200 overflow-y-auto flex-shrink-0">
            <div className="p-3">
              <h3 className="text-xs font-semibold text-gray-400 uppercase mb-2">Blocos</h3>
              {NODE_CATEGORIES.map((category) => (
                <div key={category.name} className="mb-3">
                  <p className="text-xs font-medium text-gray-500 mb-1">{category.name}</p>
                  <div className="space-y-1">
                    {category.nodes.map((node) => (
                      <button
                        key={node.type}
                        onClick={() => addNode(node.type)}
                        className={`w-full text-left px-3 py-2 rounded-lg border text-xs font-medium transition-colors hover:shadow-sm ${CATEGORY_COLORS[category.color]}`}
                      >
                        {node.icon} {node.label}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Canvas */}
        <div className="flex-1 relative">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onPaneClick={() => { setSelectedNode(null); setShowBlockPanel(true) }}
            fitView
            defaultEdgeOptions={{
              type: 'smoothstep',
              animated: true,
              markerEnd: { type: MarkerType.ArrowClosed },
            }}
          >
            <Controls />
            <MiniMap />
            <Background gap={20} size={1} />
            <Panel position="top-left">
              <button
                onClick={() => setShowBlockPanel(!showBlockPanel)}
                className="bg-white px-3 py-1.5 rounded-lg shadow border border-gray-200 text-xs font-medium text-gray-600 hover:bg-gray-50"
              >
                {showBlockPanel ? '◀ Esconder' : '▶ Blocos'}
              </button>
            </Panel>
          </ReactFlow>
        </div>

        {/* Right panel - Node config */}
        {selectedNode && (
          <div className="w-80 bg-white border-l border-gray-200 overflow-y-auto flex-shrink-0">
            <div className="p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">
                  {getNodeIcon(selectedNode.data?.nodeType)} {getNodeLabel(selectedNode.data?.nodeType)}
                </h3>
                <div className="flex gap-1">
                  <button onClick={deleteSelectedNode} className="p-1.5 text-red-400 hover:text-red-600 hover:bg-red-50 rounded">
                    <Trash2 size={14} />
                  </button>
                  <button onClick={() => { setSelectedNode(null); setShowBlockPanel(true) }} className="p-1.5 text-gray-400 hover:text-gray-600">
                    <X size={14} />
                  </button>
                </div>
              </div>

              <NodeConfigPanel
                node={selectedNode}
                onUpdate={(config) => updateNodeConfig(selectedNode.id, config)}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// Node Configuration Panel
function NodeConfigPanel({ node, onUpdate }: { node: Node; onUpdate: (config: Record<string, any>) => void }) {
  const nodeType = node.data?.nodeType || ''
  const config = node.data?.config || {}

  // --- TRIGGER NODES ---
  if (nodeType === 'trigger_inbox_message') {
    const [channels, setChannels] = useState<Array<{id: string; name: string; type: string; status: string}>>([])

    useEffect(() => {
      api.get('/channels').then(res => setChannels(res.data.channels || [])).catch(() => {})
    }, [])

    const filteredChannels = config.channel_type && config.channel_type !== 'any'
      ? channels.filter(c => c.type === config.channel_type)
      : channels

    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Tipo de canal</label>
          <select
            value={config.channel_type || 'any'}
            onChange={(e) => onUpdate({ channel_type: e.target.value, channel_id: '', channel_name: '' })}
            className="input text-sm"
          >
            <option value="any">Qualquer canal</option>
            <option value="whatsapp">WhatsApp</option>
            <option value="email">E-mail</option>
            <option value="webchat">Chat do site</option>
          </select>
        </div>

        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Caixa de entrada</label>
          <select
            value={config.channel_id || ''}
            onChange={(e) => {
              const ch = channels.find(c => c.id === e.target.value)
              onUpdate({ channel_id: e.target.value, channel_name: ch?.name || '' })
            }}
            className="input text-sm"
          >
            <option value="">Todas as caixas de entrada</option>
            {filteredChannels.map(ch => (
              <option key={ch.id} value={ch.id}>
                {ch.name} {ch.status === 'connected' ? '🟢' : '⚪'}
              </option>
            ))}
          </select>
        </div>

        {config.channel_name && (
          <div className="p-2 bg-green-50 rounded-lg text-xs text-green-700">
            ✅ Selecionado: <strong>{config.channel_name}</strong>
          </div>
        )}

        <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
          📥 Este fluxo será ativado quando uma mensagem chegar na caixa de entrada selecionada.
        </div>
      </div>
    )
  }

  if (nodeType === 'trigger_keyword') {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Palavras-chave (separar por vírgula)</label>
          <input
            type="text"
            value={config.keywords || ''}
            onChange={(e) => onUpdate({ keywords: e.target.value })}
            className="input text-sm"
            placeholder="oi, olá, preço, boleto..."
          />
        </div>
      </div>
    )
  }

  // --- MESSAGE NODES ---
  if (nodeType === 'send_text') {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Mensagem</label>
          <textarea
            value={config.message || ''}
            onChange={(e) => onUpdate({ message: e.target.value })}
            className="input text-sm resize-none"
            rows={5}
            placeholder="Olá {{nome}}, como posso ajudar?"
          />
          <p className="text-xs text-gray-400 mt-1">
            Variáveis: {'{{nome}}'}, {'{{telefone}}'}, {'{{email}}'}, {'{{empresa}}'}, {'{{atendente}}'}, {'{{protocolo}}'}
          </p>
        </div>
      </div>
    )
  }

  if (nodeType === 'send_buttons') {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Mensagem</label>
          <textarea
            value={config.message || ''}
            onChange={(e) => onUpdate({ message: e.target.value })}
            className="input text-sm resize-none"
            rows={3}
            placeholder="Escolha uma opção:"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Botões</label>
          {(config.buttons || ['', '', '']).map((btn: string, i: number) => (
            <input
              key={i}
              type="text"
              value={btn}
              onChange={(e) => {
                const newButtons = [...(config.buttons || ['', '', ''])]
                newButtons[i] = e.target.value
                onUpdate({ buttons: newButtons })
              }}
              className="input text-sm mb-1"
              placeholder={`Botão ${i + 1}`}
            />
          ))}
        </div>
      </div>
    )
  }

  // --- ASK NODES ---
  if (nodeType.startsWith('ask')) {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Pergunta</label>
          <textarea
            value={config.question || ''}
            onChange={(e) => onUpdate({ question: e.target.value })}
            className="input text-sm resize-none"
            rows={3}
            placeholder="Qual seu nome completo?"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Salvar resposta em</label>
          <input
            type="text"
            value={config.save_as || ''}
            onChange={(e) => onUpdate({ save_as: e.target.value })}
            className="input text-sm"
            placeholder="nome_completo"
          />
          <p className="text-xs text-gray-400 mt-1">Use depois como {'{{nome_completo}}'}</p>
        </div>
        {nodeType === 'ask_options' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Opções</label>
            {(config.options || ['']).map((opt: string, i: number) => (
              <input
                key={i}
                type="text"
                value={opt}
                onChange={(e) => {
                  const newOpts = [...(config.options || [''])]
                  newOpts[i] = e.target.value
                  onUpdate({ options: newOpts })
                }}
                className="input text-sm mb-1"
                placeholder={`Opção ${i + 1}`}
              />
            ))}
            <button
              onClick={() => onUpdate({ options: [...(config.options || ['']), ''] })}
              className="text-xs text-primary-600"
            >
              + Adicionar opção
            </button>
          </div>
        )}
      </div>
    )
  }

  // --- CONDITION NODES ---
  if (nodeType === 'condition') {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">SE (campo)</label>
          <select
            value={config.field || 'message'}
            onChange={(e) => onUpdate({ field: e.target.value })}
            className="input text-sm"
          >
            <option value="message">Mensagem recebida</option>
            <option value="contact_name">Nome do contato</option>
            <option value="contact_tag">Tag do contato</option>
            <option value="funnel_stage">Etapa do funil</option>
            <option value="channel">Canal</option>
            <option value="team">Time</option>
            <option value="agent">Atendente</option>
            <option value="day_of_week">Dia da semana</option>
            <option value="hour">Horário</option>
            <option value="custom_variable">Variável customizada</option>
          </select>
        </div>
        {config.field === 'custom_variable' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Nome da variável</label>
            <input
              type="text"
              value={config.variable_name || ''}
              onChange={(e) => onUpdate({ variable_name: e.target.value })}
              className="input text-sm"
              placeholder="interesse_cliente"
            />
          </div>
        )}
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Operador</label>
          <select
            value={config.operator || 'contains'}
            onChange={(e) => onUpdate({ operator: e.target.value })}
            className="input text-sm"
          >
            <option value="equals">É igual a</option>
            <option value="not_equals">É diferente de</option>
            <option value="contains">Contém</option>
            <option value="not_contains">Não contém</option>
            <option value="starts_with">Começa com</option>
            <option value="ends_with">Termina com</option>
            <option value="greater_than">Maior que</option>
            <option value="less_than">Menor que</option>
            <option value="is_empty">Está vazio</option>
            <option value="is_not_empty">Não está vazio</option>
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Valor</label>
          <input
            type="text"
            value={config.value || ''}
            onChange={(e) => onUpdate({ value: e.target.value })}
            className="input text-sm"
            placeholder="Valor para comparar..."
          />
        </div>
        <div className="p-3 bg-yellow-50 rounded-lg text-xs text-yellow-700">
          💡 Conecte duas saídas deste bloco:<br/>
          • Saída 1 → ENTÃO (condição verdadeira)<br/>
          • Saída 2 → SENÃO (condição falsa)
        </div>
      </div>
    )
  }

  if (nodeType === 'condition_business_hours') {
    return (
      <div className="p-3 bg-gray-50 rounded-lg text-xs text-gray-600">
        Verifica automaticamente se está dentro do horário comercial configurado na empresa.
        <br/><br/>
        • Saída 1 → Dentro do horário<br/>
        • Saída 2 → Fora do horário
      </div>
    )
  }

  // --- ACTION NODES ---
  if (nodeType === 'action_add_tag' || nodeType === 'action_remove_tag') {
    return (
      <div>
        <label className="block text-xs font-medium text-gray-600 mb-1">Nome da tag</label>
        <input
          type="text"
          value={config.tag_name || ''}
          onChange={(e) => onUpdate({ tag_name: e.target.value })}
          className="input text-sm"
          placeholder="lead-quente"
        />
      </div>
    )
  }

  if (nodeType === 'action_transfer_team') {
    return (
      <div>
        <label className="block text-xs font-medium text-gray-600 mb-1">Nome do time</label>
        <input
          type="text"
          value={config.team_name || ''}
          onChange={(e) => onUpdate({ team_name: e.target.value })}
          className="input text-sm"
          placeholder="Comercial, Suporte, Financeiro..."
        />
      </div>
    )
  }

  if (nodeType === 'action_assign_agent') {
    return (
      <div>
        <label className="block text-xs font-medium text-gray-600 mb-1">E-mail do atendente</label>
        <input
          type="text"
          value={config.agent_email || ''}
          onChange={(e) => onUpdate({ agent_email: e.target.value })}
          className="input text-sm"
          placeholder="atendente@empresa.com"
        />
      </div>
    )
  }

  if (nodeType === 'action_update_contact') {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Campo</label>
          <select
            value={config.contact_field || 'name'}
            onChange={(e) => onUpdate({ contact_field: e.target.value })}
            className="input text-sm"
          >
            <option value="name">Nome</option>
            <option value="email">E-mail</option>
            <option value="company_name">Empresa</option>
            <option value="city">Cidade</option>
            <option value="notes">Observações</option>
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Valor</label>
          <input
            type="text"
            value={config.contact_value || ''}
            onChange={(e) => onUpdate({ contact_value: e.target.value })}
            className="input text-sm"
            placeholder="Pode usar {{variavel}}"
          />
        </div>
      </div>
    )
  }

  if (nodeType === 'action_webhook') {
    return (
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">URL</label>
          <input
            type="text"
            value={config.url || ''}
            onChange={(e) => onUpdate({ url: e.target.value })}
            className="input text-sm"
            placeholder="https://..."
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Método</label>
          <select
            value={config.method || 'POST'}
            onChange={(e) => onUpdate({ method: e.target.value })}
            className="input text-sm"
          >
            <option value="POST">POST</option>
            <option value="GET">GET</option>
            <option value="PUT">PUT</option>
          </select>
        </div>
        <p className="text-xs text-gray-400">Envia dados do contato, conversa e variáveis do fluxo.</p>
      </div>
    )
  }

  if (nodeType === 'action_close_conversation') {
    return (
      <div className="p-3 bg-green-50 rounded-lg text-xs text-green-700">
        ✅ Marca a conversa como resolvida e encerra o fluxo.
      </div>
    )
  }

  // --- WAIT NODES ---
  if (nodeType.startsWith('wait')) {
    return (
      <div className="space-y-3">
        {nodeType === 'wait_seconds' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Segundos</label>
            <input
              type="number"
              value={config.seconds || 5}
              onChange={(e) => onUpdate({ seconds: parseInt(e.target.value) || 5 })}
              className="input text-sm"
              min={1} max={300}
            />
          </div>
        )}
        {nodeType === 'wait_minutes' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Minutos</label>
            <input
              type="number"
              value={config.minutes || 5}
              onChange={(e) => onUpdate({ minutes: parseInt(e.target.value) || 5 })}
              className="input text-sm"
              min={1} max={1440}
            />
          </div>
        )}
        {nodeType === 'wait_hours' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Horas</label>
            <input
              type="number"
              value={config.hours || 1}
              onChange={(e) => onUpdate({ hours: parseInt(e.target.value) || 1 })}
              className="input text-sm"
              min={1} max={168}
            />
          </div>
        )}
        {nodeType === 'wait_response' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Tempo máximo de espera (minutos)</label>
            <input
              type="number"
              value={config.timeout_minutes || 10}
              onChange={(e) => onUpdate({ timeout_minutes: parseInt(e.target.value) || 10 })}
              className="input text-sm"
              min={1} max={1440}
            />
            <p className="text-xs text-gray-400 mt-1">
              Se não responder, segue pela saída "timeout"
            </p>
          </div>
        )}
        {nodeType === 'wait_business_hours' && (
          <div className="p-3 bg-gray-50 rounded-lg text-xs text-gray-600">
            Pausa o fluxo e retoma quando o horário comercial iniciar.
          </div>
        )}
      </div>
    )
  }

  // Default
  return (
    <div className="p-3 bg-gray-50 rounded-lg text-xs text-gray-500">
      Selecione um bloco para configurar.
    </div>
  )
}
