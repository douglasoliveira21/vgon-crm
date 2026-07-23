'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  ArrowLeft,
  Users,
  Plus,
  Edit,
  Key,
  Trash2,
  X,
  Building2,
  MessageSquare,
  Radio,
} from 'lucide-react'

interface TenantUser {
  id: string
  name: string
  email: string
  is_active: boolean
  is_online: boolean
  role_name: string
  role_slug: string
  is_super_admin: boolean
  created_at: string
}

interface TenantInfo {
  id: string
  name: string
  slug: string
  plan: string
  max_users: number
  max_channels: number
  is_active: boolean
}

interface TenantStats {
  users: number
  conversations: number
  messages: number
  channels: number
}

export default function TenantUsersPage() {
  const params = useParams()
  const router = useRouter()
  const tenantId = params.id as string

  const [tenant, setTenant] = useState<TenantInfo | null>(null)
  const [stats, setStats] = useState<TenantStats | null>(null)
  const [users, setUsers] = useState<TenantUser[]>([])
  const [loading, setLoading] = useState(true)

  // Modals
  const [showCreateUser, setShowCreateUser] = useState(false)
  const [showResetPassword, setShowResetPassword] = useState(false)
  const [showEditUser, setShowEditUser] = useState(false)
  const [selectedUser, setSelectedUser] = useState<TenantUser | null>(null)

  // Forms
  const [createForm, setCreateForm] = useState({ name: '', email: '', password: '', role: 'agent' })
  const [newPassword, setNewPassword] = useState('')
  const [editForm, setEditForm] = useState({ name: '', email: '', role: 'agent' })

  useEffect(() => {
    fetchTenantDetails()
    fetchUsers()
  }, [tenantId])

  const fetchTenantDetails = async () => {
    try {
      const res = await api.get(`/admin/tenants/${tenantId}`)
      setTenant(res.data.tenant)
      setStats(res.data.stats)
    } catch {
      toast.error('Erro ao carregar dados da empresa')
    }
  }

  const fetchUsers = async () => {
    try {
      const res = await api.get(`/admin/tenants/${tenantId}/users`)
      setUsers(res.data.users || [])
    } catch {
      toast.error('Erro ao carregar usuários')
    } finally {
      setLoading(false)
    }
  }

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createForm.name || !createForm.email || !createForm.password) {
      toast.error('Preencha todos os campos')
      return
    }
    try {
      await api.post(`/admin/tenants/${tenantId}/users`, createForm)
      toast.success('Usuário criado com sucesso!')
      setShowCreateUser(false)
      setCreateForm({ name: '', email: '', password: '', role: 'agent' })
      fetchUsers()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao criar usuário')
    }
  }

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedUser || !newPassword) return
    try {
      await api.put(`/admin/users/${selectedUser.id}/password`, { new_password: newPassword })
      toast.success('Senha alterada com sucesso!')
      setShowResetPassword(false)
      setNewPassword('')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao alterar senha')
    }
  }

  const handleEditUser = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedUser) return
    try {
      await api.put(`/admin/users/${selectedUser.id}`, editForm)
      toast.success('Usuário atualizado!')
      setShowEditUser(false)
      fetchUsers()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao atualizar')
    }
  }

  const handleDeleteUser = async (user: TenantUser) => {
    if (!confirm(`Excluir permanentemente o usuário "${user.name}" e seus históricos vinculados? Esta ação não pode ser desfeita.`)) return
    try {
      await api.delete(`/admin/users/${user.id}`)
      toast.success('Usuário excluído permanentemente')
      fetchUsers()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao excluir usuário')
    }
  }

  const openResetPassword = (user: TenantUser) => {
    setSelectedUser(user)
    setNewPassword('')
    setShowResetPassword(true)
  }

  const openEditUser = (user: TenantUser) => {
    setSelectedUser(user)
    setEditForm({
      name: user.name,
      email: user.email,
      role: user.is_super_admin ? 'super-admin' : (user.role_slug || 'agent'),
    })
    setShowEditUser(true)
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <button
          onClick={() => router.push('/admin/tenants')}
          className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div>
          <h1 className="text-2xl font-bold text-white">{tenant?.name || 'Carregando...'}</h1>
          <p className="text-gray-400 text-sm">{tenant?.slug} • Plano: {tenant?.plan}</p>
        </div>
      </div>

      {/* Stats */}
      {stats && (
        <div className="grid grid-cols-4 gap-4 mb-8">
          <div className="bg-gray-800 border border-gray-700 rounded-xl p-4">
            <div className="flex items-center gap-2 text-gray-400 mb-1">
              <Users className="w-4 h-4" /> <span className="text-xs">Usuários</span>
            </div>
            <p className="text-xl font-bold text-white">{stats.users}/{tenant?.max_users}</p>
          </div>
          <div className="bg-gray-800 border border-gray-700 rounded-xl p-4">
            <div className="flex items-center gap-2 text-gray-400 mb-1">
              <MessageSquare className="w-4 h-4" /> <span className="text-xs">Conversas</span>
            </div>
            <p className="text-xl font-bold text-white">{stats.conversations}</p>
          </div>
          <div className="bg-gray-800 border border-gray-700 rounded-xl p-4">
            <div className="flex items-center gap-2 text-gray-400 mb-1">
              <Building2 className="w-4 h-4" /> <span className="text-xs">Mensagens</span>
            </div>
            <p className="text-xl font-bold text-white">{stats.messages}</p>
          </div>
          <div className="bg-gray-800 border border-gray-700 rounded-xl p-4">
            <div className="flex items-center gap-2 text-gray-400 mb-1">
              <Radio className="w-4 h-4" /> <span className="text-xs">Canais</span>
            </div>
            <p className="text-xl font-bold text-white">{stats.channels}/{tenant?.max_channels}</p>
          </div>
        </div>
      )}

      {/* Users section */}
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <Users className="w-5 h-5" /> Usuários da Empresa
        </h2>
        <button
          onClick={() => setShowCreateUser(true)}
          className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-sm font-medium"
        >
          <Plus className="w-4 h-4" /> Adicionar Usuário
        </button>
      </div>

      {/* Users Table */}
      <div className="bg-gray-800 border border-gray-700 rounded-xl overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-500"></div>
          </div>
        ) : users.length === 0 ? (
          <div className="p-8 text-center text-gray-400">Nenhum usuário encontrado</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-700">
                <th className="text-left px-6 py-3 text-xs font-medium text-gray-400 uppercase">Nome</th>
                <th className="text-left px-6 py-3 text-xs font-medium text-gray-400 uppercase">Email</th>
                <th className="text-center px-6 py-3 text-xs font-medium text-gray-400 uppercase">Cargo</th>
                <th className="text-center px-6 py-3 text-xs font-medium text-gray-400 uppercase">Status</th>
                <th className="text-right px-6 py-3 text-xs font-medium text-gray-400 uppercase">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-gray-750">
                  <td className="px-6 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 bg-indigo-600 rounded-full flex items-center justify-center text-white text-xs font-medium">
                        {user.name.charAt(0).toUpperCase()}
                      </div>
                      <span className="text-white text-sm">{user.name}</span>
                    </div>
                  </td>
                  <td className="px-6 py-3 text-gray-300 text-sm">{user.email}</td>
                  <td className="px-6 py-3 text-center">
                    <span className="px-2 py-0.5 rounded-full text-xs bg-gray-700 text-gray-300">
                      {user.is_super_admin ? 'Super administrador' : (user.role_name || 'Agente')}
                    </span>
                  </td>
                  <td className="px-6 py-3 text-center">
                    <span className={`px-2 py-0.5 rounded-full text-xs ${
                      user.is_active ? 'bg-green-500/20 text-green-300' : 'bg-red-500/20 text-red-300'
                    }`}>
                      {user.is_active ? (user.is_online ? '🟢 Online' : 'Ativo') : 'Inativo'}
                    </span>
                  </td>
                  <td className="px-6 py-3">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => openEditUser(user)}
                        className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded"
                        title="Editar"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => openResetPassword(user)}
                        className="p-1.5 text-gray-400 hover:text-yellow-400 hover:bg-gray-700 rounded"
                        title="Alterar senha"
                      >
                        <Key className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDeleteUser(user)}
                        className="p-1.5 text-gray-400 hover:text-red-400 hover:bg-gray-700 rounded"
                        title="Excluir permanentemente"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Create User Modal */}
      {showCreateUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-800 border border-gray-700 rounded-xl w-full max-w-md mx-4">
            <div className="flex items-center justify-between p-5 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">Adicionar Usuário</h2>
              <button onClick={() => setShowCreateUser(false)} className="text-gray-400 hover:text-white"><X className="w-5 h-5" /></button>
            </div>
            <form onSubmit={handleCreateUser} className="p-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Nome *</label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  placeholder="Nome completo"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Email *</label>
                <input
                  type="email"
                  value={createForm.email}
                  onChange={(e) => setCreateForm({ ...createForm, email: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  placeholder="usuario@empresa.com"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Senha *</label>
                <input
                  type="password"
                  value={createForm.password}
                  onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  placeholder="Mínimo 6 caracteres"
                  minLength={6}
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Cargo</label>
                <select
                  value={createForm.role}
                  onChange={(e) => setCreateForm({ ...createForm, role: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                >
                  <option value="agent">Agente</option>
                  <option value="supervisor">Supervisor</option>
                  <option value="admin">Administrador</option>
                  <option value="super-admin">Super administrador</option>
                </select>
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => setShowCreateUser(false)} className="flex-1 px-4 py-2.5 border border-gray-600 text-gray-400 rounded-lg hover:text-white">
                  Cancelar
                </button>
                <button type="submit" className="flex-1 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium">
                  Criar Usuário
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Reset Password Modal */}
      {showResetPassword && selectedUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-800 border border-gray-700 rounded-xl w-full max-w-sm mx-4">
            <div className="flex items-center justify-between p-5 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">Alterar Senha</h2>
              <button onClick={() => setShowResetPassword(false)} className="text-gray-400 hover:text-white"><X className="w-5 h-5" /></button>
            </div>
            <form onSubmit={handleResetPassword} className="p-5 space-y-4">
              <p className="text-sm text-gray-400">Alterando senha de: <strong className="text-white">{selectedUser.name}</strong></p>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Nova Senha *</label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  placeholder="Nova senha (mín. 6 caracteres)"
                  minLength={6}
                  required
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => setShowResetPassword(false)} className="flex-1 px-4 py-2.5 border border-gray-600 text-gray-400 rounded-lg hover:text-white">
                  Cancelar
                </button>
                <button type="submit" className="flex-1 px-4 py-2.5 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg font-medium">
                  Alterar Senha
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit User Modal */}
      {showEditUser && selectedUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-800 border border-gray-700 rounded-xl w-full max-w-sm mx-4">
            <div className="flex items-center justify-between p-5 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">Editar Usuário</h2>
              <button onClick={() => setShowEditUser(false)} className="text-gray-400 hover:text-white"><X className="w-5 h-5" /></button>
            </div>
            <form onSubmit={handleEditUser} className="p-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Nome</label>
                <input
                  type="text"
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Email</label>
                <input
                  type="email"
                  value={editForm.email}
                  onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">Cargo</label>
                <select
                  value={editForm.role}
                  onChange={(e) => setEditForm({ ...editForm, role: e.target.value })}
                  className="w-full px-4 py-2.5 bg-gray-900 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-indigo-500"
                >
                  <option value="agent">Agente</option>
                  <option value="supervisor">Supervisor</option>
                  <option value="admin">Administrador</option>
                  <option value="super-admin">Super administrador</option>
                </select>
                <p className="mt-1 text-xs text-gray-400">
                  Somente o Super administrador acessa o painel global.
                </p>
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => setShowEditUser(false)} className="flex-1 px-4 py-2.5 border border-gray-600 text-gray-400 rounded-lg hover:text-white">
                  Cancelar
                </button>
                <button type="submit" className="flex-1 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium">
                  Salvar
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
