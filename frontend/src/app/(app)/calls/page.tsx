'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { useSIP } from '@/hooks/useSIP'
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
  X,
  Clock,
  Search,
  Keyboard,
  Wifi,
  WifiOff,
  Users,
  Activity,
  BarChart3,
  Server,
  CheckCircle,
  XCircle,
  PhoneCall,
  ListMusic,
  GitBranch,
} from 'lucide-react'

export default function CallsPage() {
  const sip = useSIP()
  const router = useRouter()
  const [activeTab, setActiveTab] = useState('dashboard')
  const [phoneNumber, setPhoneNumber] = useState('')
  const [showConfig, setShowConfig] = useState(false)
  const [showDTMF, setShowDTMF] = useState(false)
  const [contactResults, setContactResults] = useState<Array<{name: string; phone: string}>>([])
  const [callHistory, setCallHistory] = useState<Array<{id: string; direction: string; number: string; name?: string; duration: number; status: string; date: string}>>([])
  const [extensions, setExtensions] = useState<Array<{id: string; extension_number: string; display_name: string; status: string; can_call_external: boolean; outbound_trunk_id?: string; outbound_trunk_name?: string}>>([])
  const [trunks, setTrunks] = useState<Array<{id: string; name: string; sip_server: string; sip_port: number; transport: string; caller_id?: string; is_active: boolean}>>([])
  const [queues, setQueues] = useState<Array<{id: string; name: string; strategy: string}>>([])
  const [serverStatus, setServerStatus] = useState<{ari: boolean; ami: boolean; webrtc: boolean} | null>(null)
  const [dashboardStats, setDashboardStats] = useState<any>(null)

  const [sipConfig, setSipConfig] = useState({
    display_name: '',
    extension_number: '',
    webrtc_domain: 'voip.vgon.com.br',
    webrtc_ws_url: 'wss://voip.vgon.com.br:8089/ws',
    sip_user: '',
    sip_password: '',
    outbound_trunk_id: '',
    transport: 'WSS',
    stun_server: 'stun:stun.l.google.com:19302',
    auto_register: true,
  })
  const [showTrunkConfig, setShowTrunkConfig] = useState(false)
  const [trunkConfig, setTrunkConfig] = useState({
    name: '',
    description: '',
    sip_server: '',
    sip_port: '5060',
    transport: 'UDP',
    sip_domain: '',
    username: '',
    password: '',
    caller_id: '',
    realm: '',
    outbound_proxy: '',
    codecs: ['ulaw', 'alaw'],
    nat: true,
    keep_alive: '60',
    dtmf: 'rfc4733',
    register_expires: '300',
    is_active: true,
  })

  // Incoming call CRM data
  const [incomingContact, setIncomingContact] = useState<{name: string; company?: string; phone: string; id?: string} | null>(null)
  const [loadingContact, setLoadingContact] = useState(false)

  // Fetch contact when incoming call starts
  useEffect(() => {
    if (sip.isIncoming && sip.incomingNumber) {
      setLoadingContact(true)
      api.get('/contacts', { params: { search: sip.incomingNumber, limit: 1 } })
        .then(res => {
          const contacts = res.data.contacts || []
          if (contacts.length > 0) {
            setIncomingContact({
              name: contacts[0].name,
              company: contacts[0].company || contacts[0].tags?.[0] || '',
              phone: contacts[0].phone,
              id: contacts[0].id,
            })
          } else {
            setIncomingContact(null)
          }
        })
        .catch(() => setIncomingContact(null))
        .finally(() => setLoadingContact(false))
    } else {
      setIncomingContact(null)
    }
  }, [sip.isIncoming, sip.incomingNumber])

  useEffect(() => {
    loadConfig()
    fetchTrunks()
    fetchExtensions()
    fetchQueues()
    fetchCallHistory()
    fetchDashboardStats()
    testServerConnection()
  }, [])

  const loadConfig = async () => {
    try {
      const webrtcRes = await api.get('/telephony/webrtc/config').catch(() => null)
      if (webrtcRes?.data) {
        const cfg = webrtcRes.data
        setSipConfig(prev => ({
          ...prev,
          display_name: cfg.display_name || prev.display_name,
          extension_number: cfg.extension_number || prev.extension_number,
          sip_user: cfg.username || prev.sip_user,
          sip_password: cfg.password || prev.sip_password,
          webrtc_domain: cfg.webrtc_domain || prev.webrtc_domain,
          webrtc_ws_url: cfg.webrtc_ws_url || prev.webrtc_ws_url,
          transport: cfg.transport || prev.transport,
          stun_server: cfg.stun_server || prev.stun_server,
        }))
      }
    } catch {}
  }

  const fetchTrunks = async () => {
    try {
      const res = await api.get('/telephony/trunks')
      setTrunks(res.data.trunks || [])
    } catch {}
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

  const fetchCallHistory = async () => {
    try {
      const res = await api.get('/calls/history', { params: { limit: 50 } })
      setCallHistory((res.data.calls || []).map((c: any) => ({
        id: c.id, direction: c.direction, number: c.direction === 'inbound' ? c.from_number : c.to_number,
        name: c.contact_name, duration: c.duration || 0, status: c.status, date: c.created_at,
      })))
    } catch {}
  }

  const fetchDashboardStats = async () => {
    try {
      const res = await api.get('/calls/history', { params: { limit: 500 } })
      const calls = res.data.calls || []
      const today = new Date().toISOString().split('T')[0]
      const todayCalls = calls.filter((c: any) => c.created_at?.startsWith(today))
      setDashboardStats({
        total: calls.length,
        today_inbound: todayCalls.filter((c: any) => c.direction === 'inbound').length,
        today_outbound: todayCalls.filter((c: any) => c.direction === 'outbound').length,
        today_missed: todayCalls.filter((c: any) => c.status === 'missed' || c.status === 'no_answer').length,
        avg_duration: calls.length > 0 ? Math.round(calls.reduce((s: number, c: any) => s + (c.duration || 0), 0) / calls.length) : 0,
      })
    } catch {}
  }

  const testServerConnection = async () => {
    try {
      const res = await api.get('/telephony/status')
      setServerStatus({ ari: !res.data.error, ami: true, webrtc: sip.status === 'online' })
    } catch {
      setServerStatus({ ari: false, ami: false, webrtc: false })
    }
  }

  const saveConfig = async () => {
    try {
      if (sipConfig.extension_number && sipConfig.sip_password) {
        await api.post('/telephony/extensions', {
          display_name: sipConfig.display_name || sipConfig.extension_number,
          extension_number: sipConfig.extension_number,
          extension_password: sipConfig.sip_password,
          sip_username: sipConfig.sip_user || sipConfig.extension_number,
          webrtc_domain: sipConfig.webrtc_domain,
          webrtc_ws_url: sipConfig.webrtc_ws_url,
          stun_server: sipConfig.stun_server,
          outbound_trunk_id: sipConfig.outbound_trunk_id,
          can_call_external: true,
          can_receive_calls: true,
          can_transfer: true,
          can_access_recordings: true,
        }).catch(() => {})
      }
      toast.success('ConfiguraÃ§Ã£o salva!')
      setShowConfig(false)
      if (sipConfig.auto_register) doRegister()
    } catch { toast.error('Erro ao salvar') }
  }

  const saveTrunk = async () => {
    try {
      await api.post('/telephony/trunks', {
        ...trunkConfig,
        sip_port: parseInt(trunkConfig.sip_port) || 5060,
        keep_alive: parseInt(trunkConfig.keep_alive) || 60,
        register_expires: parseInt(trunkConfig.register_expires) || 300,
      })
      toast.success('Operadora SIP salva!')
      setShowTrunkConfig(false)
      setTrunkConfig({
        name: '',
        description: '',
        sip_server: '',
        sip_port: '5060',
        transport: 'UDP',
        sip_domain: '',
        username: '',
        password: '',
        caller_id: '',
        realm: '',
        outbound_proxy: '',
        codecs: ['ulaw', 'alaw'],
        nat: true,
        keep_alive: '60',
        dtmf: 'rfc4733',
        register_expires: '300',
        is_active: true,
      })
      fetchTrunks()
    } catch { toast.error('Erro ao salvar operadora') }
  }

  const doRegister = () => {
    const sipUser = sipConfig.sip_user || sipConfig.extension_number
    if (!sipConfig.webrtc_ws_url || !sipUser) {
      toast.error('Configure o ramal primeiro')
      setShowConfig(true)
      return
    }
    sip.register({
      server: sipConfig.webrtc_domain,
      port: '5060',
      domain: sipConfig.webrtc_domain,
      websocketUrl: sipConfig.webrtc_ws_url,
      user: sipUser,
      password: sipConfig.sip_password,
      displayName: sipConfig.display_name || sipUser,
      transport: sipConfig.transport,
      stunServer: sipConfig.stun_server,
    })
  }

  const handleMakeCall = (number?: string) => {
    const n = number || phoneNumber
    if (!n.trim()) { toast.error('Digite um nÃºmero'); return }
    if (sip.status !== 'online') { doRegister(); return }
    sip.makeCall(n)
    setPhoneNumber(n)
  }

  const handleEndCall = () => {
    if (sip.callDuration > 0) {
      setCallHistory(prev => [{ id: Date.now().toString(), direction: 'outbound', number: phoneNumber || sip.remoteNumber, duration: sip.callDuration, status: 'answered', date: new Date().toISOString() }, ...prev])
    }
    sip.endCall()
  }

  const searchContact = async (q: string) => {
    setPhoneNumber(q)
    if (q.length < 3) { setContactResults([]); return }
    try {
      const res = await api.get('/contacts', { params: { search: q, limit: 5 } })
      setContactResults((res.data.contacts || []).map((c: any) => ({ name: c.name, phone: c.phone })))
    } catch { setContactResults([]) }
  }

  const fmt = (s: number) => `${Math.floor(s/60).toString().padStart(2,'0')}:${(s%60).toString().padStart(2,'0')}`
  const dialPad = ['1','2','3','4','5','6','7','8','9','*','0','#']
  const isInCall = sip.callStatus !== 'idle'

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold text-gray-900">Telefonia</h1>
          <span className={`badge ${sip.status === 'online' ? 'badge-green' : sip.status === 'registering' ? 'badge-yellow' : sip.status === 'error' ? 'badge-red' : 'badge-gray'}`}>
            {sip.status === 'online' ? 'ðŸŸ¢ Online' : sip.status === 'registering' ? 'ðŸŸ¡ Registrando' : sip.status === 'error' ? 'ðŸ”´ Erro' : 'âšª Offline'}
          </span>
        </div>
        <div className="flex gap-2">
          {sip.status === 'offline' && <button onClick={doRegister} className="btn-primary text-sm"><Wifi size={16} /> Conectar</button>}
          {sip.status === 'online' && <button onClick={() => sip.unregister()} className="btn-secondary text-sm"><WifiOff size={16} /> Desconectar</button>}
          <button onClick={() => setShowConfig(true)} className="btn-secondary text-sm"><Settings size={16} /> Ramal</button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6 overflow-x-auto">
        <button onClick={() => setActiveTab('dashboard')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap ${activeTab === 'dashboard' ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'}`}>ðŸ“Š Dashboard</button>
        <button onClick={() => setActiveTab('softphone')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap ${activeTab === 'softphone' ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'}`}>Softphone</button>
        <button onClick={() => setActiveTab('trunks')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap ${activeTab === 'trunks' ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'}`}>Operadoras SIP</button>
        <button onClick={() => setActiveTab('extensions')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap ${activeTab === 'extensions' ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'}`}>Ramais</button>
        <button onClick={() => setActiveTab('history')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap ${activeTab === 'history' ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'}`}>ðŸ“‹ HistÃ³rico</button>
        <button onClick={() => setActiveTab('queues')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap ${activeTab === 'queues' ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'}`}>ðŸ‘¥ Filas</button>
        <button onClick={() => router.push('/calls/recordings')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap text-gray-500 hover:bg-gray-100`}>ðŸŽ™ï¸ GravaÃ§Ãµes</button>
        <button onClick={() => router.push('/calls/ivr')} className={`px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap text-gray-500 hover:bg-gray-100`}>ðŸ“ž URA</button>
      </div>

      {/* Dashboard */}
      {activeTab === 'dashboard' && (
        <div className="space-y-6">
          {/* Server Status */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="card p-4 flex items-center gap-3">
              <Server size={20} className={serverStatus?.ari ? 'text-green-500' : 'text-red-500'} />
              <div>
                <p className="text-xs text-gray-500">Servidor ARI</p>
                <p className="text-sm font-medium">{serverStatus?.ari ? 'ðŸŸ¢ Conectado' : 'ðŸ”´ Desconectado'}</p>
              </div>
            </div>
            <div className="card p-4 flex items-center gap-3">
              <Activity size={20} className={serverStatus?.ami ? 'text-green-500' : 'text-red-500'} />
              <div>
                <p className="text-xs text-gray-500">AMI</p>
                <p className="text-sm font-medium">{serverStatus?.ami ? 'ðŸŸ¢ Conectado' : 'ðŸ”´ Desconectado'}</p>
              </div>
            </div>
            <div className="card p-4 flex items-center gap-3">
              <Wifi size={20} className={sip.status === 'online' ? 'text-green-500' : 'text-red-500'} />
              <div>
                <p className="text-xs text-gray-500">WebRTC</p>
                <p className="text-sm font-medium">{sip.status === 'online' ? 'ðŸŸ¢ Conectado' : sip.status === 'registering' ? 'ðŸŸ¡ Registrando' : 'ðŸ”´ Desconectado'}</p>
              </div>
            </div>
            <div className="card p-4 flex items-center gap-3">
              <Users size={20} className="text-blue-500" />
              <div>
                <p className="text-xs text-gray-500">Ramais Registrados</p>
                <p className="text-sm font-medium">{extensions.length}</p>
              </div>
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <div className="card p-4 text-center">
              <PhoneCall size={24} className="text-blue-500 mx-auto mb-2" />
              <p className="text-2xl font-bold">{dashboardStats?.today_inbound || 0}</p>
              <p className="text-xs text-gray-500">Recebidas hoje</p>
            </div>
            <div className="card p-4 text-center">
              <PhoneOutgoing size={24} className="text-green-500 mx-auto mb-2" />
              <p className="text-2xl font-bold">{dashboardStats?.today_outbound || 0}</p>
              <p className="text-xs text-gray-500">Realizadas hoje</p>
            </div>
            <div className="card p-4 text-center">
              <PhoneMissed size={24} className="text-red-500 mx-auto mb-2" />
              <p className="text-2xl font-bold">{dashboardStats?.today_missed || 0}</p>
              <p className="text-xs text-gray-500">Perdidas hoje</p>
            </div>
            <div className="card p-4 text-center">
              <Clock size={24} className="text-purple-500 mx-auto mb-2" />
              <p className="text-2xl font-bold">{dashboardStats?.avg_duration ? fmt(dashboardStats.avg_duration) : '0:00'}</p>
              <p className="text-xs text-gray-500">Tempo mÃ©dio</p>
            </div>
            <div className="card p-4 text-center">
              <BarChart3 size={24} className="text-orange-500 mx-auto mb-2" />
              <p className="text-2xl font-bold">{dashboardStats?.total || 0}</p>
              <p className="text-xs text-gray-500">Total chamadas</p>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="card p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Atalhos RÃ¡pidos</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <button onClick={() => setShowConfig(true)} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <Settings size={20} className="text-gray-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">Configurar Servidor</p>
              </button>
              <button onClick={() => setActiveTab('softphone')} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <Phone size={20} className="text-green-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">Abrir Discador</p>
              </button>
              <button onClick={() => setActiveTab('extensions')} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <Users size={20} className="text-blue-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">Ramais</p>
              </button>
              <button onClick={() => setActiveTab('history')} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <Clock size={20} className="text-purple-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">HistÃ³rico</p>
              </button>
              <button onClick={() => setActiveTab('queues')} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <ListMusic size={20} className="text-orange-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">Filas</p>
              </button>
              <button onClick={() => testServerConnection()} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <Activity size={20} className="text-cyan-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">Testar ConexÃ£o</p>
              </button>
              <button onClick={doRegister} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <Wifi size={20} className="text-indigo-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">Registrar Ramal</p>
              </button>
              <button onClick={() => setActiveTab('queues')} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-center transition-colors">
                <GitBranch size={20} className="text-pink-600 mx-auto mb-1" />
                <p className="text-xs text-gray-700 font-medium">URA</p>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Incoming Call Popup */}
      {sip.isIncoming && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-8 w-full max-w-sm text-center">
            <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4 animate-pulse">
              <PhoneIncoming size={36} className="text-green-600" />
            </div>
            <h3 className="text-xl font-bold text-gray-900">Chamada recebida</h3>
            <p className="text-lg text-gray-700 mt-2">{sip.incomingNumber}</p>

            {/* CRM Contact Info */}
            {loadingContact && (
              <p className="text-xs text-gray-400 mt-2">Buscando contato...</p>
            )}
            {incomingContact && (
              <div className="mt-3 p-3 bg-blue-50 rounded-lg text-left">
                <p className="text-sm font-semibold text-blue-900">{incomingContact.name}</p>
                {incomingContact.company && (
                  <p className="text-xs text-blue-700">{incomingContact.company}</p>
                )}
              </div>
            )}
            {!loadingContact && !incomingContact && (
              <p className="text-xs text-gray-400 mt-2">Contato nÃ£o encontrado no CRM</p>
            )}

            <div className="flex gap-3 justify-center mt-6">
              <button onClick={() => sip.endCall()} className="p-4 bg-red-600 text-white rounded-full hover:bg-red-700" title="Rejeitar">
                <PhoneOff size={24} />
              </button>
              <button onClick={() => { sip.answerCall(); setPhoneNumber(sip.incomingNumber) }} className="p-4 bg-green-600 text-white rounded-full hover:bg-green-700" title="Atender">
                <Phone size={24} />
              </button>
            </div>
            <div className="flex gap-2 justify-center mt-3">
              {!incomingContact && (
                <button
                  onClick={() => { window.open(`/contacts?new=true&phone=${sip.incomingNumber}`, '_blank') }}
                  className="text-xs text-primary-600 hover:underline"
                >
                  + Criar Contato
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Softphone */}
      {activeTab === 'softphone' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-6">
            {isInCall ? (
              <div className="text-center">
                <div className={`w-20 h-20 rounded-full flex items-center justify-center mx-auto mb-4 ${sip.isOnHold ? 'bg-yellow-100' : 'bg-green-100 animate-pulse'}`}>
                  <Phone size={32} className={sip.isOnHold ? 'text-yellow-600' : 'text-green-600'} />
                </div>
                <p className="text-sm text-gray-500">
                  {sip.callStatus === 'calling' ? 'Chamando...' : sip.callStatus === 'ringing' ? 'Tocando...' : sip.isOnHold ? 'Em espera' : 'Em chamada'}
                </p>
                <p className="text-3xl font-mono font-bold text-gray-900 my-2">{fmt(sip.callDuration)}</p>
                <p className="text-lg text-gray-700 mb-6">{phoneNumber || sip.remoteNumber}</p>

                <div className="flex items-center justify-center gap-4 mb-4">
                  <button onClick={() => sip.toggleMute()} className={`p-3 rounded-full ${sip.isMuted ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-600'}`}>{sip.isMuted ? <MicOff size={20} /> : <Mic size={20} />}</button>
                  <button onClick={() => sip.toggleHold()} className={`p-3 rounded-full ${sip.isOnHold ? 'bg-yellow-100 text-yellow-600' : 'bg-gray-100 text-gray-600'}`}>{sip.isOnHold ? <Play size={20} /> : <Pause size={20} />}</button>
                  <button onClick={() => setShowDTMF(!showDTMF)} className="p-3 rounded-full bg-gray-100 text-gray-600"><Keyboard size={20} /></button>
                  <button className="p-3 rounded-full bg-gray-100 text-gray-600"><ArrowRightLeft size={20} /></button>
                </div>

                {showDTMF && (
                  <div className="grid grid-cols-3 gap-2 max-w-[200px] mx-auto mb-4">
                    {dialPad.map(d => <button key={d} onClick={() => sip.sendDTMF(d)} className="p-2 bg-gray-50 hover:bg-gray-100 rounded-lg text-lg font-semibold">{d}</button>)}
                  </div>
                )}

                <button onClick={handleEndCall} className="p-4 bg-red-600 text-white rounded-full hover:bg-red-700"><PhoneOff size={28} /></button>
              </div>
            ) : (
              <>
                <h3 className="text-center text-sm font-medium text-gray-500 mb-4">DISCADOR</h3>
                <div className="relative mb-4">
                  <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input type="text" value={phoneNumber} onChange={(e) => searchContact(e.target.value)} placeholder="NÃºmero ou contato..." className="input pl-9 text-center text-lg font-mono" onKeyDown={(e) => { if (e.key === 'Enter') handleMakeCall() }} />
                  {contactResults.length > 0 && (
                    <div className="absolute w-full mt-1 bg-white border rounded-lg shadow-lg z-10">
                      {contactResults.map((c, i) => (
                        <button key={i} onClick={() => { setPhoneNumber(c.phone); setContactResults([]) }} className="w-full text-left px-3 py-2 hover:bg-gray-50 text-sm">
                          <span className="font-medium">{c.name}</span> <span className="text-gray-400">{c.phone}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
                <div className="grid grid-cols-3 gap-2 max-w-[240px] mx-auto mb-4">
                  {dialPad.map(key => <button key={key} onClick={() => setPhoneNumber(p => p + key)} className="p-3 bg-gray-50 hover:bg-gray-100 rounded-xl text-xl font-semibold text-gray-700">{key}</button>)}
                </div>
                <div className="flex items-center justify-center gap-4">
                  <button onClick={() => setPhoneNumber(p => p.slice(0, -1))} className="p-3 bg-gray-100 text-gray-500 rounded-full"><X size={20} /></button>
                  <button onClick={() => handleMakeCall()} disabled={!phoneNumber.trim()} className="p-4 bg-green-600 text-white rounded-full hover:bg-green-700 disabled:opacity-50 shadow-lg"><Phone size={28} /></button>
                </div>
              </>
            )}
          </div>

          <div className="space-y-4">
            <div className="card p-4">
              <p className="text-xs text-gray-400">Seu ramal</p>
              <p className="text-lg font-bold text-gray-900">{sipConfig.extension_number || 'NÃ£o configurado'}</p>
              <p className="text-xs text-gray-500">{sipConfig.webrtc_domain || 'Sem dominio WebRTC'}</p>
            </div>
            <div className="card p-4">
              <h4 className="text-sm font-semibold text-gray-900 mb-3">Recentes</h4>
              {callHistory.length === 0 ? <p className="text-xs text-gray-400 text-center py-4">Nenhuma chamada</p> : (
                <div className="space-y-2">
                  {callHistory.slice(0, 8).map(call => (
                    <div key={call.id} className="flex items-center justify-between py-2 border-b border-gray-50">
                      <div className="flex items-center gap-2">
                        <PhoneOutgoing size={14} className="text-blue-500" />
                        <div><p className="text-sm">{call.number}</p><p className="text-xs text-gray-400">{fmt(call.duration)}</p></div>
                      </div>
                      <button onClick={() => handleMakeCall(call.number)} className="p-1 text-green-500 hover:bg-green-50 rounded"><Phone size={14} /></button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* SIP Trunks */}
      {activeTab === 'trunks' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">Operadoras SIP</h3>
              <p className="text-sm text-gray-500">Cadastre trunks PJSIP. Ramais apenas escolhem qual operadora usar para saída.</p>
            </div>
            <button onClick={() => setShowTrunkConfig(true)} className="btn-primary text-sm">+ Nova Operadora</button>
          </div>
          {trunks.length === 0 ? (
            <div className="text-center py-10 text-gray-400">
              <Server size={36} className="mx-auto mb-3 opacity-50" />
              <p>Nenhuma operadora SIP cadastrada</p>
            </div>
          ) : (
            <div className="space-y-2">
              {trunks.map((trunk) => (
                <div key={trunk.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{trunk.name}</p>
                    <p className="text-xs text-gray-500">{trunk.transport} {trunk.sip_server}:{trunk.sip_port}</p>
                  </div>
                  <span className={`badge text-xs ${trunk.is_active ? 'badge-green' : 'badge-gray'}`}>
                    {trunk.is_active ? 'Ativa' : 'Inativa'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* History */}
      {activeTab === 'history' && (
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">HistÃ³rico</h3>
          {callHistory.length === 0 ? (
            <div className="text-center py-12"><Clock size={40} className="text-gray-300 mx-auto mb-4" /><p className="text-gray-500">Nenhuma chamada</p></div>
          ) : (
            <div className="space-y-2">
              {callHistory.map(call => (
                <div key={call.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <PhoneOutgoing size={14} className="text-blue-600" />
                    <div><p className="text-sm font-medium">{call.number}</p><p className="text-xs text-gray-400">{new Date(call.date).toLocaleString('pt-BR')}</p></div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm">{fmt(call.duration)}</p>
                    <button onClick={() => handleMakeCall(call.number)} className="text-xs text-primary-600">Religar</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Extensions Tab */}
      {activeTab === 'extensions' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Ramais</h3>
            <button onClick={() => setShowConfig(true)} className="btn-primary text-sm">+ Novo Ramal</button>
          </div>
          {extensions.length === 0 ? (
            <div className="text-center py-8 text-gray-400">
              <Phone size={32} className="mx-auto mb-3 opacity-50" />
              <p>Nenhum ramal cadastrado</p>
              <p className="text-xs mt-1">Configure os ramais na aba de configuraÃ§Ãµes do servidor</p>
            </div>
          ) : (
            <div className="space-y-2">
              {extensions.map((ext) => (
                <div key={ext.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className={`w-3 h-3 rounded-full ${ext.status === 'online' ? 'bg-green-500' : 'bg-gray-300'}`} />
                    <div>
                      <p className="text-sm font-medium text-gray-900">{ext.display_name}</p>
                      <p className="text-xs text-gray-500">Ramal: {ext.extension_number}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`badge text-xs ${ext.status === 'online' ? 'badge-green' : 'badge-gray'}`}>
                      {ext.status === 'online' ? 'Online' : 'Offline'}
                    </span>
                    {ext.can_call_external && <span className="text-xs text-gray-400">Ext. âœ“</span>}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Queues Tab */}
      {activeTab === 'queues' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Filas de Atendimento</h3>
          </div>
          {queues.length === 0 ? (
            <div className="text-center py-8 text-gray-400">
              <Users size={32} className="mx-auto mb-3 opacity-50" />
              <p>Nenhuma fila criada</p>
              <p className="text-xs mt-1">Filas distribuem chamadas entre os atendentes</p>
            </div>
          ) : (
            <div className="space-y-2">
              {queues.map((q) => (
                <div key={q.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{q.name}</p>
                    <p className="text-xs text-gray-500">EstratÃ©gia: {q.strategy}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* SIP Trunk Modal */}
      {showTrunkConfig && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-3xl max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Cadastrar Operadora SIP</h3>
              <button onClick={() => setShowTrunkConfig(false)} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
            </div>
            <p className="text-sm text-gray-500 mb-4">
              Estes dados sao da operadora/trunk. O ramal apenas escolhe uma operadora de saida.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Nome</label><input type="text" value={trunkConfig.name} onChange={e => setTrunkConfig({...trunkConfig, name: e.target.value})} className="input" placeholder="Vono, Vivo, Algar..." /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Servidor SIP</label><input type="text" value={trunkConfig.sip_server} onChange={e => setTrunkConfig({...trunkConfig, sip_server: e.target.value})} className="input" placeholder="sip.operadora.com.br" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Porta</label><input type="number" value={trunkConfig.sip_port} onChange={e => setTrunkConfig({...trunkConfig, sip_port: e.target.value})} className="input" placeholder="5060" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Transporte</label><select value={trunkConfig.transport} onChange={e => setTrunkConfig({...trunkConfig, transport: e.target.value})} className="input"><option value="UDP">UDP</option><option value="TCP">TCP</option><option value="TLS">TLS</option></select></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Dominio SIP</label><input type="text" value={trunkConfig.sip_domain} onChange={e => setTrunkConfig({...trunkConfig, sip_domain: e.target.value})} className="input" placeholder="operadora.com.br" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Usuario</label><input type="text" value={trunkConfig.username} onChange={e => setTrunkConfig({...trunkConfig, username: e.target.value})} className="input" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Senha</label><input type="password" value={trunkConfig.password} onChange={e => setTrunkConfig({...trunkConfig, password: e.target.value})} className="input" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">CallerID</label><input type="text" value={trunkConfig.caller_id} onChange={e => setTrunkConfig({...trunkConfig, caller_id: e.target.value})} className="input" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Realm opcional</label><input type="text" value={trunkConfig.realm} onChange={e => setTrunkConfig({...trunkConfig, realm: e.target.value})} className="input" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Proxy Outbound opcional</label><input type="text" value={trunkConfig.outbound_proxy} onChange={e => setTrunkConfig({...trunkConfig, outbound_proxy: e.target.value})} className="input" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">DTMF</label><select value={trunkConfig.dtmf} onChange={e => setTrunkConfig({...trunkConfig, dtmf: e.target.value})} className="input"><option value="rfc4733">RFC4733</option><option value="inband">Inband</option><option value="info">SIP INFO</option></select></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Tempo de Registro</label><input type="number" value={trunkConfig.register_expires} onChange={e => setTrunkConfig({...trunkConfig, register_expires: e.target.value})} className="input" placeholder="300" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Keep Alive</label><input type="number" value={trunkConfig.keep_alive} onChange={e => setTrunkConfig({...trunkConfig, keep_alive: e.target.value})} className="input" placeholder="60" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Descricao</label><input type="text" value={trunkConfig.description} onChange={e => setTrunkConfig({...trunkConfig, description: e.target.value})} className="input" /></div>
            </div>
            <div className="mt-4">
              <label className="block text-xs font-medium text-gray-700 mb-2">Codecs</label>
              <div className="flex flex-wrap gap-3">
                {['ulaw', 'alaw', 'opus', 'g722', 'g729'].map(codec => (
                  <label key={codec} className="flex items-center gap-2 text-sm text-gray-700">
                    <input
                      type="checkbox"
                      checked={trunkConfig.codecs.includes(codec)}
                      onChange={e => setTrunkConfig({
                        ...trunkConfig,
                        codecs: e.target.checked ? [...trunkConfig.codecs, codec] : trunkConfig.codecs.filter(c => c !== codec),
                      })}
                    />
                    {codec.toUpperCase()}
                  </label>
                ))}
              </div>
            </div>
            <div className="flex flex-wrap gap-4 mt-4">
              <label className="flex items-center gap-2"><input type="checkbox" checked={trunkConfig.nat} onChange={e => setTrunkConfig({...trunkConfig, nat: e.target.checked})} /><span className="text-sm text-gray-700">NAT</span></label>
              <label className="flex items-center gap-2"><input type="checkbox" checked={trunkConfig.is_active} onChange={e => setTrunkConfig({...trunkConfig, is_active: e.target.checked})} /><span className="text-sm text-gray-700">Ativa</span></label>
            </div>
            <div className="flex gap-3 mt-6">
              <button onClick={() => setShowTrunkConfig(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={saveTrunk} className="btn-primary flex-1">Salvar Operadora</button>
            </div>
          </div>
        </div>
      )}

      {/* Config Modal */}
      {showConfig && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Configurar Discador WebRTC</h3>
              <button onClick={() => setShowConfig(false)} className="text-gray-400 hover:text-gray-600"><X size={20} /></button>
            </div>
            <p className="text-sm text-gray-500 mb-4">
              Informe somente o ramal que vai registrar no Asterisk. A operadora/tronco VoIP fica na aba Operadoras SIP.
            </p>
            <div className="space-y-3">
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Nome</label><input type="text" value={sipConfig.display_name} onChange={e => setSipConfig({...sipConfig, display_name: e.target.value})} className="input" placeholder="JoÃ£o Silva" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Ramal</label><input type="text" value={sipConfig.extension_number} onChange={e => setSipConfig({...sipConfig, extension_number: e.target.value})} className="input" placeholder="1001" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="block text-xs font-medium text-gray-700 mb-1">UsuÃ¡rio SIP</label><input type="text" value={sipConfig.sip_user} onChange={e => setSipConfig({...sipConfig, sip_user: e.target.value})} className="input" placeholder="1001" /></div>
                <div><label className="block text-xs font-medium text-gray-700 mb-1">Senha</label><input type="password" value={sipConfig.sip_password} onChange={e => setSipConfig({...sipConfig, sip_password: e.target.value})} className="input" /></div>
              </div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">DomÃ­nio WebRTC do Asterisk</label><input type="text" value={sipConfig.webrtc_domain} onChange={e => setSipConfig({...sipConfig, webrtc_domain: e.target.value})} className="input" placeholder="voip.vgon.com.br" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">Operadora de saída</label><select value={sipConfig.outbound_trunk_id} onChange={e => setSipConfig({...sipConfig, outbound_trunk_id: e.target.value})} className="input"><option value="">Nenhuma operadora</option>{trunks.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}</select></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">WebSocket WSS</label><input type="text" value={sipConfig.webrtc_ws_url} onChange={e => setSipConfig({...sipConfig, webrtc_ws_url: e.target.value})} className="input" placeholder="wss://voip.vgon.com.br:8089/ws" /></div>
              <div><label className="block text-xs font-medium text-gray-700 mb-1">STUN</label><input type="text" value={sipConfig.stun_server} onChange={e => setSipConfig({...sipConfig, stun_server: e.target.value})} className="input" /></div>
              <label className="flex items-center gap-2"><input type="checkbox" checked={sipConfig.auto_register} onChange={e => setSipConfig({...sipConfig, auto_register: e.target.checked})} className="rounded" /><span className="text-sm text-gray-700">Registrar ao salvar</span></label>
              <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">O CRM Ã© somente o discador do atendente. Ele registra o ramal no Asterisk via WSS; quem escolhe a operadora e a rota da ligaÃ§Ã£o Ã© o Asterisk.</div>
            </div>
            <div className="flex gap-3 mt-6">
              <button onClick={() => setShowConfig(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={saveConfig} className="btn-primary flex-1">Salvar e Registrar</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
