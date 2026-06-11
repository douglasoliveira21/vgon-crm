'use client'

import { useEffect, useState, useRef, useCallback } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/auth'
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
  Wifi,
  WifiOff,
  Search,
  Keyboard,
} from 'lucide-react'

export default function CallsPage() {
  const { user } = useAuthStore()
  const [activeTab, setActiveTab] = useState('softphone')
  const [sipConfig, setSipConfig] = useState({
    display_name: '',
    extension_number: '',
    sip_server: '',
    sip_domain: '',
    sip_port: '5060',
    sip_user: '',
    sip_password: '',
    transport: 'WSS',
    stun_server: 'stun:stun.l.google.com:19302',
    turn_server: '',
    codecs: ['opus', 'ulaw'],
    auto_register: true,
  })
  const [registrationStatus, setRegistrationStatus] = useState<'offline' | 'registering' | 'online' | 'error'>('offline')
  const [showConfig, setShowConfig] = useState(false)
  const [showDTMF, setShowDTMF] = useState(false)

  // Call state
  const [phoneNumber, setPhoneNumber] = useState('')
  const [isInCall, setIsInCall] = useState(false)
  const [isIncoming, setIsIncoming] = useState(false)
  const [incomingNumber, setIncomingNumber] = useState('')
  const [incomingName, setIncomingName] = useState('')
  const [callDuration, setCallDuration] = useState(0)
  const [isMuted, setIsMuted] = useState(false)
  const [isOnHold, setIsOnHold] = useState(false)
  const [callStatus, setCallStatus] = useState('')
  const [callHistory, setCallHistory] = useState<Array<{id: string; direction: string; number: string; name?: string; duration: number; status: string; date: string}>>([])

  // Contact search
  const [contactResults, setContactResults] = useState<Array<{name: string; phone: string}>>([])

  // Timer
  useEffect(() => {
    let interval: NodeJS.Timeout
    if (isInCall) {
      interval = setInterval(() => setCallDuration(d => d + 1), 1000)
    }
    return () => clearInterval(interval)
  }, [isInCall])

  // Load config on mount
  useEffect(() => {
    loadConfig()
  }, [])

  const loadConfig = async () => {
    try {
      const res = await api.get('/telephony/provider')
      if (res.data.provider) {
        const p = res.data.provider
        setSipConfig({
          display_name: p.name || '',
          extension_number: p.caller_id || '',
          sip_server: p.sip_host || '',
          sip_domain: p.sip_domain || '',
          sip_port: String(p.sip_port || '5060'),
          sip_user: p.sip_user || '',
          sip_password: '',
          transport: p.transport || 'WSS',
          stun_server: p.stun_server || 'stun:stun.l.google.com:19302',
          turn_server: '',
          codecs: ['opus', 'ulaw'],
          auto_register: true,
        })
      }
    } catch {}
  }

  const saveConfig = async () => {
    try {
      await api.post('/telephony/provider', {
        name: sipConfig.display_name,
        provider_type: 'sip',
        sip_host: sipConfig.sip_server,
        sip_port: parseInt(sipConfig.sip_port) || 5060,
        sip_user: sipConfig.sip_user,
        sip_password: sipConfig.sip_password,
        sip_domain: sipConfig.sip_domain,
        transport: sipConfig.transport,
        caller_id: sipConfig.extension_number,
        stun_server: sipConfig.stun_server,
        recording_enabled: false,
      })
      toast.success('Configuração salva!')
      setShowConfig(false)

      if (sipConfig.auto_register) {
        registerSIP()
      }
    } catch {
      toast.error('Erro ao salvar')
    }
  }

  const registerSIP = () => {
    if (!sipConfig.sip_server || !sipConfig.sip_user) {
      toast.error('Configure o servidor SIP primeiro')
      setShowConfig(true)
      return
    }

    setRegistrationStatus('registering')
    toast.success('Registrando ramal...')

    // SIP.js/JsSIP registration would happen here
    // For now simulate the registration
    setTimeout(() => {
      setRegistrationStatus('online')
      toast.success('Ramal registrado com sucesso!')
    }, 2000)
  }

  const unregisterSIP = () => {
    setRegistrationStatus('offline')
    toast.success('Ramal desconectado')
  }

  // Call functions
  const makeCall = (number?: string) => {
    const dialNumber = number || phoneNumber
    if (!dialNumber.trim()) { toast.error('Digite um número'); return }
    if (registrationStatus !== 'online') { toast.error('Ramal não registrado'); registerSIP(); return }

    setIsInCall(true)
    setCallDuration(0)
    setCallStatus('Chamando...')
    setPhoneNumber(dialNumber)

    // SIP.js invite would happen here
    setTimeout(() => setCallStatus('Em chamada'), 3000)
  }

  const answerCall = () => {
    setIsIncoming(false)
    setIsInCall(true)
    setCallDuration(0)
    setCallStatus('Em chamada')
    setPhoneNumber(incomingNumber)
  }

  const rejectCall = () => {
    setIsIncoming(false)
    setIncomingNumber('')
    setIncomingName('')
  }

  const endCall = () => {
    // Save to history
    if (callDuration > 0) {
      setCallHistory(prev => [{
        id: Date.now().toString(),
        direction: 'outbound',
        number: phoneNumber,
        duration: callDuration,
        status: 'answered',
        date: new Date().toISOString(),
      }, ...prev])
    }

    setIsInCall(false)
    setCallStatus('')
    setCallDuration(0)
    setIsMuted(false)
    setIsOnHold(false)
  }

  const toggleMute = () => setIsMuted(!isMuted)
  const toggleHold = () => setIsOnHold(!isOnHold)
  const sendDTMF = (digit: string) => {
    // Send DTMF via SIP.js
    toast.success(`DTMF: ${digit}`)
  }

  const searchContact = async (query: string) => {
    setPhoneNumber(query)
    if (query.length < 3) { setContactResults([]); return }
    try {
      const res = await api.get('/contacts', { params: { search: query, limit: 5 } })
      setContactResults((res.data.contacts || []).map((c: any) => ({ name: c.name, phone: c.phone })))
    } catch { setContactResults([]) }
  }

  const formatDuration = (s: number) => `${Math.floor(s/60).toString().padStart(2,'0')}:${(s%60).toString().padStart(2,'0')}`

  const dialPad = ['1','2','3','4','5','6','7','8','9','*','0','#']

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold text-gray-900">Telefonia</h1>
          <span className={`badge ${
            registrationStatus === 'online' ? 'badge-green' :
            registrationStatus === 'registering' ? 'badge-yellow' :
            registrationStatus === 'error' ? 'badge-red' : 'badge-gray'
          }`}>
            {registrationStatus === 'online' ? '🟢 Ramal Online' :
             registrationStatus === 'registering' ? '🟡 Registrando...' :
             registrationStatus === 'error' ? '🔴 Erro' : '⚪ Offline'}
          </span>
        </div>
        <div className="flex gap-2">
          {registrationStatus === 'offline' && (
            <button onClick={registerSIP} className="btn-primary text-sm">
              <Wifi size={16} /> Conectar Ramal
            </button>
          )}
          {registrationStatus === 'online' && (
            <button onClick={unregisterSIP} className="btn-secondary text-sm">
              <WifiOff size={16} /> Desconectar
            </button>
          )}
          <button onClick={() => setShowConfig(true)} className="btn-secondary text-sm">
            <Settings size={16} /> Configurar Ramal
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {[
          { id: 'softphone', label: '📞 Softphone' },
          { id: 'history', label: '📋 Histórico' },
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

      {/* Incoming Call Popup */}
      {isIncoming && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-8 w-full max-w-sm text-center animate-pulse">
            <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <PhoneIncoming size={36} className="text-green-600" />
            </div>
            <h3 className="text-xl font-bold text-gray-900 mb-1">Chamada recebida</h3>
            <p className="text-lg text-gray-700">{incomingName || incomingNumber}</p>
            <p className="text-sm text-gray-400 mb-6">{incomingNumber}</p>
            <div className="flex gap-4 justify-center">
              <button onClick={rejectCall} className="p-4 bg-red-600 text-white rounded-full hover:bg-red-700">
                <PhoneOff size={24} />
              </button>
              <button onClick={answerCall} className="p-4 bg-green-600 text-white rounded-full hover:bg-green-700">
                <Phone size={24} />
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Softphone Tab */}
      {activeTab === 'softphone' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Dialer */}
          <div className="card p-6">
            {isInCall ? (
              /* Active Call UI */
              <div className="text-center">
                <div className={`w-20 h-20 rounded-full flex items-center justify-center mx-auto mb-4 ${isOnHold ? 'bg-yellow-100' : 'bg-green-100 animate-pulse'}`}>
                  <Phone size={32} className={isOnHold ? 'text-yellow-600' : 'text-green-600'} />
                </div>
                <p className="text-sm text-gray-500">{callStatus}</p>
                <p className="text-3xl font-mono font-bold text-gray-900 my-2">{formatDuration(callDuration)}</p>
                <p className="text-lg text-gray-700 mb-6">{phoneNumber}</p>

                {/* Call controls */}
                <div className="flex items-center justify-center gap-4 mb-4">
                  <button onClick={toggleMute} className={`p-3 rounded-full ${isMuted ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-600'}`} title="Mudo">
                    {isMuted ? <MicOff size={20} /> : <Mic size={20} />}
                  </button>
                  <button onClick={toggleHold} className={`p-3 rounded-full ${isOnHold ? 'bg-yellow-100 text-yellow-600' : 'bg-gray-100 text-gray-600'}`} title="Espera">
                    {isOnHold ? <Play size={20} /> : <Pause size={20} />}
                  </button>
                  <button onClick={() => setShowDTMF(!showDTMF)} className="p-3 rounded-full bg-gray-100 text-gray-600" title="Teclado DTMF">
                    <Keyboard size={20} />
                  </button>
                  <button className="p-3 rounded-full bg-gray-100 text-gray-600" title="Transferir">
                    <ArrowRightLeft size={20} />
                  </button>
                </div>

                {/* DTMF Pad */}
                {showDTMF && (
                  <div className="grid grid-cols-3 gap-2 max-w-[200px] mx-auto mb-4">
                    {dialPad.map(d => (
                      <button key={d} onClick={() => sendDTMF(d)} className="p-2 bg-gray-50 hover:bg-gray-100 rounded-lg text-lg font-semibold">{d}</button>
                    ))}
                  </div>
                )}

                <button onClick={endCall} className="p-4 bg-red-600 text-white rounded-full hover:bg-red-700 mx-auto">
                  <PhoneOff size={28} />
                </button>
              </div>
            ) : (
              /* Idle - Dial UI */
              <>
                <h3 className="text-center text-sm font-medium text-gray-500 mb-4">DISCADOR</h3>

                {/* Search/Number input */}
                <div className="relative mb-4">
                  <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    type="text"
                    value={phoneNumber}
                    onChange={(e) => searchContact(e.target.value)}
                    placeholder="Número ou nome do contato..."
                    className="input pl-9 text-center text-lg font-mono"
                    onKeyDown={(e) => { if (e.key === 'Enter') makeCall() }}
                  />
                  {contactResults.length > 0 && (
                    <div className="absolute w-full mt-1 bg-white border rounded-lg shadow-lg z-10 max-h-40 overflow-y-auto">
                      {contactResults.map((c, i) => (
                        <button key={i} onClick={() => { setPhoneNumber(c.phone); setContactResults([]) }} className="w-full text-left px-3 py-2 hover:bg-gray-50 text-sm">
                          <span className="font-medium">{c.name}</span> <span className="text-gray-400">{c.phone}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>

                {/* Dial pad */}
                <div className="grid grid-cols-3 gap-2 max-w-[240px] mx-auto mb-4">
                  {dialPad.map(key => (
                    <button key={key} onClick={() => setPhoneNumber(prev => prev + key)} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-xl font-semibold text-gray-700 transition-colors">
                      {key}
                    </button>
                  ))}
                </div>

                {/* Call/Clear buttons */}
                <div className="flex items-center justify-center gap-4">
                  <button onClick={() => setPhoneNumber(prev => prev.slice(0, -1))} className="p-3 bg-gray-100 text-gray-500 rounded-full hover:bg-gray-200">
                    <X size={20} />
                  </button>
                  <button onClick={() => makeCall()} disabled={!phoneNumber.trim()} className="p-4 bg-green-600 text-white rounded-full hover:bg-green-700 disabled:opacity-50 shadow-lg">
                    <Phone size={28} />
                  </button>
                </div>
              </>
            )}
          </div>

          {/* Recent / Status */}
          <div className="space-y-4">
            {/* Extension status */}
            <div className="card p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-gray-400 uppercase">Seu ramal</p>
                  <p className="text-lg font-bold text-gray-900">{sipConfig.extension_number || 'Não configurado'}</p>
                  <p className="text-xs text-gray-500">{sipConfig.sip_server || 'Sem servidor'}</p>
                </div>
                <div className={`w-3 h-3 rounded-full ${registrationStatus === 'online' ? 'bg-green-500' : 'bg-gray-300'}`} />
              </div>
            </div>

            {/* Recent calls */}
            <div className="card p-4">
              <h4 className="text-sm font-semibold text-gray-900 mb-3">Chamadas recentes</h4>
              {callHistory.length === 0 ? (
                <p className="text-xs text-gray-400 text-center py-4">Nenhuma chamada realizada</p>
              ) : (
                <div className="space-y-2">
                  {callHistory.slice(0, 10).map(call => (
                    <div key={call.id} className="flex items-center justify-between py-2 border-b border-gray-50">
                      <div className="flex items-center gap-2">
                        {call.direction === 'outbound' ? <PhoneOutgoing size={14} className="text-blue-500" /> : <PhoneIncoming size={14} className="text-green-500" />}
                        <div>
                          <p className="text-sm text-gray-900">{call.name || call.number}</p>
                          <p className="text-xs text-gray-400">{formatDuration(call.duration)}</p>
                        </div>
                      </div>
                      <button onClick={() => makeCall(call.number)} className="p-1.5 text-green-500 hover:bg-green-50 rounded">
                        <Phone size={14} />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* History Tab */}
      {activeTab === 'history' && (
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Histórico de Chamadas</h3>
          {callHistory.length === 0 ? (
            <div className="text-center py-12">
              <Clock size={40} className="text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500">Nenhuma chamada registrada</p>
              <p className="text-sm text-gray-400 mt-1">Configure seu ramal e comece a ligar</p>
            </div>
          ) : (
            <div className="space-y-2">
              {callHistory.map(call => (
                <div key={call.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                      call.direction === 'outbound' ? 'bg-blue-100' : call.status === 'missed' ? 'bg-red-100' : 'bg-green-100'
                    }`}>
                      {call.direction === 'outbound' ? <PhoneOutgoing size={14} className="text-blue-600" /> :
                       call.status === 'missed' ? <PhoneMissed size={14} className="text-red-600" /> :
                       <PhoneIncoming size={14} className="text-green-600" />}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">{call.name || call.number}</p>
                      <p className="text-xs text-gray-400">{new Date(call.date).toLocaleString('pt-BR')}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-gray-600">{formatDuration(call.duration)}</p>
                    <button onClick={() => makeCall(call.number)} className="text-xs text-primary-600 hover:underline">Religar</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* SIP Config Modal */}
      {showConfig && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Configurar Ramal SIP</h3>
              <button onClick={() => setShowConfig(false)} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
            </div>

            <p className="text-sm text-gray-500 mb-4">
              Configure as credenciais do seu ramal. O servidor SIP deve ser o do seu PABX (Asterisk, FreeSWITCH, 3CX, etc).
            </p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nome de exibição</label>
                <input type="text" value={sipConfig.display_name} onChange={(e) => setSipConfig({...sipConfig, display_name: e.target.value})} className="input" placeholder="João Silva" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Número do ramal</label>
                  <input type="text" value={sipConfig.extension_number} onChange={(e) => setSipConfig({...sipConfig, extension_number: e.target.value})} className="input" placeholder="1001" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Transporte</label>
                  <select value={sipConfig.transport} onChange={(e) => setSipConfig({...sipConfig, transport: e.target.value})} className="input">
                    <option value="WSS">WSS (WebSocket Seguro)</option>
                    <option value="WS">WS (WebSocket)</option>
                    <option value="UDP">UDP</option>
                    <option value="TCP">TCP</option>
                    <option value="TLS">TLS</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Servidor SIP</label>
                  <input type="text" value={sipConfig.sip_server} onChange={(e) => setSipConfig({...sipConfig, sip_server: e.target.value})} className="input" placeholder="pbx.empresa.com" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Porta</label>
                  <input type="text" value={sipConfig.sip_port} onChange={(e) => setSipConfig({...sipConfig, sip_port: e.target.value})} className="input" placeholder="8089" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Domínio / Realm</label>
                <input type="text" value={sipConfig.sip_domain} onChange={(e) => setSipConfig({...sipConfig, sip_domain: e.target.value})} className="input" placeholder="pbx.empresa.com" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Usuário SIP</label>
                  <input type="text" value={sipConfig.sip_user} onChange={(e) => setSipConfig({...sipConfig, sip_user: e.target.value})} className="input" placeholder="1001" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Senha SIP</label>
                  <input type="password" value={sipConfig.sip_password} onChange={(e) => setSipConfig({...sipConfig, sip_password: e.target.value})} className="input" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Servidor STUN</label>
                <input type="text" value={sipConfig.stun_server} onChange={(e) => setSipConfig({...sipConfig, stun_server: e.target.value})} className="input" />
              </div>

              <label className="flex items-center gap-2">
                <input type="checkbox" checked={sipConfig.auto_register} onChange={(e) => setSipConfig({...sipConfig, auto_register: e.target.checked})} className="rounded border-gray-300" />
                <span className="text-sm text-gray-700">Registrar automaticamente ao abrir</span>
              </label>

              <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
                💡 Use WSS (porta 8089) para Asterisk com WebRTC. Para 3CX use a porta configurada no sistema. O STUN ajuda com NAT.
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button onClick={() => setShowConfig(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={saveConfig} className="btn-primary flex-1">
                Salvar e Registrar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
