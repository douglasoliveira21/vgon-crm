'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  Building2,
  Plus,
  Search,
  Edit,
  Power,
  Users,
  MessageSquare,
  X,
} from 'lucide-react'

interface Tenant {
  id: string
  name: string
  slug: string
  plan: string
  max_users: number
  max_channels: number
  is_active: boolean
  created_at: string
  user_count: number
  conversation_count: number
}

interface CreateTenantForm {
  company_name: string
  slug: string
  admin_name: string
  admin_email: string
  admin_password: string
  plan: string
  max_users: number
  max_channels: number
}

interface EditTenantForm {
  name: string
  plan: string
  max_users: number
  max_channels: number
  is_active: boolean
}

export default function TenantsPage() {
  const router = useRouter()
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null)
  const [createForm, setCreateForm] = useState<CreateTenantForm>({
    company_name: '',
    slug: '',
    admin_name: '',
    admin_email: '',
    admin_password: '',
    plan: 'basic',
    max_users: 5,
    max_channels: 2,
  })
  const [editForm, setEditForm] = useState<EditTenantForm>({
    name: '',
    plan: '',
    max_users: 0,
    max_channels: 0,
    is_active: true,
  })
  const [submitting, setSubmitting] = useState(false)

  const fetchTenants = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (search) params.append('search', search)
      if (statusFilter) params.append('status', statusFilter)

      const response = await api.get(`/admin/tenants?${params.toString()}`)
      setTenants(response.data.tenants || [])
    } catch (error) {
      toast.error('Erro ao carregar empresas')
    } finally {
      setLoading(false)
    }
  }, [search, statusFilter])

  useEffect(() => {
    fetchTenants()
  }, [fetchTenants])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createForm.company_name || !createForm.admin_email || !createForm.admin_password) {
      toast.error('Preencha todos os campos obrigatórios')
      return
    }

    setSubmitting(true)
    try {
      await api.post('/admin/tenants', createForm)
      toast.success('Empresa criada com sucesso!')
      setShowCreateModal(false)
      setCreateForm({
        company_name: '',
        slug: '',
        admin_name: '',
        admin_email: '',
        admin_password: '',
        plan: 'basic',
        max_users: 5,
        max_channels: 2,
      })
      fetchTenants()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao criar empresa')
    } finally {
      setSubmitting(false)
    }
  }

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingTenant) return

    setSubmitting(true)
    try {
      await api.put(`/admin/tenants/${editingTenant.id}`, editForm)
      toast.success('Empresa atualizada com sucesso!')
      setShowEditModal(false)
      setEditingTenant(null)
      fetchTenants()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao atualizar empresa')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeactivate = async (tenant: Tenant) => {
    if (!confirm(`Deseja realmente desativar a empresa "${tenant.name}"?`)) return

    try {
      await api.delete(`/admin/tenants/${tenant.id}`)
      toast.success('Empresa desativada com sucesso!')
      fetchTenants()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao desativar empresa')
    }
  }

  const openEditModal = (tenant: Tenant) => {
    setEditingTenant(tenant)
    setEditForm({
      name: tenant.name,
      plan: tenant.plan,
      max_users: tenant.max_users,
      max_channels: tenant.max_channels,
      is_active: tenant.is_active,
    })
    setShowEditModal(true)
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    })
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Empresas (Tenants)</h1>
          <p className="text-gray-400 mt-1">Gerencie todas as empresas da plataforma</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          Nova Empresa
        </button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 mb-6">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Buscar por nome ou slug..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
        >
          <option value="">Todos os status</option>
          <option value="active">Ativas</option>
          <option value="inactive">Inativas</option>
        </select>
      </div>

      {/* Table */}
      <div className="bg-gray-800 border border-gray-700 rounded-xl overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center h-48">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
          </div>
        ) : tenants.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 text-gray-400">
            <Building2 className="w-12 h-12 mb-3 opacity-50" />
            <p>Nenhuma empresa encontrada</p>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-700">
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Empresa
                </th>
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Plano
                </th>
                <th className="text-center px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Usuários
                </th>
                <th className="text-center px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Conversas
                </th>
                <th className="text-center px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Status
                </th>
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Criada em
                </th>
                <th className="text-right px-6 py-4 text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Ações
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {tenants.map((tenant) => (
                <tr key={tenant.id} className="hover:bg-gray-750 transition-colors">
                  <td className="px-6 py-4">
                    <div>
                      <p className="text-white font-medium">{tenant.name}</p>
                      <p className="text-gray-500 text-sm">{tenant.slug}</p>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span className="px-2.5 py-1 rounded-full text-xs font-medium bg-indigo-500/20 text-indigo-300 capitalize">
                      {tenant.plan}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-center">
                    <div className="flex items-center justify-center gap-1 text-gray-300">
                      <Users className="w-3.5 h-3.5" />
                      <span className="text-sm">{tenant.user_count}/{tenant.max_users}</span>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-center">
                    <div className="flex items-center justify-center gap-1 text-gray-300">
                      <MessageSquare className="w-3.5 h-3.5" />
                      <span className="text-sm">{tenant.conversation_count}</span>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-center">
                    <span
                      className={`px-2.5 py-1 rounded-full text-xs font-medium ${
                        tenant.is_active
                          ? 'bg-green-500/20 text-green-300'
                          : 'bg-red-500/20 text-red-300'
                      }`}
                    >
                      {tenant.is_active ? 'Ativa' : 'Inativa'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-gray-400 text-sm">
                    {formatDate(tenant.created_at)}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => router.push(`/admin/tenants/${tenant.id}`)}
                        className="p-2 text-gray-400 hover:text-blue-400 hover:bg-gray-700 rounded-lg transition-colors"
                        title="Gerenciar usuários"
                      >
                        <Users className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => openEditModal(tenant)}
                        className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                        title="Editar"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      {tenant.is_active && (
                        <button
                          onClick={() => handleDeactivate(tenant)}
                          className="p-2 text-gray-400 hover:text-red-400 hover:bg-gray-700 rounded-lg transition-colors"
                          title="Desativar"
                        >
                          <Power className="w-4 h-4" />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-800 border border-gray-700 rounded-xl w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between p-6 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">Nova Empresa</h2>
              <button
                onClick={() => setShowCreateModal(false)}
                className="p-2 text-gray-400 hover:text-white rounded-lg"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5">
                  Nome da Empresa *
                </label>
                <input
                  type="text"
                  value={createForm.company_name}
                  onChange={(e) => setCreateForm({ ...createForm, company_name: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  placeholder="Nome da empresa"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5">
                  Slug (opcional)
                </label>
                <input
                  type="text"
                  value={createForm.slug}
                  onChange={(e) => setCreateForm({ ...createForm, slug: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  placeholder="minha-empresa"
                />
              </div>

              <div className="border-t border-gray-700 pt-4">
                <p className="text-sm font-medium text-gray-300 mb-3">Usuário Admin</p>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1.5">
                      Nome do Admin
                    </label>
                    <input
                      type="text"
                      value={createForm.admin_name}
                      onChange={(e) => setCreateForm({ ...createForm, admin_name: e.target.value })}
                      className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                      placeholder="Nome do administrador"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1.5">
                      Email do Admin *
                    </label>
                    <input
                      type="email"
                      value={createForm.admin_email}
                      onChange={(e) => setCreateForm({ ...createForm, admin_email: e.target.value })}
                      className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                      placeholder="admin@empresa.com"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1.5">
                      Senha do Admin *
                    </label>
                    <input
                      type="password"
                      value={createForm.admin_password}
                      onChange={(e) => setCreateForm({ ...createForm, admin_password: e.target.value })}
                      className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                      placeholder="Mínimo 8 caracteres"
                      minLength={8}
                      required
                    />
                  </div>
                </div>
              </div>

              <div className="border-t border-gray-700 pt-4">
                <p className="text-sm font-medium text-gray-300 mb-3">Configurações</p>
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1.5">
                      Plano
                    </label>
                    <select
                      value={createForm.plan}
                      onChange={(e) => setCreateForm({ ...createForm, plan: e.target.value })}
                      className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                    >
                      <option value="basic">Basic</option>
                      <option value="professional">Professional</option>
                      <option value="enterprise">Enterprise</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1.5">
                      Máx. Usuários
                    </label>
                    <input
                      type="number"
                      value={createForm.max_users}
                      onChange={(e) => setCreateForm({ ...createForm, max_users: parseInt(e.target.value) || 5 })}
                      className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                      min={1}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1.5">
                      Máx. Canais
                    </label>
                    <input
                      type="number"
                      value={createForm.max_channels}
                      onChange={(e) => setCreateForm({ ...createForm, max_channels: parseInt(e.target.value) || 2 })}
                      className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                      min={1}
                    />
                  </div>
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-4 border-t border-gray-700">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2.5 text-gray-400 hover:text-white border border-gray-600 rounded-lg transition-colors"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                >
                  {submitting ? 'Criando...' : 'Criar Empresa'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {showEditModal && editingTenant && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-800 border border-gray-700 rounded-xl w-full max-w-lg mx-4">
            <div className="flex items-center justify-between p-6 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">
                Editar: {editingTenant.name}
              </h2>
              <button
                onClick={() => setShowEditModal(false)}
                className="p-2 text-gray-400 hover:text-white rounded-lg"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleEdit} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5">
                  Nome da Empresa
                </label>
                <input
                  type="text"
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-1.5">
                    Plano
                  </label>
                  <select
                    value={editForm.plan}
                    onChange={(e) => setEditForm({ ...editForm, plan: e.target.value })}
                    className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                  >
                    <option value="basic">Basic</option>
                    <option value="professional">Professional</option>
                    <option value="enterprise">Enterprise</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-1.5">
                    Máx. Usuários
                  </label>
                  <input
                    type="number"
                    value={editForm.max_users}
                    onChange={(e) => setEditForm({ ...editForm, max_users: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                    min={1}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-1.5">
                    Máx. Canais
                  </label>
                  <input
                    type="number"
                    value={editForm.max_channels}
                    onChange={(e) => setEditForm({ ...editForm, max_channels: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                    min={1}
                  />
                </div>
              </div>

              <div>
                <label className="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={editForm.is_active}
                    onChange={(e) => setEditForm({ ...editForm, is_active: e.target.checked })}
                    className="w-4 h-4 rounded border-gray-600 bg-gray-900 text-indigo-600 focus:ring-indigo-500"
                  />
                  <span className="text-sm text-gray-300">Empresa ativa</span>
                </label>
              </div>

              <div className="flex justify-end gap-3 pt-4 border-t border-gray-700">
                <button
                  type="button"
                  onClick={() => setShowEditModal(false)}
                  className="px-4 py-2.5 text-gray-400 hover:text-white border border-gray-600 rounded-lg transition-colors"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                >
                  {submitting ? 'Salvando...' : 'Salvar Alterações'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
