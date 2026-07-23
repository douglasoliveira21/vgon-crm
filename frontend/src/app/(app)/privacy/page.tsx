'use client'

import { useCallback, useEffect, useState } from 'react'
import { Ban, CheckCircle2, Clock3, Plus, RefreshCw, ShieldCheck, Trash2 } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '@/lib/api'

type PrivacyRequest = {
  id: string
  contact_name?: string
  requester_name?: string
  requester_email?: string
  request_type: 'access' | 'correction' | 'deletion'
  status: string
  due_at?: string
  created_at: string
}

type Suppression = {
  id: string
  channel: 'email' | 'whatsapp'
  destination: string
  reason?: string
  source?: string
  created_at: string
}

const requestLabels = { access: 'Acesso aos dados', correction: 'Correção', deletion: 'Exclusão' }

export default function PrivacyPage() {
  const [requests, setRequests] = useState<PrivacyRequest[]>([])
  const [suppressions, setSuppressions] = useState<Suppression[]>([])
  const [retentionDays, setRetentionDays] = useState(730)
  const [retentionActive, setRetentionActive] = useState(false)
  const [loading, setLoading] = useState(true)
  const [requestForm, setRequestForm] = useState({ request_type: 'access', requester_name: '', requester_email: '', notes: '' })
  const [suppressionForm, setSuppressionForm] = useState({ channel: 'email', destination: '', reason: '' })

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [requestResponse, suppressionResponse, retentionResponse] = await Promise.all([
        api.get('/privacy/requests'),
        api.get('/campaigns/suppressions'),
        api.get('/privacy/retention'),
      ])
      setRequests(requestResponse.data.requests || [])
      setSuppressions(suppressionResponse.data.suppressions || [])
      const policy = (retentionResponse.data.policies || []).find((item: any) => item.resource_type === 'inactive_contacts')
      if (policy) {
        setRetentionDays(policy.retention_days)
        setRetentionActive(policy.is_active)
      }
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao carregar controles de privacidade')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  const createRequest = async () => {
    if (!requestForm.requester_name && !requestForm.requester_email) {
      toast.error('Informe o nome ou e-mail do solicitante')
      return
    }
    try {
      await api.post('/privacy/requests', requestForm)
      setRequestForm({ request_type: 'access', requester_name: '', requester_email: '', notes: '' })
      toast.success('Solicitação registrada com prazo de 15 dias')
      loadData()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao registrar solicitação')
    }
  }

  const completeRequest = async (request: PrivacyRequest) => {
    try {
      await api.put(`/privacy/requests/${request.id}`, { status: 'completed', notes: 'Atendimento concluído' })
      toast.success('Solicitação concluída')
      loadData()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao concluir solicitação')
    }
  }

  const saveRetention = async () => {
    try {
      await api.put('/privacy/retention', {
        resource_type: 'inactive_contacts',
        retention_days: retentionDays,
        is_active: retentionActive,
      })
      toast.success('Política de retenção atualizada')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao salvar retenção')
    }
  }

  const addSuppression = async () => {
    if (!suppressionForm.destination.trim()) {
      toast.error('Informe o telefone ou e-mail')
      return
    }
    try {
      await api.post('/campaigns/suppressions', suppressionForm)
      setSuppressionForm({ ...suppressionForm, destination: '', reason: '' })
      toast.success('Destino adicionado à lista de supressão')
      loadData()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao adicionar supressão')
    }
  }

  const removeSuppression = async (id: string) => {
    try {
      await api.delete(`/campaigns/suppressions/${id}`)
      toast.success('Supressão removida')
      loadData()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao remover supressão')
    }
  }

  return (
    <div className="mx-auto max-w-6xl p-6">
      <div className="mb-7 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Privacidade e LGPD</h1>
          <p className="mt-1 text-sm text-gray-500">Solicitações dos titulares, retenção e bloqueios de marketing.</p>
        </div>
        <button type="button" onClick={loadData} className="btn-secondary" disabled={loading}>
          <RefreshCw size={16} className={loading ? 'animate-spin' : ''} /> Atualizar
        </button>
      </div>

      <div className="grid gap-5 lg:grid-cols-2">
        <section className="card p-5">
          <div className="mb-4 flex items-center gap-2">
            <ShieldCheck size={19} className="text-primary-600" />
            <h2 className="font-semibold text-gray-900 dark:text-white">Nova solicitação formal</h2>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <select className="input" value={requestForm.request_type} onChange={(event) => setRequestForm({ ...requestForm, request_type: event.target.value })}>
              <option value="access">Acesso aos dados</option>
              <option value="correction">Correção</option>
              <option value="deletion">Exclusão</option>
            </select>
            <input className="input" placeholder="Nome do solicitante" value={requestForm.requester_name} onChange={(event) => setRequestForm({ ...requestForm, requester_name: event.target.value })} />
            <input className="input" type="email" placeholder="E-mail do solicitante" value={requestForm.requester_email} onChange={(event) => setRequestForm({ ...requestForm, requester_email: event.target.value })} />
            <input className="input" placeholder="Observações" value={requestForm.notes} onChange={(event) => setRequestForm({ ...requestForm, notes: event.target.value })} />
          </div>
          <button type="button" onClick={createRequest} className="btn-primary mt-3"><Plus size={16} /> Registrar</button>
        </section>

        <section className="card p-5">
          <div className="mb-4 flex items-center gap-2">
            <Clock3 size={19} className="text-primary-600" />
            <h2 className="font-semibold text-gray-900 dark:text-white">Retenção automática</h2>
          </div>
          <p className="mb-3 text-sm text-gray-500">Anonimiza dados pessoais de contatos inativos, preservando mensagens e indicadores sem identificação.</p>
          <div className="flex flex-wrap items-center gap-3">
            <input className="input w-36" type="number" min={30} value={retentionDays} onChange={(event) => setRetentionDays(Number(event.target.value))} />
            <span className="text-sm text-gray-500">dias de inatividade</span>
            <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input type="checkbox" checked={retentionActive} onChange={(event) => setRetentionActive(event.target.checked)} />
              Política ativa
            </label>
          </div>
          <button type="button" onClick={saveRetention} className="btn-primary mt-3">Salvar política</button>
        </section>
      </div>

      <section className="card mt-5 p-5">
        <div className="mb-4 flex items-center gap-2">
          <Ban size={19} className="text-red-600" />
          <h2 className="font-semibold text-gray-900 dark:text-white">Lista de supressão</h2>
        </div>
        <div className="mb-4 grid gap-3 sm:grid-cols-[160px_1fr_1fr_auto]">
          <select className="input" value={suppressionForm.channel} onChange={(event) => setSuppressionForm({ ...suppressionForm, channel: event.target.value })}>
            <option value="email">E-mail</option>
            <option value="whatsapp">WhatsApp</option>
          </select>
          <input className="input" placeholder="E-mail ou telefone" value={suppressionForm.destination} onChange={(event) => setSuppressionForm({ ...suppressionForm, destination: event.target.value })} />
          <input className="input" placeholder="Motivo" value={suppressionForm.reason} onChange={(event) => setSuppressionForm({ ...suppressionForm, reason: event.target.value })} />
          <button type="button" onClick={addSuppression} className="btn-primary"><Plus size={16} /> Adicionar</button>
        </div>
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {suppressions.map((item) => (
            <div key={item.id} className="flex items-center gap-3 py-3 text-sm">
              <span className="badge badge-gray">{item.channel === 'email' ? 'E-mail' : 'WhatsApp'}</span>
              <span className="font-medium text-gray-900 dark:text-white">{item.destination}</span>
              <span className="flex-1 text-gray-500">{item.reason || item.source || 'Sem motivo informado'}</span>
              <button type="button" onClick={() => removeSuppression(item.id)} className="p-2 text-gray-400 hover:text-red-600" title="Remover"><Trash2 size={16} /></button>
            </div>
          ))}
          {!loading && suppressions.length === 0 && <p className="py-5 text-center text-sm text-gray-500">Nenhum destino suprimido.</p>}
        </div>
      </section>

      <section className="card mt-5 p-5">
        <h2 className="mb-4 font-semibold text-gray-900 dark:text-white">Solicitações registradas</h2>
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {requests.map((request) => (
            <div key={request.id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
              <span className={`badge ${request.status === 'completed' ? 'badge-green' : 'badge-yellow'}`}>{request.status === 'completed' ? 'Concluída' : 'Pendente'}</span>
              <span className="font-medium text-gray-900 dark:text-white">{requestLabels[request.request_type]}</span>
              <span className="text-gray-500">{request.contact_name || request.requester_name || request.requester_email}</span>
              <span className="flex-1 text-xs text-gray-400">{request.due_at ? `Prazo: ${new Date(request.due_at).toLocaleDateString('pt-BR')}` : ''}</span>
              {request.status !== 'completed' && (
                <button type="button" onClick={() => completeRequest(request)} className="btn-secondary py-1.5 text-xs"><CheckCircle2 size={14} /> Concluir</button>
              )}
            </div>
          ))}
          {!loading && requests.length === 0 && <p className="py-5 text-center text-sm text-gray-500">Nenhuma solicitação registrada.</p>}
        </div>
      </section>
    </div>
  )
}
