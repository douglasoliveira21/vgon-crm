'use client'

import { useEffect, useRef, useState } from 'react'
import api from '@/lib/api'
import wsService from '@/lib/websocket'
import toast from 'react-hot-toast'
import {
  Plus,
  Wifi,
  WifiOff,
  Loader2,
  QrCode,
  Trash2,
  RefreshCw,
  Power,
  X,
} from 'lucide-react'
import { ChannelIcon } from '@/components/channel-icon'
import { SafeImage } from '@/components/safe-image'

const QR_CODE_TTL_SECONDS = 45

interface WhatsAppInstance {
  id: string
  instance_name: string
  status: string
  phone_number?: string
  qrcode?: string
  last_status_check_at?: string
  last_status_error?: string
  disconnected_alerted_at?: string
  created_at: string
}

interface EmailChannel {
  id: string
  name: string
  type: string
  status: string
  settings?: {
    provider?: string
    imap_host?: string
    imap_port?: number
    username?: string
    mailbox?: string
    last_uid?: number
  }
  created_at: string
}

const defaultEmailForm = {
  provider: 'imap',
  name: '',
  imap_host: '',
  imap_port: 993,
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  smtp_use_tls: false,
  username: '',
  password: '',
  mailbox: 'INBOX',
  use_tls: true,
  max_import: 500,
}

export default function ChannelsPage() {
  const [instances, setInstances] = useState<WhatsAppInstance[]>([])
  const [emailChannels, setEmailChannels] = useState<EmailChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('whatsapp')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEmailModal, setShowEmailModal] = useState(false)
  const [instanceName, setInstanceName] = useState('')
  const [creating, setCreating] = useState(false)
  const [savingEmail, setSavingEmail] = useState(false)
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [qrInstanceId, setQrInstanceId] = useState<string | null>(null)
  const [qrSecondsLeft, setQrSecondsLeft] = useState(QR_CODE_TTL_SECONDS)
  const [refreshingQRCode, setRefreshingQRCode] = useState(false)
  const [emailForm, setEmailForm] = useState(defaultEmailForm)
  const [showImapForm, setShowImapForm] = useState(false)
  const instancesRef = useRef<WhatsAppInstance[]>([])
  const qrInstanceIdRef = useRef<string | null>(null)

  const applyEmailProvider = (provider: 'gmail' | 'outlook' | 'imap') => {
    const currentName = emailForm.name
    if (provider === 'gmail') {
      setEmailForm({
        ...emailForm,
        provider,
        name: currentName || 'Gmail',
        imap_host: 'imap.gmail.com',
        imap_port: 993,
        mailbox: 'INBOX',
        use_tls: true,
      })
      return
    }
    if (provider === 'outlook') {
      setEmailForm({
        ...emailForm,
        provider,
        name: currentName || 'Outlook',
        imap_host: 'outlook.office365.com',
        imap_port: 993,
        mailbox: 'INBOX',
        use_tls: true,
      })
      return
    }
    setEmailForm({
      ...emailForm,
      provider,
      name: currentName === 'Gmail' || currentName === 'Outlook' ? '' : currentName,
      imap_host: '',
      imap_port: 993,
      smtp_host: '',
      smtp_port: 587,
      smtp_username: '',
      smtp_password: '',
      smtp_use_tls: false,
      mailbox: 'INBOX',
      use_tls: true,
    })
  }

  useEffect(() => {
    instancesRef.current = instances
  }, [instances])

  useEffect(() => {
    qrInstanceIdRef.current = qrInstanceId
  }, [qrInstanceId])

  useEffect(() => {
    fetchInstances()
    fetchEmailChannels()

    const handleStatus = (data: any) => {
      setInstances((prev) =>
        prev.map((inst) =>
          inst.instance_name === data.instance_name ? { ...inst, status: data.status } : inst
        )
      )
      if (data.status === 'connected' && instancesRef.current.find((inst) => inst.id === qrInstanceIdRef.current)?.instance_name === data.instance_name) {
        setQrCode(null)
        setQrInstanceId(null)
        setQrSecondsLeft(QR_CODE_TTL_SECONDS)
      }
    }
    const handleQRCode = (data: any) => {
      if (!data.qrcode) return
      setQrCode(data.qrcode)
      setQrSecondsLeft(QR_CODE_TTL_SECONDS)
      if (data.instance_name) {
        const matched = instancesRef.current.find((inst) => inst.instance_name === data.instance_name)
        if (matched) setQrInstanceId(matched.id)
      }
    }

    wsService.on('channel_status', handleStatus)
    wsService.on('qrcode_update', handleQRCode)
    return () => {
      wsService.off('channel_status', handleStatus)
      wsService.off('qrcode_update', handleQRCode)
    }
  }, [])

  useEffect(() => {
    if (!qrCode || !qrInstanceId) return

    setQrSecondsLeft(QR_CODE_TTL_SECONDS)
    const timer = window.setInterval(() => {
      setQrSecondsLeft((current) => {
        if (current <= 1) {
          fetchQRCode(qrInstanceId, true)
          return QR_CODE_TTL_SECONDS
        }
        return current - 1
      })
    }, 1000)

    return () => window.clearInterval(timer)
  }, [qrCode, qrInstanceId])

  const fetchInstances = async () => {
    try {
      const response = await api.get('/whatsapp/instances')
      setInstances(response.data.instances || [])
    } catch (error) {
      console.error('Error fetching instances:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchEmailChannels = async () => {
    try {
      const response = await api.get('/channels')
      setEmailChannels((response.data.channels || []).filter((channel: EmailChannel) => channel.type === 'email'))
    } catch (error) {
      console.error('Error fetching email channels:', error)
    }
  }

  const createInstance = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!instanceName.trim()) return

    setCreating(true)
    try {
      const response = await api.post('/whatsapp/instances', {
        instance_name: instanceName.trim().toLowerCase().replace(/\s+/g, '-'),
        channel_name: `WhatsApp - ${instanceName}`,
      })
      const instance = response.data
      setInstances((prev) => [instance, ...prev])
      if (instance.qrcode) {
        setQrCode(instance.qrcode)
        setQrInstanceId(instance.id)
        setQrSecondsLeft(QR_CODE_TTL_SECONDS)
      }
      setInstanceName('')
      setShowCreateModal(false)
      toast.success('Instancia criada! Escaneie o QR Code.')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao criar instancia')
    } finally {
      setCreating(false)
    }
  }

  const createEmailChannel = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!emailForm.name.trim() || !emailForm.imap_host.trim() || !emailForm.username.trim() || !emailForm.password.trim()) {
      toast.error('Preencha nome, servidor IMAP, usuario e senha')
      return
    }

    setSavingEmail(true)
    try {
      const response = await api.post('/channels/email', emailForm)
      toast.success(`E-mail conectado. ${response.data.imported || 0} mensagens importadas.`)
      if (response.data.warning) toast.error(`Falha ao sincronizar: ${response.data.warning}`)
      setShowEmailModal(false)
      setEmailForm(defaultEmailForm)
      fetchEmailChannels()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao conectar e-mail')
    } finally {
      setSavingEmail(false)
    }
  }

  const startEmailOAuth = async (provider: 'gmail' | 'outlook') => {
    try {
      const response = await api.get(`/channels/email/oauth/${provider}/start`)
      if (response.data.auth_url) {
        window.location.href = response.data.auth_url
      }
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao iniciar autorização')
    }
  }

  const fetchQRCode = async (instanceId: string, automatic = false) => {
    if (refreshingQRCode) return
    setRefreshingQRCode(true)
    try {
      const response = await api.get(`/whatsapp/instances/${instanceId}/qrcode`)
      setQrCode(response.data.qrcode)
      setQrInstanceId(instanceId)
      setQrSecondsLeft(response.data.expires_in || QR_CODE_TTL_SECONDS)
      setInstances((prev) => prev.map((inst) => inst.id === instanceId ? { ...inst, status: 'qr_code', qrcode: response.data.qrcode } : inst))
      if (!automatic) toast.success('QR Code atualizado')
    } catch {
      if (!automatic) toast.error('Erro ao buscar QR Code')
    } finally {
      setRefreshingQRCode(false)
    }
  }

  const checkStatus = async (instanceId: string) => {
    try {
      const response = await api.get(`/whatsapp/instances/${instanceId}/status`)
      setInstances((prev) => prev.map((inst) => inst.id === instanceId ? { ...inst, status: response.data.status } : inst))
      toast.success(`Status: ${response.data.status}`)
    } catch {
      toast.error('Erro ao verificar status')
    }
  }

  const disconnectInstance = async (instanceId: string) => {
    try {
      await api.post(`/whatsapp/instances/${instanceId}/disconnect`)
      setInstances((prev) => prev.map((inst) => inst.id === instanceId ? { ...inst, status: 'disconnected' } : inst))
      toast.success('Instancia desconectada')
    } catch {
      toast.error('Erro ao desconectar')
    }
  }

  const deleteInstance = async (instanceId: string) => {
    if (!confirm('Tem certeza que deseja remover esta instancia?')) return
    try {
      await api.delete(`/whatsapp/instances/${instanceId}`)
      setInstances((prev) => prev.filter((inst) => inst.id !== instanceId))
      toast.success('Instancia removida')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  const syncContacts = async (instanceId: string) => {
    try {
      const response = await api.post(`/whatsapp/instances/${instanceId}/sync-contacts`)
      toast.success(`${response.data.count} contatos sincronizados!`)
    } catch {
      toast.error('Erro ao sincronizar contatos')
    }
  }

  const syncEmailChannel = async (channelId: string) => {
    try {
      const response = await api.post(`/channels/email/${channelId}/sync`)
      toast.success(`${response.data.imported || 0} e-mails importados`)
      fetchEmailChannels()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao sincronizar e-mail')
    }
  }

  const deleteEmailChannel = async (channelId: string) => {
    if (!confirm('Tem certeza que deseja remover este canal de e-mail? As conversas importadas permanecerao no historico.')) return
    try {
      await api.delete(`/channels/email/${channelId}`)
      setEmailChannels((prev) => prev.filter((channel) => channel.id !== channelId))
      toast.success('Canal de e-mail removido')
    } catch {
      toast.error('Erro ao remover e-mail')
    }
  }

  const getStatusBadge = (status: string) => {
    const statusMap: Record<string, { label: string; class: string; icon: any }> = {
      connected: { label: 'Conectado', class: 'badge-green', icon: Wifi },
      connecting: { label: 'Conectando', class: 'badge-yellow', icon: Loader2 },
      qr_code: { label: 'Aguardando QR Code', class: 'badge-yellow', icon: QrCode },
      disconnected: { label: 'Desconectado', class: 'badge-red', icon: WifiOff },
      error: { label: 'Erro', class: 'badge-red', icon: WifiOff },
    }
    const item = statusMap[status] || statusMap.disconnected
    const Icon = item.icon
    return (
      <span className={`badge ${item.class} gap-1`}>
        <Icon size={12} className={status === 'connecting' ? 'animate-spin' : ''} />
        {item.label}
      </span>
    )
  }

  return (
    <div className="mx-auto max-w-5xl p-4 sm:p-6">
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Canais & Integracoes</h1>
          <p className="text-gray-500 mt-1">Gerencie suas conexoes de atendimento</p>
        </div>
        {activeTab === 'whatsapp' ? (
          <button onClick={() => setShowCreateModal(true)} className="btn-primary">
            <Plus size={18} />
            Conectar WhatsApp
          </button>
        ) : (
          <button onClick={() => setShowEmailModal(true)} className="btn-primary">
            <Plus size={18} />
            Conectar E-mail
          </button>
        )}
      </div>

      <div className="flex gap-2 mb-6">
        <button
          onClick={() => setActiveTab('whatsapp')}
          className={`px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 transition-colors ${
            activeTab === 'whatsapp' ? 'bg-green-100 text-green-700' : 'text-gray-500 hover:bg-gray-100'
          }`}
        >
          <ChannelIcon type="whatsapp" size={16} /> WhatsApp
        </button>
        <button
          onClick={() => setActiveTab('email')}
          className={`px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 transition-colors ${
            activeTab === 'email' ? 'bg-blue-100 text-blue-700' : 'text-gray-500 hover:bg-gray-100'
          }`}
        >
          <ChannelIcon type="email" size={16} /> E-mail
        </button>
      </div>

      {activeTab === 'whatsapp' && (
        <>
          {qrCode && (
            <div className="card p-8 mb-6 text-center">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900">Escaneie o QR Code</h3>
                <button onClick={() => { setQrCode(null); setQrInstanceId(null); setQrSecondsLeft(QR_CODE_TTL_SECONDS) }} className="text-gray-400 hover:text-gray-600">
                  <X size={20} />
                </button>
              </div>
              <p className="text-sm text-gray-500 mb-2">Abra o WhatsApp no celular e escaneie o codigo abaixo.</p>
              <p className="text-xs text-gray-400 mb-6">
                Atualizacao automatica em {qrSecondsLeft}s. O QR Code expira rapidamente por seguranca.
              </p>
              <div className="inline-block p-4 bg-white rounded-2xl shadow-lg border border-gray-100">
                <SafeImage
                  src={qrCode.startsWith('data:') ? qrCode : `data:image/png;base64,${qrCode}`}
                  alt="QR Code para conectar o WhatsApp"
                  className="h-64 w-64"
                  fallback={<div className="flex h-64 w-64 flex-col items-center justify-center rounded-lg bg-gray-100 px-6 text-center text-sm text-gray-500"><QrCode size={36} className="mb-3" />QR Code indisponível. Clique em atualizar.</div>}
                />
              </div>
              <div className="mt-4 flex flex-col items-center gap-2">
                <div className="h-2 w-64 overflow-hidden rounded-full bg-gray-100">
                  <div
                    className="h-full rounded-full bg-primary-600 transition-all"
                    style={{ width: `${Math.max(0, Math.min(100, (qrSecondsLeft / QR_CODE_TTL_SECONDS) * 100))}%` }}
                  />
                </div>
                <button
                  onClick={() => qrInstanceId && fetchQRCode(qrInstanceId)}
                  disabled={!qrInstanceId || refreshingQRCode}
                  className="btn-secondary text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  <RefreshCw size={14} className={refreshingQRCode ? 'animate-spin' : ''} />
                  {refreshingQRCode ? 'Atualizando...' : 'Atualizar QR Code'}
                </button>
              </div>
            </div>
          )}

          <div className="space-y-4">
            {instances.map((instance) => (
              <div key={instance.id} className="card p-5 flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${instance.status === 'connected' ? 'bg-green-100' : 'bg-gray-100'}`}>
                    <ChannelIcon type="whatsapp" size={22} className={instance.status === 'connected' ? undefined : 'text-gray-400'} />
                  </div>
                  <div>
                    <h3 className="font-medium text-gray-900">{instance.instance_name}</h3>
                    <div className="flex items-center gap-2 mt-1">
                      {getStatusBadge(instance.status)}
                      {instance.phone_number && <span className="text-xs text-gray-400">{instance.phone_number}</span>}
                    </div>
                    {(instance.last_status_check_at || instance.last_status_error) && (
                      <p className="text-xs text-gray-400 mt-1">
                        Monitor: {instance.last_status_check_at ? new Date(instance.last_status_check_at).toLocaleString('pt-BR') : 'sem checagem'}
                        {instance.last_status_error ? ` - erro: ${instance.last_status_error}` : ''}
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {instance.status !== 'connected' && (
                    <button onClick={() => fetchQRCode(instance.id)} className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg transition-colors" title="Ver QR Code">
                      <QrCode size={18} />
                    </button>
                  )}
                  {instance.status === 'connected' && (
                    <button onClick={() => syncContacts(instance.id)} className="p-2 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded-lg transition-colors" title="Sincronizar contatos">
                      <RefreshCw size={18} />
                    </button>
                  )}
                  <button onClick={() => checkStatus(instance.id)} className="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors" title="Verificar status">
                    <RefreshCw size={18} />
                  </button>
                  {instance.status === 'connected' && (
                    <button onClick={() => disconnectInstance(instance.id)} className="p-2 text-gray-400 hover:text-yellow-600 hover:bg-yellow-50 rounded-lg transition-colors" title="Desconectar">
                      <Power size={18} />
                    </button>
                  )}
                  <button onClick={() => deleteInstance(instance.id)} className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors" title="Remover">
                    <Trash2 size={18} />
                  </button>
                </div>
              </div>
            ))}

            {instances.length === 0 && !loading && (
              <div className="card p-12 text-center">
                <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                  <ChannelIcon type="whatsapp" size={28} />
                </div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Conecte seu WhatsApp</h3>
                <p className="text-gray-500 text-sm mb-6 max-w-sm mx-auto">Crie uma integracao para receber e enviar mensagens pelo WhatsApp.</p>
                <button onClick={() => setShowCreateModal(true)} className="btn-primary inline-flex">
                  <Plus size={18} />
                  Criar integracao WhatsApp
                </button>
              </div>
            )}
          </div>
        </>
      )}

      {activeTab === 'email' && (
        <div className="space-y-4">
          <p className="text-gray-500">Conecte uma caixa IMAP para importar e-mails como conversas.</p>

          {emailChannels.map((channel) => (
            <div key={channel.id} className="card p-5 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${channel.status === 'connected' ? 'bg-blue-100' : 'bg-gray-100'}`}>
                  <ChannelIcon type="email" size={22} className={channel.status === 'connected' ? undefined : 'text-gray-400'} />
                </div>
                <div>
                  <h3 className="font-medium text-gray-900">{channel.name}</h3>
                  <div className="flex items-center gap-2 mt-1">
                    {getStatusBadge(channel.status)}
                    {channel.settings?.provider && <span className="text-xs text-gray-400 uppercase">{channel.settings.provider}</span>}
                    <span className="text-xs text-gray-400">{channel.settings?.username}</span>
                    <span className="text-xs text-gray-400">{channel.settings?.imap_host}:{channel.settings?.imap_port}</span>
                  </div>
                  {channel.settings?.last_uid ? <p className="text-xs text-gray-400 mt-1">Ultimo UID sincronizado: {channel.settings.last_uid}</p> : null}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button onClick={() => syncEmailChannel(channel.id)} className="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors" title="Sincronizar caixa de entrada">
                  <RefreshCw size={18} />
                </button>
                <button onClick={() => deleteEmailChannel(channel.id)} className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors" title="Remover">
                  <Trash2 size={18} />
                </button>
              </div>
            </div>
          ))}

          {emailChannels.length === 0 && !loading && (
            <div className="card p-12 text-center">
              <ChannelIcon type="email" size={40} className="mx-auto mb-4 text-blue-300" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">Conecte sua caixa de entrada</h3>
              <p className="text-gray-500 text-sm max-w-md mx-auto mb-6">Use IMAP para importar e-mails recebidos e exibi-los em Conversas.</p>
              <button onClick={() => setShowEmailModal(true)} className="btn-primary inline-flex">
                <Plus size={18} />
                Conectar E-mail
              </button>
            </div>
          )}
        </div>
      )}

      {showEmailModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-semibold text-gray-900">Conectar E-mail</h3>
              <button onClick={() => setShowEmailModal(false)} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>

            <div className="space-y-3 mb-5">
              <button
                type="button"
                onClick={() => startEmailOAuth('gmail')}
                className="w-full flex items-center justify-between rounded-xl border border-gray-200 p-4 text-left hover:border-blue-500 hover:bg-blue-50 transition-colors"
              >
                <div>
                  <p className="font-medium text-gray-900">Gmail</p>
                  <p className="text-xs text-gray-500">Abrir tela do Google e permitir leitura de e-mails</p>
                </div>
                <span className="text-sm font-medium text-blue-600">Conectar</span>
              </button>
              <button
                type="button"
                onClick={() => startEmailOAuth('outlook')}
                className="w-full flex items-center justify-between rounded-xl border border-gray-200 p-4 text-left hover:border-blue-500 hover:bg-blue-50 transition-colors"
              >
                <div>
                  <p className="font-medium text-gray-900">Outlook / Microsoft 365</p>
                  <p className="text-xs text-gray-500">Abrir tela da Microsoft e permitir leitura de e-mails</p>
                </div>
                <span className="text-sm font-medium text-blue-600">Conectar</span>
              </button>
              <button
                type="button"
                onClick={() => {
                  applyEmailProvider('imap')
                  setShowImapForm((value) => !value)
                }}
                className="w-full flex items-center justify-between rounded-xl border border-gray-200 p-4 text-left hover:border-gray-400 hover:bg-gray-50 transition-colors"
              >
                <div>
                  <p className="font-medium text-gray-900">Outro provedor IMAP</p>
                  <p className="text-xs text-gray-500">Preencher servidor, porta, usuário e senha manualmente</p>
                </div>
                <span className="text-sm font-medium text-gray-600">{showImapForm ? 'Ocultar' : 'Configurar'}</span>
              </button>
            </div>

            {showImapForm && (
            <form onSubmit={createEmailChannel} className="space-y-4 border-t border-gray-100 pt-5">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Nome do canal</label>
                <input className="input" value={emailForm.name} onChange={e => setEmailForm({ ...emailForm, name: e.target.value })} placeholder="ex: Suporte" required />
              </div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                <div className="col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Servidor IMAP</label>
                  <input className="input" value={emailForm.imap_host} onChange={e => setEmailForm({ ...emailForm, imap_host: e.target.value })} placeholder="imap.gmail.com" required />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Porta</label>
                  <input type="number" className="input" value={emailForm.imap_port} onChange={e => setEmailForm({ ...emailForm, imap_port: Number(e.target.value) })} required />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">E-mail / usuario</label>
                <input className="input" value={emailForm.username} onChange={e => setEmailForm({ ...emailForm, username: e.target.value })} placeholder="contato@empresa.com" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Senha ou senha de aplicativo</label>
                <input type="password" className="input" value={emailForm.password} onChange={e => setEmailForm({ ...emailForm, password: e.target.value })} required />
              </div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                <div className="col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Servidor SMTP</label>
                  <input className="input" value={emailForm.smtp_host} onChange={e => setEmailForm({ ...emailForm, smtp_host: e.target.value })} placeholder="smtp.seudominio.com" required />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Porta SMTP</label>
                  <input type="number" className="input" value={emailForm.smtp_port} onChange={e => setEmailForm({ ...emailForm, smtp_port: Number(e.target.value) })} required />
                </div>
              </div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Usuário SMTP</label>
                  <input className="input" value={emailForm.smtp_username} onChange={e => setEmailForm({ ...emailForm, smtp_username: e.target.value })} placeholder="vazio usa o mesmo e-mail" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Senha SMTP</label>
                  <input type="password" className="input" value={emailForm.smtp_password} onChange={e => setEmailForm({ ...emailForm, smtp_password: e.target.value })} placeholder="vazio usa a mesma senha" />
                </div>
              </div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Pasta</label>
                  <input className="input" value={emailForm.mailbox} onChange={e => setEmailForm({ ...emailForm, mailbox: e.target.value })} placeholder="INBOX" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Maximo por sync</label>
                  <input type="number" className="input" value={emailForm.max_import} onChange={e => setEmailForm({ ...emailForm, max_import: Number(e.target.value) })} />
                </div>
              </div>
              <label className="flex items-center gap-2 text-sm text-gray-700">
                <input type="checkbox" checked={emailForm.use_tls} onChange={e => setEmailForm({ ...emailForm, use_tls: e.target.checked, imap_port: e.target.checked ? 993 : 143 })} />
                Usar TLS/SSL
              </label>
              <label className="flex items-center gap-2 text-sm text-gray-700">
                <input type="checkbox" checked={emailForm.smtp_use_tls} onChange={e => setEmailForm({ ...emailForm, smtp_use_tls: e.target.checked, smtp_port: e.target.checked ? 465 : 587 })} />
                SMTP com SSL direto
              </label>
              <div className="p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
                Gmail e Outlook normalmente exigem senha de aplicativo quando a conta tem verificacao em duas etapas.
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => setShowEmailModal(false)} className="btn-secondary flex-1">Cancelar</button>
                <button type="submit" disabled={savingEmail} className="btn-primary flex-1">
                  {savingEmail ? <Loader2 size={18} className="animate-spin" /> : <Plus size={18} />}
                  {savingEmail ? 'Conectando...' : 'Conectar'}
                </button>
              </div>
            </form>
            )}
          </div>
        </div>
      )}

      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Criar integracao WhatsApp</h3>
            <p className="text-sm text-gray-500 mb-6">Informe um nome para a instancia. Apos criar, escaneie o QR Code para conectar.</p>

            <form onSubmit={createInstance}>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Nome da instancia</label>
                <input type="text" value={instanceName} onChange={(e) => setInstanceName(e.target.value)} placeholder="ex: suporte-principal" className="input" required />
                <p className="text-xs text-gray-400 mt-1">Use apenas letras minusculas, numeros e hifens</p>
              </div>

              <div className="flex gap-3">
                <button type="button" onClick={() => setShowCreateModal(false)} className="btn-secondary flex-1">Cancelar</button>
                <button type="submit" disabled={creating || !instanceName.trim()} className="btn-primary flex-1">
                  {creating ? <Loader2 size={18} className="animate-spin" /> : <Plus size={18} />}
                  {creating ? 'Criando...' : 'Criar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
