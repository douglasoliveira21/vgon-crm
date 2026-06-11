'use client'

import { useEffect, useState } from 'react'
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
  MessageCircle,
  X,
  Mail,
} from 'lucide-react'

interface WhatsAppInstance {
  id: string
  instance_name: string
  status: string
  phone_number?: string
  qrcode?: string
  connected_at?: string
  created_at: string
}

export default function ChannelsPage() {
  const [instances, setInstances] = useState<WhatsAppInstance[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [activeTab, setActiveTab] = useState('whatsapp')
  const [showEmailModal, setShowEmailModal] = useState(false)
  const [instanceName, setInstanceName] = useState('')
  const [creating, setCreating] = useState(false)
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [qrInstanceId, setQrInstanceId] = useState<string | null>(null)

  useEffect(() => {
    fetchInstances()

    // Listen for status updates
    const handleStatus = (data: any) => {
      setInstances((prev) =>
        prev.map((inst) =>
          inst.instance_name === data.instance_name
            ? { ...inst, status: data.status }
            : inst
        )
      )
    }

    const handleQRCode = (data: any) => {
      if (data.qrcode) {
        setQrCode(data.qrcode)
      }
    }

    wsService.on('channel_status', handleStatus)
    wsService.on('qrcode_update', handleQRCode)

    return () => {
      wsService.off('channel_status', handleStatus)
      wsService.off('qrcode_update', handleQRCode)
    }
  }, [])

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

      // Show QR code
      if (instance.qrcode) {
        setQrCode(instance.qrcode)
        setQrInstanceId(instance.id)
      }

      setInstanceName('')
      setShowCreateModal(false)
      toast.success('Instância criada! Escaneie o QR Code.')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao criar instância')
    } finally {
      setCreating(false)
    }
  }

  const fetchQRCode = async (instanceId: string) => {
    try {
      const response = await api.get(`/whatsapp/instances/${instanceId}/qrcode`)
      setQrCode(response.data.qrcode)
      setQrInstanceId(instanceId)
    } catch (error: any) {
      toast.error('Erro ao buscar QR Code')
    }
  }

  const checkStatus = async (instanceId: string) => {
    try {
      const response = await api.get(`/whatsapp/instances/${instanceId}/status`)
      setInstances((prev) =>
        prev.map((inst) =>
          inst.id === instanceId ? { ...inst, status: response.data.status } : inst
        )
      )
      toast.success(`Status: ${response.data.status}`)
    } catch (error) {
      toast.error('Erro ao verificar status')
    }
  }

  const disconnectInstance = async (instanceId: string) => {
    try {
      await api.post(`/whatsapp/instances/${instanceId}/disconnect`)
      setInstances((prev) =>
        prev.map((inst) =>
          inst.id === instanceId ? { ...inst, status: 'disconnected' } : inst
        )
      )
      toast.success('Instância desconectada')
    } catch (error) {
      toast.error('Erro ao desconectar')
    }
  }

  const deleteInstance = async (instanceId: string) => {
    if (!confirm('Tem certeza que deseja remover esta instância?')) return

    try {
      await api.delete(`/whatsapp/instances/${instanceId}`)
      setInstances((prev) => prev.filter((inst) => inst.id !== instanceId))
      toast.success('Instância removida')
    } catch (error) {
      toast.error('Erro ao remover')
    }
  }

  const syncContacts = async (instanceId: string) => {
    try {
      const response = await api.post(`/whatsapp/instances/${instanceId}/sync-contacts`)
      toast.success(`${response.data.count} contatos sincronizados! Fotos sendo baixadas em segundo plano.`)
    } catch (error) {
      toast.error('Erro ao sincronizar contatos')
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

    const s = statusMap[status] || statusMap.disconnected
    const Icon = s.icon

    return (
      <span className={`badge ${s.class} gap-1`}>
        <Icon size={12} className={status === 'connecting' ? 'animate-spin' : ''} />
        {s.label}
      </span>
    )
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Canais & Integrações</h1>
          <p className="text-gray-500 mt-1">Gerencie suas conexões de atendimento</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn-primary"
        >
          <Plus size={18} />
          Conectar WhatsApp
        </button>
      </div>

      {/* Channel type tabs */}
      <div className="flex gap-2 mb-6">
        <button
          onClick={() => setActiveTab('whatsapp')}
          className={`px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 transition-colors ${
            activeTab === 'whatsapp' ? 'bg-green-100 text-green-700' : 'text-gray-500 hover:bg-gray-100'
          }`}
        >
          <MessageCircle size={16} /> WhatsApp
        </button>
        <button
          onClick={() => setActiveTab('email')}
          className={`px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 transition-colors ${
            activeTab === 'email' ? 'bg-blue-100 text-blue-700' : 'text-gray-500 hover:bg-gray-100'
          }`}
        >
          <Mail size={16} /> E-mail
        </button>
      </div>

      {/* WhatsApp Tab */}
      {activeTab === 'whatsapp' && (
      <>
      {/* QR Code Display */}
      {qrCode && (
        <div className="card p-8 mb-6 text-center">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Escaneie o QR Code</h3>
            <button
              onClick={() => { setQrCode(null); setQrInstanceId(null) }}
              className="text-gray-400 hover:text-gray-600"
            >
              <X size={20} />
            </button>
          </div>
          <p className="text-sm text-gray-500 mb-6">
            Abra o WhatsApp no seu celular e escaneie o código abaixo
          </p>
          <div className="inline-block p-4 bg-white rounded-2xl shadow-lg border border-gray-100">
            <img
              src={qrCode.startsWith('data:') ? qrCode : `data:image/png;base64,${qrCode}`}
              alt="QR Code WhatsApp"
              className="w-64 h-64"
            />
          </div>
          <div className="mt-4 flex items-center justify-center gap-3">
            <button
              onClick={() => qrInstanceId && fetchQRCode(qrInstanceId)}
              className="btn-secondary text-sm"
            >
              <RefreshCw size={14} />
              Atualizar QR Code
            </button>
          </div>
          <p className="text-xs text-gray-400 mt-4">
            O QR Code expira em poucos segundos. Atualize se necessário.
          </p>
        </div>
      )}

      {/* Instances List */}
      <div className="space-y-4">
        {instances.map((instance) => (
          <div key={instance.id} className="card p-5 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                instance.status === 'connected' ? 'bg-green-100' : 'bg-gray-100'
              }`}>
                <MessageCircle size={22} className={
                  instance.status === 'connected' ? 'text-green-600' : 'text-gray-400'
                } />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">{instance.instance_name}</h3>
                <div className="flex items-center gap-2 mt-1">
                  {getStatusBadge(instance.status)}
                  {instance.phone_number && (
                    <span className="text-xs text-gray-400">{instance.phone_number}</span>
                  )}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              {instance.status !== 'connected' && (
                <button
                  onClick={() => fetchQRCode(instance.id)}
                  className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg transition-colors"
                  title="Ver QR Code"
                >
                  <QrCode size={18} />
                </button>
              )}
              {instance.status === 'connected' && (
                <button
                  onClick={() => syncContacts(instance.id)}
                  className="p-2 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded-lg transition-colors"
                  title="Sincronizar contatos e fotos"
                >
                  <RefreshCw size={18} />
                </button>
              )}
              <button
                onClick={() => checkStatus(instance.id)}
                className="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                title="Verificar status"
              >
                <RefreshCw size={18} />
              </button>
              {instance.status === 'connected' && (
                <button
                  onClick={() => disconnectInstance(instance.id)}
                  className="p-2 text-gray-400 hover:text-yellow-600 hover:bg-yellow-50 rounded-lg transition-colors"
                  title="Desconectar"
                >
                  <Power size={18} />
                </button>
              )}
              <button
                onClick={() => deleteInstance(instance.id)}
                className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                title="Remover"
              >
                <Trash2 size={18} />
              </button>
            </div>
          </div>
        ))}

        {instances.length === 0 && !loading && (
          <div className="card p-12 text-center">
            <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <MessageCircle size={28} className="text-green-600" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">Conecte seu WhatsApp</h3>
            <p className="text-gray-500 text-sm mb-6 max-w-sm mx-auto">
              Crie uma integração para começar a receber e enviar mensagens pelo WhatsApp
            </p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="btn-primary inline-flex"
            >
              <Plus size={18} />
              Criar integração WhatsApp
            </button>
          </div>
        )}
      </div>
      </>
      )}

      {/* Email Tab */}
      {activeTab === 'email' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between mb-4">
            <p className="text-gray-500">Configure integrações de e-mail para receber e responder mensagens</p>
            <button onClick={() => setShowEmailModal(true)} className="btn-primary">
              <Plus size={18} /> Conectar E-mail
            </button>
          </div>

          <div className="card p-12 text-center">
            <Mail size={40} className="text-blue-300 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">Integração de E-mail</h3>
            <p className="text-gray-500 text-sm max-w-md mx-auto mb-6">
              Conecte sua conta de e-mail para receber e responder mensagens diretamente pelo CRM.
              Suportamos Gmail, Outlook e IMAP genérico.
            </p>
            <button onClick={() => setShowEmailModal(true)} className="btn-primary inline-flex">
              <Plus size={18} /> Conectar E-mail
            </button>
          </div>
        </div>
      )}

      {/* Email Setup Modal */}
      {showEmailModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-semibold text-gray-900">Conectar E-mail</h3>
              <button onClick={() => setShowEmailModal(false)} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>

            <p className="text-sm text-gray-500 mb-6">Selecione seu provedor de e-mail:</p>

            <div className="space-y-3">
              <button className="w-full flex items-center gap-4 p-4 border border-gray-200 rounded-xl hover:border-blue-500 hover:bg-blue-50 transition-colors text-left">
                <img src="https://www.google.com/gmail/about/static-2.0/images/logo-gmail.png" alt="Gmail" className="w-8 h-8 object-contain" onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }} />
                <div>
                  <p className="font-medium text-gray-900">Gmail</p>
                  <p className="text-xs text-gray-500">Conectar via Google OAuth</p>
                </div>
                <span className="ml-auto badge badge-yellow">Em breve</span>
              </button>

              <button className="w-full flex items-center gap-4 p-4 border border-gray-200 rounded-xl hover:border-blue-500 hover:bg-blue-50 transition-colors text-left">
                <Mail size={24} className="text-blue-600" />
                <div>
                  <p className="font-medium text-gray-900">Outlook / Microsoft 365</p>
                  <p className="text-xs text-gray-500">Conectar via Microsoft OAuth</p>
                </div>
                <span className="ml-auto badge badge-yellow">Em breve</span>
              </button>

              <button className="w-full flex items-center gap-4 p-4 border border-gray-200 rounded-xl hover:border-blue-500 hover:bg-blue-50 transition-colors text-left">
                <Mail size={24} className="text-gray-600" />
                <div>
                  <p className="font-medium text-gray-900">IMAP / SMTP</p>
                  <p className="text-xs text-gray-500">Conectar qualquer provedor via IMAP</p>
                </div>
                <span className="ml-auto badge badge-yellow">Em breve</span>
              </button>
            </div>

            <div className="mt-6 p-4 bg-gray-50 rounded-lg">
              <p className="text-xs text-gray-500">
                💡 A integração de e-mail está em desenvolvimento. Em breve você poderá receber e responder
                e-mails diretamente pelo CRM, na mesma interface das conversas WhatsApp.
              </p>
            </div>

            <button onClick={() => setShowEmailModal(false)} className="btn-secondary w-full mt-4">
              Fechar
            </button>
          </div>
        </div>
      )}

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              Criar integração WhatsApp
            </h3>
            <p className="text-sm text-gray-500 mb-6">
              Informe um nome para a instância. Após criar, escaneie o QR Code para conectar.
            </p>

            <form onSubmit={createInstance}>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Nome da instância
                </label>
                <input
                  type="text"
                  value={instanceName}
                  onChange={(e) => setInstanceName(e.target.value)}
                  placeholder="ex: suporte-principal"
                  className="input"
                  required
                />
                <p className="text-xs text-gray-400 mt-1">
                  Use apenas letras minúsculas, números e hífens
                </p>
              </div>

              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="btn-secondary flex-1"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={creating || !instanceName.trim()}
                  className="btn-primary flex-1"
                >
                  {creating ? (
                    <Loader2 size={18} className="animate-spin" />
                  ) : (
                    <Plus size={18} />
                  )}
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
