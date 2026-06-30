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
  {
    name: 'GLPI - Chamados',
    color: 'orange',
    nodes: [
      { type: 'glpi_open_ticket', label: 'Abrir chamado GLPI', icon: '🎫' },
      { type: 'glpi_check_status', label: 'Consultar chamado GLPI', icon: '🔍' },
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
  orange: 'border-orange-500 bg-orange-50',
}

const NODE_COLORS: Record<string, string> = {
  trigger: '#3B82F6',
  send: '#10B981',
  ask: '#14B8A6',
  condition: '#F59E0B',
  action: '#8B5CF6',
  wait: '#6B7280',
  glpi: '#F97316',
}

type AutomationUser = {
  id: string
  name: string
  email: string
  is_active: boolean
  availability_status?: string
  role_name?: string | null
}

type AutomationTeam = {
  id: string
  name: string
  is_active: boolean
  member_count?: number
}

function getNodeColor(type: string): string {
  if (type.startsWith('glpi')) return NODE_COLORS.glpi
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

function getDefaultFlowPriority(triggerType: string): number {
  if (triggerType === 'off_hours' || triggerType === 'trigger_off_hours') return 100
  if (triggerType === 'no_response' || triggerType === 'trigger_no_response') return 80
  if (triggerType === 'new_conversation' || triggerType === 'trigger_new_conversation' || triggerType === 'trigger_inbox_message') return 10
  return 20
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
  const [priority, setPriority] = useState(10)
  const [stopOnMatch, setStopOnMatch] = useState(true)
  const [saving, setSaving] = useState(false)
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [showBlockPanel, setShowBlockPanel] = useState(true)
  const [users, setUsers] = useState<AutomationUser[]>([])
  const [teams, setTeams] = useState<AutomationTeam[]>([])

  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  useEffect(() => {
    if (!isNew) fetchFlow()
    fetchAutomationOptions()
  }, [flowId])

  const fetchAutomationOptions = async () => {
    try {
      const [usersResponse, teamsResponse] = await Promise.all([
        api.get('/users'),
        api.get('/teams'),
      ])
      setUsers((usersResponse.data.users || []).filter((user: AutomationUser) => user.is_active))
      setTeams((teamsResponse.data.teams || []).filter((team: AutomationTeam) => team.is_active))
    } catch {
      setUsers([])
      setTeams([])
    }
  }

  const fetchFlow = async () => {
    try {
      const response = await api.get(`/bot-flows/${flowId}`)
      const flow = response.data
      if (flow) {
        setName(flow.name || '')
        setDescription(flow.description || '')
        setIsActive(flow.is_active || false)
        setPriority(flow.priority ?? getDefaultFlowPriority(flow.trigger_type || 'trigger_new_conversation'))
        setStopOnMatch(flow.stop_on_match ?? true)

        // Parse nodes - handle double-encoded strings
        let parsedNodes: any[] = []
        if (flow.nodes) {
          let nodesData = flow.nodes
          // If it's a string, parse it
          if (typeof nodesData === 'string') {
            try { nodesData = JSON.parse(nodesData) } catch {}
          }
          // If still a string (double encoded), parse again
          if (typeof nodesData === 'string') {
            try { nodesData = JSON.parse(nodesData) } catch {}
          }
          if (Array.isArray(nodesData)) {
            parsedNodes = nodesData.map((n: any) => ({
              ...n,
              style: {
                borderColor: getNodeColor(n.data?.nodeType || ''),
                borderWidth: 2,
                borderRadius: 12,
                padding: 12,
                fontSize: 13,
                minWidth: 180,
              },
            }))
          }
        }
        setNodes(parsedNodes)

        // Parse edges
        let parsedEdges: any[] = []
        if (flow.edges) {
          let edgesData = flow.edges
          if (typeof edgesData === 'string') {
            try { edgesData = JSON.parse(edgesData) } catch {}
          }
          if (typeof edgesData === 'string') {
            try { edgesData = JSON.parse(edgesData) } catch {}
          }
          if (Array.isArray(edgesData)) {
            parsedEdges = edgesData
          }
        }
        setEdges(parsedEdges)
      }
    } catch (err) {
      console.error('Error loading flow:', err)
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
        const updated = { ...n, data: { ...n.data, config: { ...n.data.config, ...config } } }
        return updated
      }
      return n
    }))
  }

  // Derive the current selected node from the nodes array (always up to date)
  const currentSelectedNode = selectedNode ? nodes.find(n => n.id === selectedNode.id) || selectedNode : null

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
      const triggerNode = nodes.find(n => n.data?.nodeType?.startsWith('trigger'))
      const triggerConfig = triggerNode?.data?.config || {}
      const payload = {
        name,
        description,
        trigger_type: triggerNode?.data?.nodeType || 'trigger_new_conversation',
        trigger_value: triggerConfig.keywords || triggerConfig.channel_id || '',
        is_active: isActive,
        priority,
        stop_on_match: stopOnMatch,
        nodes: nodes,
        edges: edges,
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
    <div className="automation-builder h-screen flex flex-col">
      <style jsx global>{`
        .dark .automation-builder .bg-white {
          background-color: rgb(17 24 39) !important;
        }
        .dark .automation-builder .bg-gray-50,
        .dark .automation-builder .bg-gray-100 {
          background-color: rgb(31 41 55) !important;
        }
        .dark .automation-builder .bg-gray-200 {
          background-color: rgb(55 65 81) !important;
        }
        .dark .automation-builder .bg-blue-50,
        .dark .automation-builder .bg-green-50,
        .dark .automation-builder .bg-teal-50,
        .dark .automation-builder .bg-yellow-50,
        .dark .automation-builder .bg-purple-50,
        .dark .automation-builder .bg-orange-50,
        .dark .automation-builder .bg-red-50 {
          background-color: rgb(31 41 55) !important;
        }
        .dark .automation-builder .border-teal-500 {
          border-color: rgb(20 184 166) !important;
        }
        .dark .automation-builder .border-yellow-500 {
          border-color: rgb(234 179 8) !important;
        }
        .dark .automation-builder .text-gray-900 {
          color: rgb(249 250 251) !important;
        }
        .dark .automation-builder .text-gray-700,
        .dark .automation-builder .text-gray-600 {
          color: rgb(209 213 219) !important;
        }
        .dark .automation-builder .text-gray-500,
        .dark .automation-builder .text-gray-400 {
          color: rgb(156 163 175) !important;
        }
        .dark .automation-builder .text-blue-600 {
          color: rgb(96 165 250) !important;
        }
        .dark .automation-builder .text-green-600,
        .dark .automation-builder .text-green-700 {
          color: rgb(74 222 128) !important;
        }
        .dark .automation-builder .text-purple-600 {
          color: rgb(192 132 252) !important;
        }
        .dark .automation-builder .text-teal-600,
        .dark .automation-builder .text-teal-700 {
          color: rgb(45 212 191) !important;
        }
        .dark .automation-builder .text-yellow-600,
        .dark .automation-builder .text-yellow-700 {
          color: rgb(250 204 21) !important;
        }
        .dark .automation-builder .text-orange-600,
        .dark .automation-builder .text-orange-700 {
          color: rgb(251 146 60) !important;
        }
        .dark .automation-builder .text-red-400,
        .dark .automation-builder .text-red-600 {
          color: rgb(248 113 113) !important;
        }
        .dark .automation-builder .border-gray-200,
        .dark .automation-builder .border-gray-300 {
          border-color: rgb(55 65 81) !important;
        }
        .dark .automation-builder .border-blue-500 {
          border-color: rgb(59 130 246) !important;
        }
        .dark .automation-builder .border-green-500 {
          border-color: rgb(34 197 94) !important;
        }
        .dark .automation-builder .border-purple-500 {
          border-color: rgb(168 85 247) !important;
        }
        .dark .automation-builder .border-orange-500 {
          border-color: rgb(249 115 22) !important;
        }
        .dark .automation-builder .border-gray-500 {
          border-color: rgb(107 114 128) !important;
        }
        .dark .automation-builder .hover\\:bg-gray-50:hover,
        .dark .automation-builder .hover\\:bg-gray-100:hover {
          background-color: rgb(55 65 81) !important;
        }
        .dark .automation-builder input,
        .dark .automation-builder textarea,
        .dark .automation-builder select {
          color: rgb(243 244 246);
          background-color: rgb(31 41 55);
          border-color: rgb(55 65 81);
        }
        .dark .automation-builder input::placeholder,
        .dark .automation-builder textarea::placeholder {
          color: rgb(107 114 128);
        }
        .dark .automation-builder button.border-teal-500 {
          color: rgb(204 251 241);
          background-color: rgb(19 78 74) !important;
        }
        .dark .automation-builder button.border-teal-500:hover {
          background-color: rgb(17 94 89) !important;
        }
        .dark .automation-builder .react-flow__controls,
        .dark .automation-builder .react-flow__minimap {
          background: rgb(17 24 39);
          border-color: rgb(55 65 81);
        }
        .dark .automation-builder .react-flow__controls-button {
          background: rgb(31 41 55);
          border-color: rgb(55 65 81);
          color: rgb(229 231 235);
        }
      `}</style>
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
          <label className="flex items-center gap-1.5 text-xs text-gray-500">
            <span>Prioridade</span>
            <input
              type="number"
              min={1}
              max={999}
              value={priority}
              onChange={(e) => setPriority(Math.max(1, Number(e.target.value) || 1))}
              className="w-16 rounded-lg border border-gray-200 bg-white px-2 py-1 text-sm text-gray-900 outline-none focus:border-primary-500 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100"
              title="Maior prioridade executa antes"
            />
          </label>
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
        {currentSelectedNode && (
          <div className="w-80 bg-white border-l border-gray-200 overflow-y-auto flex-shrink-0">
            <div className="p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">
                  {getNodeIcon(currentSelectedNode.data?.nodeType)} {getNodeLabel(currentSelectedNode.data?.nodeType)}
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
                node={currentSelectedNode}
                users={users}
                teams={teams}
                onUpdate={(config) => updateNodeConfig(currentSelectedNode.id, config)}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// Node Configuration Panel
function NodeConfigPanel({
  node,
  users,
  teams,
  onUpdate,
}: {
  node: Node
  users: AutomationUser[]
  teams: AutomationTeam[]
  onUpdate: (config: Record<string, any>) => void
}) {
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

  if (nodeType === 'trigger_off_hours') {
    return <BusinessHoursConfig config={config} onUpdate={onUpdate} mode="outside" />
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
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-300 mb-1">Pergunta</label>
          <textarea
            value={config.question || ''}
            onChange={(e) => onUpdate({ question: e.target.value })}
            className="input text-sm resize-none dark:bg-gray-800 dark:text-gray-100 dark:border-gray-700 dark:placeholder-gray-500"
            rows={3}
            placeholder="Qual seu nome completo?"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-300 mb-1">Salvar resposta em</label>
          <input
            type="text"
            value={config.save_as || ''}
            onChange={(e) => onUpdate({ save_as: e.target.value })}
            className="input text-sm dark:bg-gray-800 dark:text-gray-100 dark:border-gray-700 dark:placeholder-gray-500"
            placeholder="nome_completo"
          />
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">Use depois como {'{{nome_completo}}'}</p>
        </div>
        {nodeType === 'ask_options' && (
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-300 mb-1">Opções</label>
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
                className="input text-sm mb-1 dark:bg-gray-800 dark:text-gray-100 dark:border-gray-700 dark:placeholder-gray-500"
                placeholder={`Opção ${i + 1}`}
              />
            ))}
            <button
              onClick={() => onUpdate({ options: [...(config.options || ['']), ''] })}
              className="text-xs font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
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
    return <BusinessHoursConfig config={config} onUpdate={onUpdate} mode="condition" />
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
      <div className="space-y-2">
        <label className="block text-xs font-medium text-gray-600 mb-1">Time de destino</label>
        <select
          value={config.team_id || ''}
          onChange={(e) => {
            const team = teams.find(t => t.id === e.target.value)
            onUpdate({ team_id: e.target.value, team_name: team?.name || '' })
          }}
          className="input text-sm"
        >
          <option value="">Selecione um time ativo</option>
          {teams.map((team) => (
            <option key={team.id} value={team.id}>
              {team.name}{typeof team.member_count === 'number' ? ` (${team.member_count} membros)` : ''}
            </option>
          ))}
        </select>
        {teams.length === 0 && (
          <p className="text-xs text-yellow-600">Nenhum time ativo encontrado em Times.</p>
        )}
        {config.team_name && (
          <p className="text-xs text-gray-500">Selecionado: {config.team_name}</p>
        )}
      </div>
    )
  }

  if (nodeType === 'action_assign_agent') {
    return (
      <div className="space-y-2">
        <label className="block text-xs font-medium text-gray-600 mb-1">Atendente de destino</label>
        <select
          value={config.agent_id || ''}
          onChange={(e) => {
            const user = users.find(u => u.id === e.target.value)
            onUpdate({
              agent_id: e.target.value,
              agent_name: user?.name || '',
              agent_email: user?.email || '',
            })
          }}
          className="input text-sm"
        >
          <option value="">Selecione um atendente ativo</option>
          {users.map((user) => (
            <option key={user.id} value={user.id}>
              {user.name} - {user.email}{user.availability_status ? ` (${user.availability_status})` : ''}
            </option>
          ))}
        </select>
        {users.length === 0 && (
          <p className="text-xs text-yellow-600">Nenhum usuário ativo encontrado em Usuários.</p>
        )}
        {config.agent_name && (
          <p className="text-xs text-gray-500">Selecionado: {config.agent_name}</p>
        )}
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

  if (nodeType === 'wait_business_hours') {
    return <BusinessHoursConfig config={config} onUpdate={onUpdate} mode="wait" />
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

  // --- GLPI NODES ---
  if (nodeType === 'glpi_open_ticket') {
    return (
      <div className="space-y-3">
        <div className="p-3 bg-orange-50 rounded-lg text-xs text-orange-700">
          🎫 <strong>Abrir Chamado GLPI</strong>
          <br/><br/>
          Este bloco inicia um fluxo conversacional com o cliente para abertura de chamado no GLPI.
          <br/><br/>
          O bot irá perguntar automaticamente:
          <br/>1. Nome da empresa (busca entidade no GLPI)
          <br/>2. Confirmação da entidade
          <br/>3. Nome completo
          <br/>4. E-mail de contato
          <br/>5. Título do problema
          <br/>6. Descrição detalhada
          <br/><br/>
          Após coletar todas as informações, o chamado será aberto no GLPI e o número do ticket retornado ao cliente.
          <br/><br/>
          📞 O telefone é capturado automaticamente do WhatsApp.
        </div>
        <div className="p-2 bg-yellow-50 rounded-lg text-xs text-yellow-700">
          ⚠️ Requer configuração das variáveis GLPI_BASE_URL, GLPI_APP_TOKEN e GLPI_USER_TOKEN no servidor.
        </div>
      </div>
    )
  }

  if (nodeType === 'glpi_check_status') {
    return (
      <div className="space-y-3">
        <div className="p-3 bg-orange-50 rounded-lg text-xs text-orange-700">
          🔍 <strong>Consultar Chamado GLPI</strong>
          <br/><br/>
          Este bloco permite que o cliente consulte o status de um chamado existente.
          <br/><br/>
          O bot irá:
          <br/>1. Pedir o número do ticket
          <br/>2. Buscar o chamado no GLPI
          <br/>3. Retornar: título, status, data de abertura e última atualização
        </div>
        <div className="p-2 bg-yellow-50 rounded-lg text-xs text-yellow-700">
          ⚠️ Requer configuração das variáveis GLPI_BASE_URL, GLPI_APP_TOKEN e GLPI_USER_TOKEN no servidor.
        </div>
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

function BusinessHoursConfig({
  config,
  onUpdate,
  mode,
}: {
  config: Record<string, any>
  onUpdate: (config: Record<string, any>) => void
  mode: 'outside' | 'condition' | 'wait'
}) {
  const startTime = config.start_time || '08:00'
  const endTime = config.end_time || '18:00'
  const descriptions = {
    outside: 'Este fluxo sera ativado fora do horario abaixo.',
    condition: 'Saida verdadeira: dentro do horario. Saida falsa: fora do horario.',
    wait: 'Pausa o fluxo ate chegar no horario inicial configurado.',
  }

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Inicio</label>
          <input
            type="time"
            value={startTime}
            onChange={(e) => onUpdate({ start_time: e.target.value })}
            className="input text-sm"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Fim</label>
          <input
            type="time"
            value={endTime}
            onChange={(e) => onUpdate({ end_time: e.target.value })}
            className="input text-sm"
          />
        </div>
      </div>
      <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
        Horario atual configurado: <strong>{startTime} as {endTime}</strong>.
        <br />
        {descriptions[mode]}
      </div>
    </div>
  )
}
