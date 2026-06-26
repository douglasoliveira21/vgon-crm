'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  Phone,
  Plus,
  Trash2,
  Edit,
  Power,
  PowerOff,
  ArrowLeft,
  Clock,
  X,
} from 'lucide-react'

interface IVROption {
  digit: string
  label: string
  action: string
  destination: string
}

interface IVR {
  id: string
  name: string
  welcome_message: string
  business_hours_start: string
  business_hours_end: string
  off_hours_message: string
  timeout_message: string
  invalid_message: string
  max_retries: number
  options: IVROption[]
  is_active: boolean
  created_at: string
  updated_at: string
}

const defaultIVR: Omit<IVR, 'id' | 'created_at' | 'updated_at'> = {
  name: '',
  welcome_message: 'Bem-vindo. Escolha uma opção.',
  business_hours_start: '08:00',
  business_hours_end: '18:00',
  off_hours_message: 'Estamos fora do horário de atendimento.',
  timeout_message: 'Não entendi sua opção. Tente novamente.',
  invalid_message: 'Opção inválida. Tente novamente.',
  max_retries: 3,
  options: [],
  is_active: true,
}

const actionLabels: Record<string, string> = {
  extension: 'Ramal',
  queue: 'Fila',
  external: 'Número Externo',
  voicemail: 'Correio de Voz',
  ivr: 'Sub-menu URA',
}

const digitOptions = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '*']

export default function IVRPage() {
  const router = useRouter()
  const [ivrs, setIvrs] = useState<IVR[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingIVR, setEditingIVR] = useState<IVR | null>(null)
  const [form, setForm] = useState(defaultIVR)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  useEffect(() => {
    fetchIVRs()
  }, [])

  const fetchIVRs = async () => {
    try {
      const res = await api.get('/telephony/ivr')
      setIvrs(res.data.ivrs || [])
    } catch {
      toast.error('Erro ao carregar URAs')
    } finally {
      setLoading(false)
    }
  }

  const openCreate = () => {
    setEditingIVR(null)
    setForm({ ...defaultIVR, options: [] })
    setShowModal(true)
  }

  const openEdit = (ivr: IVR) => {
    setEditingIVR(ivr)
    setForm({
      name: ivr.name,
      welcome_message: ivr.welcome_message,
      business_hours_start: ivr.business_hours_start,
      business_hours_end: ivr.business_hours_end,
      off_hours_message: ivr.off_hours_message,
      timeout_message: ivr.timeout_message,
      invalid_message: ivr.invalid_message,
      max_retries: ivr.max_retries,
      options: ivr.options || [],
      is_active: ivr.is_active,
    })
    setShowModal(true)
  }

  const handleSave = async () => {
    if (!form.name.trim()) {
      toast.error('Nome é obrigatório')
      return
    }

    try {
      if (editingIVR) {
        await api.put(`/telephony/ivr/${editingIVR.id}`, form)
        toast.success('URA atualizada!')
      } else {
        await api.post('/telephony/ivr', form)
        toast.success('URA criada!')
      }
      setShowModal(false)
      fetchIVRs()
    } catch {
      toast.error('Erro ao salvar URA')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await api.delete(`/telephony/ivr/${id}`)
      toast.success('URA excluída')
      setDeleteConfirm(null)
      fetchIVRs()
    } catch {
      toast.error('Erro ao excluir URA')
    }
  }

  const addOption = () => {
    const usedDigits = form.options.map(o => o.digit)
    const nextDigit = digitOptions.find(d => !usedDigits.includes(d)) || '1'
    setForm({
      ...form,
      options: [...form.options, { digit: nextDigit, label: '', action: 'extension', destination: '' }],
    })
  }

  const removeOption = (index: number) => {
    setForm({
      ...form,
      options: form.options.filter((_, i) => i !== index),
    })
  }

  const updateOption = (index: number, field: keyof IVROption, value: string) => {
    const newOptions = [...form.options]
    newOptions[index] = { ...newOptions[index], [field]: value }
    setForm({ ...form, options: newOptions })
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <button onClick={() => router.push('/calls')} className="p-2 hover:bg-gray-100 rounded-lg">
            <ArrowLeft size={20} className="text-gray-500" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">URA (IVR)</h1>
            <p className="text-sm text-gray-500">Gerencie as URAs de atendimento automático</p>
          </div>
        </div>
        <button onClick={openCreate} className="btn-primary flex items-center gap-2">
          <Plus size={16} /> Nova URA
        </button>
      </div>

      {/* List */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Carregando...</div>
      ) : ivrs.length === 0 ? (
        <div className="card p-12 text-center">
          <Phone size={48} className="text-gray-300 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-700 mb-2">Nenhuma URA cadastrada</h3>
          <p className="text-sm text-gray-500 mb-4">
            URAs permitem direcionar chamadas automaticamente com menus interativos.
          </p>
          <button onClick={openCreate} className="btn-primary">
            <Plus size={16} className="inline mr-1" /> Criar primeira URA
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {ivrs.map((ivr) => (
            <div key={ivr.id} className="card p-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className={`w-10 h-10 rounded-full flex items-center justify-center ${ivr.is_active ? 'bg-green-100' : 'bg-gray-100'}`}>
                  {ivr.is_active ? <Power size={18} className="text-green-600" /> : <PowerOff size={18} className="text-gray-400" />}
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-gray-900">{ivr.name}</h3>
                  <div className="flex items-center gap-3 mt-1">
                    <span className={`text-xs px-2 py-0.5 rounded-full ${ivr.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                      {ivr.is_active ? 'Ativa' : 'Inativa'}
                    </span>
                    <span className="text-xs text-gray-400 flex items-center gap-1">
                      <Clock size={12} /> {ivr.business_hours_start} - {ivr.business_hours_end}
                    </span>
                    <span className="text-xs text-gray-400">
                      {(ivr.options || []).length} opções
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button onClick={() => openEdit(ivr)} className="p-2 hover:bg-gray-100 rounded-lg text-gray-500" title="Editar">
                  <Edit size={16} />
                </button>
                <button onClick={() => setDeleteConfirm(ivr.id)} className="p-2 hover:bg-red-50 rounded-lg text-red-500" title="Excluir">
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-sm">
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Excluir URA?</h3>
            <p className="text-sm text-gray-500 mb-6">Esta ação não pode ser desfeita. Todas as configurações desta URA serão perdidas.</p>
            <div className="flex gap-3">
              <button onClick={() => setDeleteConfirm(null)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={() => handleDelete(deleteConfirm)} className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 text-sm font-medium">Excluir</button>
            </div>
          </div>
        </div>
      )}

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">
                {editingIVR ? 'Editar URA' : 'Nova URA'}
              </h3>
              <button onClick={() => setShowModal(false)} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>

            <div className="space-y-4">
              {/* Name */}
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Nome da URA</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className="input"
                  placeholder="Ex: Menu Principal"
                />
              </div>

              {/* Welcome Message */}
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Mensagem de Boas-vindas</label>
                <textarea
                  value={form.welcome_message}
                  onChange={(e) => setForm({ ...form, welcome_message: e.target.value })}
                  className="input min-h-[80px] resize-y"
                  placeholder="Mensagem reproduzida ao atender a chamada"
                />
              </div>

              {/* Business Hours */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Horário Início</label>
                  <input
                    type="time"
                    value={form.business_hours_start}
                    onChange={(e) => setForm({ ...form, business_hours_start: e.target.value })}
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Horário Fim</label>
                  <input
                    type="time"
                    value={form.business_hours_end}
                    onChange={(e) => setForm({ ...form, business_hours_end: e.target.value })}
                    className="input"
                  />
                </div>
              </div>

              {/* Off-hours Message */}
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Mensagem Fora do Horário</label>
                <textarea
                  value={form.off_hours_message}
                  onChange={(e) => setForm({ ...form, off_hours_message: e.target.value })}
                  className="input min-h-[60px] resize-y"
                  placeholder="Mensagem quando fora do expediente"
                />
              </div>

              {/* Timeout / Invalid Messages */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Mensagem de Timeout</label>
                  <input
                    type="text"
                    value={form.timeout_message}
                    onChange={(e) => setForm({ ...form, timeout_message: e.target.value })}
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Mensagem Opção Inválida</label>
                  <input
                    type="text"
                    value={form.invalid_message}
                    onChange={(e) => setForm({ ...form, invalid_message: e.target.value })}
                    className="input"
                  />
                </div>
              </div>

              {/* Max Retries */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Máximo de Tentativas</label>
                  <input
                    type="number"
                    min={1}
                    max={10}
                    value={form.max_retries}
                    onChange={(e) => setForm({ ...form, max_retries: parseInt(e.target.value) || 3 })}
                    className="input"
                  />
                </div>
                <div className="flex items-end pb-1">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={form.is_active}
                      onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                      className="rounded"
                    />
                    <span className="text-sm text-gray-700">URA Ativa</span>
                  </label>
                </div>
              </div>

              {/* Options */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-xs font-medium text-gray-700">Opções do Menu</label>
                  <button
                    onClick={addOption}
                    className="text-xs text-primary-600 hover:text-primary-700 font-medium flex items-center gap-1"
                  >
                    <Plus size={14} /> Adicionar opção
                  </button>
                </div>

                {form.options.length === 0 ? (
                  <div className="text-center py-6 bg-gray-50 rounded-lg">
                    <p className="text-sm text-gray-400">Nenhuma opção adicionada</p>
                    <button onClick={addOption} className="text-xs text-primary-600 mt-1 hover:underline">
                      + Adicionar primeira opção
                    </button>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {form.options.map((opt, idx) => (
                      <div key={idx} className="flex items-center gap-2 p-3 bg-gray-50 rounded-lg">
                        {/* Digit */}
                        <select
                          value={opt.digit}
                          onChange={(e) => updateOption(idx, 'digit', e.target.value)}
                          className="input w-16 text-center font-mono text-sm"
                        >
                          {digitOptions.map((d) => (
                            <option key={d} value={d}>{d}</option>
                          ))}
                        </select>

                        {/* Label */}
                        <input
                          type="text"
                          value={opt.label}
                          onChange={(e) => updateOption(idx, 'label', e.target.value)}
                          className="input flex-1 text-sm"
                          placeholder="Descrição (ex: Suporte Técnico)"
                        />

                        {/* Action */}
                        <select
                          value={opt.action}
                          onChange={(e) => updateOption(idx, 'action', e.target.value)}
                          className="input w-40 text-sm"
                        >
                          {Object.entries(actionLabels).map(([value, label]) => (
                            <option key={value} value={value}>{label}</option>
                          ))}
                        </select>

                        {/* Destination */}
                        <input
                          type="text"
                          value={opt.destination}
                          onChange={(e) => updateOption(idx, 'destination', e.target.value)}
                          className="input w-32 text-sm"
                          placeholder="Destino"
                        />

                        {/* Remove */}
                        <button
                          onClick={() => removeOption(idx)}
                          className="p-1.5 text-red-400 hover:text-red-600 hover:bg-red-50 rounded"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="flex gap-3 mt-6 pt-4 border-t">
              <button onClick={() => setShowModal(false)} className="btn-secondary flex-1">Cancelar</button>
              <button onClick={handleSave} className="btn-primary flex-1">
                {editingIVR ? 'Salvar Alterações' : 'Criar URA'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
