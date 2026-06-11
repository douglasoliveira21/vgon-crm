'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  Phone,
  PhoneIncoming,
  PhoneOutgoing,
  PhoneMissed,
  PhoneOff,
  Mic,
  MicOff,
  Pause,
  Play,
  ArrowRightLeft,
  Settings,
  Plus,
  X,
  Clock,
  Users,
  Trash2,
  Volume2,
} from 'lucide-react'

interface CallRecord {
  id: string
  contact_name?: string
  from_number: string
  to_number: string
  call_direction: string
  call_status: string
  duration_seconds: number
  recording_url?: string
  notes?: string
  call_result?: string
  created_at: string
}

interface Extension {
  id: string
  extension_number: string
  display_name: string
  user_id?: string
  status: string
  can_call_external: boolean
}

export default function CallsPage() {
  const [activeTab, setActiveTab] = useState('dialer')
  const [callHistory, setCallHistory] = useState<CallRecord[]>([])
  const [extensions, setExtensions] = useState<Extension[]>([])
  const [queues, setQueues] = useState<Array<{id: string; name: string; strategy: string}>>([])
  const [loading, setLoading] = useState(true)
  const [showExtensionForm, setShowExtensionForm] = useState(false)
  const [showQueueForm, setShowQueueForm] = useState(false)
  const [extForm, setExtForm] = useState({ display_name: '', extension_number: '', extension_password: '', can_call_external: true, can_receive_calls: true, can_transfer: true, can_access_recordings: false })
  const [queueForm, setQueueForm] = useState({ name: '', strategy: 'ringall', max_wait_time: 120 })

  // Dialer state
  const [phoneNumber, setPhoneNumber] = useState('')
  const [isInCall, setIsInCall] = useState(false)
  const [callDuration, setCallDuration] = useState(0)
  const [isMuted, setIsMuted] = useState(false)
  const [isOnHold, setIsOnHold] = useState(false)
  const [callStatus, setCallStatus] = useState('')

  // Provider config
  const [showConfig, setShowConfig] = useState(false)
  const [providerConfig, setProviderConfig] = useState({
    name: '',
    provider_type: 'sip',
    sip_host: '',
    sip_port: 5060,
    sip_user: '',
    sip_password: '',
    sip_domain: '',
    transport: 'UDP',
    caller_id: '',
    stun_server: 'stun:stun.l.google.com:19302',
    recording_enabled: true,
  })

  useEffect(() => {
    fetchCallHistory()
    fetchExtensions()
    fetchQueues()
  }, [])

  useEffect(() => {
    let interval: NodeJS.Timeout
    if (isInCall) {
      interval = setInterval(() => setCallDuration(d => d + 1), 1000)
    }
    return () => clearInterval(interval)
  }, [isInCall])

  const fetchCallHistory = async () => {
    try {
      // Use existing calls endpoint or generic
      setLoading(false)
    } catch {
      setLoading(false)
    }
  }

  const fetchExtensions = async () => {
    try {
      const res = await api.get('/telephony/extensions')
      setExtensions(res.data.extensions || [])
    } catch {}
  }

  const fetchQueues = async () => {
    try {
      const res = await api.get('/telephony/queues')
      setQueues(res.data.queues || [])
    } catch {}
  }

  const createExtension = async () => {
    if (!extForm.display_name || !extForm.extension_number) { toast.error('Preencha nome e número'); return }
    try {
      await api.post('/telephony/extensions', extForm)
      toast.success('Ramal criado!')
      setShowExtensionForm(false)
      setExtForm({ display_name: '', extension_number: '', extension_password: '', can_call_external: true, can_receive_calls: true, can_transfer: true, can_access_recordings: false })
      fetchExtensions()
    } catch { toast.error('Erro ao criar ramal') }
  }

  const deleteExtension = async (id: string) => {
    if (!confirm('Remover este ramal?')) return
    try {
      await api.delete(`/telephony/extensions/${id}`)
      setExtensions(prev => prev.filter(e => e.id !== id))
      toast.success('Ramal removido')
    } catch { toast.error('Erro ao remover') }
  }

  const createQueue = async () => {
    if (!queueForm.name) { toast.error('Nome é obrigatório'); return }
    try {
      await api.post('/telephony/queues', queueForm)
      toast.success('Fila criada!')
      setShowQueueForm(false)
      setQueueForm({ name: '', strategy: 'ringall', max_wait_time: 120 })
      fetchQueues()
    } catch { toast.error('Erro ao criar fila') }
  }

  const deleteQueue = async (id: string) => {
    if (!confirm('Remover esta fila?')) return
    try {
      await api.delete(`/telephony/queues/${id}`)
      setQueues(prev => prev.filter(q => q.id !== id))
      toast.success('Fila removida')
    } catch { toast.error('Erro ao remover') }
  }

  const startCall = async () => {
    if (!phoneNumber.trim()) { toast.error('Digite um número'); return }
    setIsInCall(true)
    setCallDuration(0)
    setCallStatus('Chamando...')

    // In a real implementation, this would initiate a WebRTC/SIP call
    toast.success(`Ligando para ${phoneNumber}...`)

    // Simulate connection after 3 seconds
    setTimeout(() => setCallStatus('Em chamada'), 3000)
  }

  const endCall = () => {
    setIsInCall(false)
    setCallStatus('')
    setCallDuration(0)
    toast.success('Chamada encerrada')
  }

  const toggleMute = () => setIsMuted(!isMuted)
  const toggleHold = () => setIsOnHold(!isOnHold)

  const formatDuration = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }

  const dialPad = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '*', '0', '#']

  const saveProviderConfig = async () => {
    try {
      await api.post('/telephony/provider', providerConfig)
      toast.success('Configuração salva!')
      setShowConfig(false)
    } catch {
      toast.error('Erro ao salvar')
    }
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Telefonia</h1>
          <p className="text-gray-500 mt-1">Chamadas VoIP, ramais e histórico</p>
        </div>
        <button onClick={() => setShowConfig(true)} className="btn-secondary">
          <Settings size={18} /> Configurar
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {[
          { id: 'dialer', label: '📞 Discador' },
          { id: 'history', label: '📋 Histórico' },
          { id: 'extensions', label: '📱 Ramais' },
          { id: 'queues', label: '👥 Filas' },
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              activeTab === tab.id ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Dialer Tab */}
      {activeTab === 'dialer' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Softphone */}
          <div className="card p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4 text-center">Softphone</h3>

            {/* Call status */}
            {isInCall && (
              <div className="text-center mb-4">
                <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-2 animate-pulse">
                  <Phone size={28} className="text-green-600" />
                </div>
                <p className="text-sm text-gray-600">{callStatus}</p>
                <p className="text-2xl font-mono font-bold text-gray-900 mt-1">{formatDuration(callDuration)}</p>
                <p className="text-sm text-gray-500">{phoneNumber}</p>
              </div>
            )}

            {/* Phone number input */}
            {!isInCall && (
              <>
                <input
                  type="tel"
                  value={phoneNumber}
                  onChange={(e) => setPhoneNumber(e.target.value)}
                  placeholder="Digite o número..."
                  className="input text-center text-lg font-mono mb-4"
                />

                {/* Dial pad */}
                <div className="grid grid-cols-3 gap-2 mb-4">
                  {dialPad.map(key => (
                    <button
                      key={key}
                      onClick={() => setPhoneNumber(prev => prev + key)}
                      className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-lg font-semibold text-gray-700 transition-colors"
                    >
                      {key}
                    </button>
                  ))}
                </div>
              </>
            )}

            {/* Call controls */}
            <div className="flex items-center justify-center gap-3">
              {isInCall ? (
                <>
                  <button
                    onClick={toggleMute}
                    className={`p-3 rounded-full transition-colors ${isMuted ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}
                    title={isMuted ? 'Desmutar' : 'Mutar'}
                  >
                    {isMuted ? <MicOff size={20} /> : <Mic size={20} />}
                  </button>
                  <button
                    onClick={toggleHold}
                    className={`p-3 rounded-full transition-colors ${isOnHold ? 'bg-yellow-100 text-yellow-600' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}
                    title={isOnHold ? 'Retomar' : 'Espera'}
                  >
                    {isOnHold ? <Play size={20} /> : <Pause size={20} />}
                  </button>
                  <button
                    onClick={endCall}
                    className="p-4 bg-red-600 text-white rounded-full hover:bg-red-700"
                    title="Desligar"
                  >
                    <PhoneOff size={24} />
                  </button>
                  <button
                    className="p-3 bg-gray-100 text-gray-600 hover:bg-gray-200 rounded-full"
                    title="Transferir"
                  >
                    <ArrowRightLeft size={20} />
                  </button>
                </>
              ) : (
                <>
                  <button
                    onClick={() => setPhoneNumber(prev => prev.slice(0, -1))}
                    className="p-3 bg-gray-100 text-gray-600 hover:bg-gray-200 rounded-full"
                    title="Apagar"
                  >
                    <X size={20} />
                  </button>
                  <button
                    onClick={startCall}
                    disabled={!phoneNumber.trim()}
                    className="p-4 bg-green-600 text-white rounded-full hover:bg-green-700 disabled:opacity-50"
                    title="Ligar"
                  >
                    <Phone size={24} />
                  </button>
                </>
              )}
            </div>
          </div>

          {/* Recent calls */}
          <div className="card p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Chamadas recentes</h3>
            <div className="space-y-3">
              {callHistory.length === 0 && (
                <p className="text-sm text-gray-400 text-center py-8">Nenhuma chamada registrada</p>
              )}
              {callHistory.map(call => (
                <div key={call.id} className="flex items-center gap-3 p-3 bg-gray-50 rounded-lg">
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                    call.call_direction === 'outbound' ? 'bg-blue-100' :
                    call.call_status === 'missed' ? 'bg-red-100' : 'bg-green-100'
                  }`}>
                    {call.call_direction === 'outbound' ? <PhoneOutgoing size={14} className="text-blue-600" /> :
                     call.call_status === 'missed' ? <PhoneMissed size={14} className="text-red-600" /> :
                     <PhoneIncoming size={14} className="text-green-600" />}
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">{call.contact_name || call.to_number || call.from_number}</p>
                    <p className="text-xs text-gray-400">{formatDuration(call.duration_seconds)}</p>
                  </div>
                  {call.recording_url && (
                    <button className="p-1 text-gray-400 hover:text-primary-600">
                      <Volume2 size={14} />
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* History Tab */}
      {activeTab === 'history' && (
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Histórico de Chamadas</h3>
          <div className="text-center py-12">
            <Clock size={40} className="text-gray-300 mx-auto mb-4" />
            <p className="text-gray-500">O histórico será preenchido conforme as chamadas forem realizadas.</p>
            <p className="text-sm text-gray-400 mt-2">Configure um provedor SIP para começar.</p>
          </div>
        </div>
      )}

      {/* Extensions Tab */}
      {activeTab === 'extensions' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Ramais</h3>
            <button onClick={() => setShowExtensionForm(true)} className="btn-primary text-sm">
              <Plus size={16} /> Novo ramal
            </button>
          </div>

          {extensions.length > 0 ? (
            <div className="space-y-3">
              {extensions.map(ext => (
                <div key={ext.id} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                      ext.status === 'online' ? 'bg-green-100' : ext.status === 'busy' ? 'bg-yellow-100' : 'bg-gray-100'
                    }`}>
                      <Phone size={18} className={
                        ext.status === 'online' ? 'text-green-600' : ext.status === 'busy' ? 'text-yellow-600' : 'text-gray-400'
                      } />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">{ext.display_name}</p>
                      <p className="text-xs text-gray-500">Ramal: {ext.extension_number}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`badge ${
                      ext.status === 'online' ? 'badge-green' : ext.status === 'busy' ? 'badge-yellow' : 'badge-gray'
                    }`}>
                      {ext.status === 'online' ? 'Online' : ext.status === 'busy' ? 'Em chamada' : 'Offline'}
                    </span>
                    <button onClick={() => deleteExtension(ext.id)} className="p-1.5 text-gray-400 hover:text-red-500">
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <Phone size={40} className="text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500 mb-2">Nenhum ramal configurado</p>
              <p className="text-sm text-gray-400">Crie ramais para seus atendentes realizarem e receberem chamadas.</p>
              <button onClick={() => setShowExtensionForm(true)} className="btn-primary inline-flex mt-4 text-sm">
                <Plus size={16} /> Criar primeiro ramal
              </button>
            </div>
          )}
        </div>
      )}

      {/* Queues Tab */}
      {activeTab === 'queues' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Filas de Atendimento</h3>
            <button onClick={() => setShowQueueForm(true)} className="btn-primary text-sm">
              <Plus size={16} /> Nova fila
            </button>
          </div>

          {queues.length > 0 ? (
            <div className="space-y-3">
              {queues.map(queue => (
                <div key={queue.id} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-purple-100 rounded-full flex items-center justify-center">
                      <Users size={18} className="text-purple-600" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">{queue.name}</p>
                      <p className="text-xs text-gray-500">Estratégia: {
                        queue.strategy === 'ringall' ? 'Tocar todos' :
                        queue.strategy === 'roundrobin' ? 'Round Robin' :
                        queue.strategy === 'leastrecent' ? 'Menos recente' : queue.strategy
                      }</p>
                    </div>
                  </div>
                  <button onClick={() => deleteQueue(queue.id)} className="p-1.5 text-gray-400 hover:text-red-500">
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <Users size={40} className="text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500 mb-2">Nenhuma fila configurada</p>
              <p className="text-sm text-gray-400">Crie filas para distribuir chamadas entre seus atendentes.</p>
              <button onClick={() => setShowQueueForm(true)} className="btn-primary inline-flex mt-4 text-sm">
                <Plus size={16} /> Criar primeira fila
              </button>
            </div>
          )}
        </div>
      )}

      {/* Extension Form Modal */}
      {showExtensionForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Novo Ramal</h3>
              <button onClick={() => setShowExtensionForm(false)} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nome do atendente</label>
                <input type="text" value={extForm.display_name} onChange={(e) => setExtForm({...extForm, display_name: e.target.value})} className="input" placeholder="João Silva" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Número do ramal</label>
                  <input type="text" value={extForm.extension_number} onChange={(e) => setExtForm({...extForm, extension_number: e.target.value})} className="input" placeholder="1001" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Senha do ramal</label>
                  <input type="password" value={extForm.extension_password} onChange={(e) => setExtForm({...extForm, extension_password: e.target.value})} className="input" placeholder="••••••" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Permissões</label>
                <div className="space-y-2">
                  <label className="flex items-center gap-2"><input type="checkbox" checked={extForm.can_call_external} onChange={(e) => setExtForm({...extForm, can_call_external: e.target.checked})} className="rounded border-gray-300" /><span className="text-sm text-gray-700">Realizar chamadas externas</span></label>
                  <label className="flex items-center gap-2"><input type="checkbox" checked={extForm.can_receive_calls} onChange={(e) => setExtForm({...extForm, can_receive_calls: e.target.checked})} className="rounded border-gray-300" /><span className="text-sm text-gray-700">Receber chamadas</span></label>
                  <label className="flex items-center gap-2"><input type="checkbox" checked={extForm.can_transfer} onChange={(e) => setExtForm({...extForm, can_transfer: e.target.checked})} className="rounded border-gray-300" /><span className="text-sm text-gray-700">Transferir chamadas</span></label>
                  <label className="flex items-center gap-2"><input type="checkbox" checked={extForm.can_access_recordings} onChange={(e) => setExtForm({...extForm, can_access_recordings: e.target.checked})} className="rounded border-gray-300" /><span className="text-sm text-gray-700">Ouvir gravações</span></label>
                </div>
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button onClick={() => setShowExtensionForm(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={createExtension} className="btn-primary flex-1">Criar ramal</button>
            </div>
          </div>
        </div>
      )}

      {/* Queue Form Modal */}
      {showQueueForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Nova Fila</h3>
              <button onClick={() => setShowQueueForm(false)} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nome da fila</label>
                <input type="text" value={queueForm.name} onChange={(e) => setQueueForm({...queueForm, name: e.target.value})} className="input" placeholder="Suporte, Comercial, Financeiro..." />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Estratégia de toque</label>
                <select value={queueForm.strategy} onChange={(e) => setQueueForm({...queueForm, strategy: e.target.value})} className="input">
                  <option value="ringall">Tocar todos ao mesmo tempo</option>
                  <option value="roundrobin">Round Robin (revezamento)</option>
                  <option value="leastrecent">Menos recente (quem não atende há mais tempo)</option>
                  <option value="random">Aleatório</option>
                  <option value="fewestcalls">Menos chamadas</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Tempo máximo de espera (segundos)</label>
                <input type="number" value={queueForm.max_wait_time} onChange={(e) => setQueueForm({...queueForm, max_wait_time: parseInt(e.target.value) || 120})} className="input" />
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button onClick={() => setShowQueueForm(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={createQueue} className="btn-primary flex-1">Criar fila</button>
            </div>
          </div>
        </div>
      )}

      {/* Provider Config Modal */}
      {showConfig && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Configurar Telefonia SIP</h3>
              <button onClick={() => setShowConfig(false)} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nome da integração</label>
                <input
                  type="text"
                  value={providerConfig.name}
                  onChange={(e) => setProviderConfig({...providerConfig, name: e.target.value})}
                  className="input"
                  placeholder="Minha Operadora VoIP"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Tipo de provedor</label>
                <select
                  value={providerConfig.provider_type}
                  onChange={(e) => setProviderConfig({...providerConfig, provider_type: e.target.value})}
                  className="input"
                >
                  <option value="sip">SIP Personalizado</option>
                  <option value="asterisk">Asterisk</option>
                  <option value="freeswitch">FreeSWITCH</option>
                  <option value="twilio">Twilio</option>
                  <option value="zenvia">Zenvia</option>
                  <option value="totalvoice">TotalVoice</option>
                </select>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Servidor SIP</label>
                  <input
                    type="text"
                    value={providerConfig.sip_host}
                    onChange={(e) => setProviderConfig({...providerConfig, sip_host: e.target.value})}
                    className="input"
                    placeholder="sip.provedor.com"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Porta</label>
                  <input
                    type="number"
                    value={providerConfig.sip_port}
                    onChange={(e) => setProviderConfig({...providerConfig, sip_port: parseInt(e.target.value)})}
                    className="input"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Usuário SIP</label>
                  <input
                    type="text"
                    value={providerConfig.sip_user}
                    onChange={(e) => setProviderConfig({...providerConfig, sip_user: e.target.value})}
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Senha SIP</label>
                  <input
                    type="password"
                    value={providerConfig.sip_password}
                    onChange={(e) => setProviderConfig({...providerConfig, sip_password: e.target.value})}
                    className="input"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Domínio / Realm</label>
                <input
                  type="text"
                  value={providerConfig.sip_domain}
                  onChange={(e) => setProviderConfig({...providerConfig, sip_domain: e.target.value})}
                  className="input"
                  placeholder="sip.provedor.com"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Transporte</label>
                  <select
                    value={providerConfig.transport}
                    onChange={(e) => setProviderConfig({...providerConfig, transport: e.target.value})}
                    className="input"
                  >
                    <option value="UDP">UDP</option>
                    <option value="TCP">TCP</option>
                    <option value="TLS">TLS</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Caller ID</label>
                  <input
                    type="text"
                    value={providerConfig.caller_id}
                    onChange={(e) => setProviderConfig({...providerConfig, caller_id: e.target.value})}
                    className="input"
                    placeholder="+5531999999999"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Servidor STUN</label>
                <input
                  type="text"
                  value={providerConfig.stun_server}
                  onChange={(e) => setProviderConfig({...providerConfig, stun_server: e.target.value})}
                  className="input"
                />
              </div>

              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={providerConfig.recording_enabled}
                  onChange={(e) => setProviderConfig({...providerConfig, recording_enabled: e.target.checked})}
                  className="rounded border-gray-300"
                />
                <span className="text-sm text-gray-700">Ativar gravação de chamadas</span>
              </label>

              <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
                💡 A integração SIP requer um servidor Asterisk ou FreeSWITCH configurado. O softphone WebRTC se conecta diretamente ao servidor SIP.
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button onClick={() => setShowConfig(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={saveProviderConfig} className="btn-primary flex-1">
                Salvar configuração
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
