'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Users, Edit2, Trash2, UserPlus, UserMinus, X, Shield, Settings } from 'lucide-react'
import { useAuthStore } from '@/store/auth'

interface Team {
  id: string
  name: string
  description?: string
  distribution_rule: string
  is_active: boolean
  member_count: number
}

interface UserItem {
  id: string
  name: string
  email: string
  role_name?: string
  is_online: boolean
  avatar_url?: string
}

interface TeamMember {
  id: string
  user_id: string
  is_supervisor: boolean
  user_name?: string
  user_email?: string
}

export default function TeamsPage() {
	const { user } = useAuthStore()
	const canManage = !!user && user.role_slug !== 'agent'
	const isSupervisor = user?.role_slug === 'supervisor'
	const canConfigureTeams = canManage && !isSupervisor
	const canAddMembers = canManage
  const [teams, setTeams] = useState<Team[]>([])
  const [users, setUsers] = useState<UserItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [editingTeam, setEditingTeam] = useState<Team | null>(null)
  const [showMembers, setShowMembers] = useState<string | null>(null)
  const [members, setMembers] = useState<TeamMember[]>([])

  useEffect(() => {
    fetchTeams()
	if (canManage) fetchUsers()
  }, [canManage])

  const fetchTeams = async () => {
    try {
      const response = await api.get('/teams')
      setTeams(response.data.teams || [])
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchUsers = async () => {
    try {
      const response = await api.get('/users')
      setUsers(response.data.users || [])
    } catch {}
  }

  const createTeam = async (data: { name: string; description: string; distribution_rule: string }) => {
    try {
      await api.post('/teams', data)
      toast.success('Time criado')
      setShowForm(false)
      fetchTeams()
    } catch {
      toast.error('Erro ao criar time')
    }
  }

  const updateTeam = async (id: string, data: { name: string; description: string; distribution_rule: string }) => {
    try {
      await api.put(`/teams/${id}`, data)
      toast.success('Time atualizado')
      setShowForm(false)
      setEditingTeam(null)
      fetchTeams()
    } catch {
      toast.error('Erro ao atualizar')
    }
  }

  const deleteTeam = async (id: string) => {
    if (!confirm('Remover este time? Todas as conversas serão desvinculadas.')) return
    try {
      await api.delete(`/teams/${id}`)
      setTeams((prev) => prev.filter((t) => t.id !== id))
      toast.success('Time removido')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  const addMember = async (teamId: string, userId: string, isSupervisor: boolean = false) => {
    try {
      await api.post(`/teams/${teamId}/members`, { user_id: userId, is_supervisor: isSupervisor })
      toast.success('Membro adicionado')
      fetchTeams()
    } catch {
      toast.error('Erro ao adicionar membro')
    }
  }

  const removeMember = async (teamId: string, userId: string) => {
    try {
      await api.delete(`/teams/${teamId}/members/${userId}`)
      toast.success('Membro removido')
      fetchTeams()
    } catch {
      toast.error('Erro ao remover membro')
    }
  }

  const distributionLabels: Record<string, string> = {
    'round-robin': 'Round Robin',
    'manual': 'Manual',
    'least-busy': 'Menor fila',
    'first-available': 'Primeiro disponível',
    'keyword': 'Por palavra-chave',
    'schedule': 'Por horário',
  }

  const distributionDescriptions: Record<string, string> = {
    'round-robin': 'Distribui conversas igualmente entre os membros',
    'manual': 'Atendente escolhe manualmente qual conversa atender',
    'least-busy': 'Envia para o atendente com menos conversas abertas',
    'first-available': 'Envia para o primeiro atendente online disponível',
    'keyword': 'Direciona com base em palavras-chave da mensagem',
    'schedule': 'Distribui conforme horário de trabalho dos atendentes',
  }

  return (
    <div className="mx-auto max-w-5xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Times</h1>
          <p className="text-gray-500 mt-1">Organize seus atendentes em equipes com regras de distribuição</p>
        </div>
		{canConfigureTeams && <button onClick={() => { setEditingTeam(null); setShowForm(true) }} className="btn-primary">
          <Plus size={18} />
          Novo time
		</button>}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {teams.map((team) => (
          <div key={team.id} className="card p-5 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between mb-3">
              <div className="w-10 h-10 bg-primary-100 rounded-xl flex items-center justify-center">
                <Users size={20} className="text-primary-600" />
              </div>
			  {canManage && <div className="flex gap-1">
				{canConfigureTeams && <>
                <button
                  onClick={() => { setEditingTeam(team); setShowForm(true) }}
                  className="p-1.5 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded"
                  title="Editar"
                >
                  <Edit2 size={14} />
                </button>
				</>}
                <button
                  onClick={() => setShowMembers(team.id)}
                  className="p-1.5 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded"
				  title={isSupervisor ? 'Adicionar membro' : 'Gerenciar membros'}
                >
                  <UserPlus size={14} />
                </button>
				{canConfigureTeams && <>
                <button
                  onClick={() => deleteTeam(team.id)}
                  className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                  title="Remover"
                >
                  <Trash2 size={14} />
                </button>
				</>}
			  </div>}
            </div>

            <h3 className="font-semibold text-gray-900">{team.name}</h3>
            {team.description && (
              <p className="text-sm text-gray-500 mt-1 line-clamp-2">{team.description}</p>
            )}

            <div className="mt-4 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs text-gray-400 flex items-center gap-1">
                  <Users size={12} /> {team.member_count} membros
                </span>
                <span className="badge badge-blue">
                  {distributionLabels[team.distribution_rule] || team.distribution_rule}
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {teams.length === 0 && !loading && (
        <div className="card p-12 text-center">
          <Users size={40} className="text-gray-300 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">Nenhum time criado</h3>
		  <p className="text-gray-500 mb-6 max-w-sm mx-auto">
			{canConfigureTeams ? 'Crie times para organizar seu atendimento. Cada time pode ter regras de distribuição diferentes.' : isSupervisor ? 'Você não supervisiona nenhum time.' : 'Você ainda não foi adicionado a nenhum time.'}
		  </p>
		  {canConfigureTeams && <button onClick={() => setShowForm(true)} className="btn-primary inline-flex">
            <Plus size={18} /> Criar primeiro time
		  </button>}
        </div>
      )}

      {/* Create/Edit Team Modal */}
	  {canConfigureTeams && showForm && (
        <TeamFormModal
          team={editingTeam}
          onClose={() => { setShowForm(false); setEditingTeam(null) }}
          onSave={(data) => {
            if (editingTeam) {
              updateTeam(editingTeam.id, data)
            } else {
              createTeam(data)
            }
          }}
          distributionLabels={distributionLabels}
          distributionDescriptions={distributionDescriptions}
        />
      )}

      {/* Members Modal */}
	  {canAddMembers && showMembers && (
        <MembersModal
          teamId={showMembers}
          teamName={teams.find(t => t.id === showMembers)?.name || ''}
          users={users}
          onClose={() => setShowMembers(null)}
          onAddMember={addMember}
          onRemoveMember={removeMember}
		  canSetSupervisor={!isSupervisor}
        />
      )}
    </div>
  )
}

// Team Form Modal
function TeamFormModal({
  team,
  onClose,
  onSave,
  distributionLabels,
  distributionDescriptions,
}: {
  team: Team | null
  onClose: () => void
  onSave: (data: { name: string; description: string; distribution_rule: string }) => void
  distributionLabels: Record<string, string>
  distributionDescriptions: Record<string, string>
}) {
  const [name, setName] = useState(team?.name || '')
  const [description, setDescription] = useState(team?.description || '')
  const [distributionRule, setDistributionRule] = useState(team?.distribution_rule || 'round-robin')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-lg">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold text-gray-900">
            {team ? 'Editar time' : 'Novo time'}
          </h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X size={20} />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nome do time</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="input"
              placeholder="Ex: Suporte, Vendas, Financeiro..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Descrição</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="input resize-none"
              rows={2}
              placeholder="Descrição do time (opcional)"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Regra de distribuição</label>
            <div className="space-y-2">
              {Object.entries(distributionLabels).map(([key, label]) => (
                <label
                  key={key}
                  className={`flex items-start gap-3 p-3 border rounded-lg cursor-pointer transition-colors ${
                    distributionRule === key ? 'border-primary-500 bg-primary-50' : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <input
                    type="radio"
                    name="distribution"
                    value={key}
                    checked={distributionRule === key}
                    onChange={() => setDistributionRule(key)}
                    className="mt-0.5"
                  />
                  <div>
                    <p className="text-sm font-medium text-gray-900">{label}</p>
                    <p className="text-xs text-gray-500">{distributionDescriptions[key]}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button onClick={onClose} className="btn-secondary flex-1">Cancelar</button>
          <button
            onClick={() => onSave({ name, description, distribution_rule: distributionRule })}
            disabled={!name.trim()}
            className="btn-primary flex-1"
          >
            {team ? 'Salvar' : 'Criar time'}
          </button>
        </div>
      </div>
    </div>
  )
}

// Members Modal
function MembersModal({
  teamId,
  teamName,
  users,
  onClose,
  onAddMember,
  onRemoveMember,
	canSetSupervisor,
}: {
  teamId: string
  teamName: string
  users: UserItem[]
  onClose: () => void
  onAddMember: (teamId: string, userId: string, isSupervisor: boolean) => void
  onRemoveMember: (teamId: string, userId: string) => void
	canSetSupervisor: boolean
}) {
  const [teamMembers, setTeamMembers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchMembers()
  }, [])

  const fetchMembers = async () => {
    try {
      // For now, use users list and mark those in team
      // TODO: Add dedicated endpoint for team members
      setLoading(false)
    } catch {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl p-6 w-full max-w-md max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Membros do time</h3>
            <p className="text-sm text-gray-500">{teamName}</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X size={20} />
          </button>
        </div>

		<p className="text-xs text-gray-400 mb-4">{canSetSupervisor ? 'Adicione membros ou defina outro supervisor.' : 'Adicione atendentes ao seu time.'}</p>

        <div className="space-y-2">
          {users.map((user) => (
            <div key={user.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-primary-100 rounded-full flex items-center justify-center">
                  <span className="text-primary-700 text-xs font-medium">{user.name.charAt(0)}</span>
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">{user.name}</p>
                  <p className="text-xs text-gray-400">{user.role_name || user.email}</p>
                </div>
                {user.is_online && <div className="w-2 h-2 bg-green-500 rounded-full" />}
              </div>
              <div className="flex gap-1">
				<button
                  onClick={() => onAddMember(teamId, user.id, false)}
                  className="px-2 py-1 text-xs bg-primary-100 text-primary-700 rounded hover:bg-primary-200"
                  title="Adicionar como membro"
                >
                  + Membro
				</button>
				{canSetSupervisor && <button
                  onClick={() => onAddMember(teamId, user.id, true)}
                  className="px-2 py-1 text-xs bg-yellow-100 text-yellow-700 rounded hover:bg-yellow-200"
                  title="Adicionar como supervisor"
                >
                  ⭐ Supervisor
				</button>}
              </div>
            </div>
          ))}
        </div>

        <div className="mt-4">
          <button onClick={onClose} className="btn-secondary w-full">Fechar</button>
        </div>
      </div>
    </div>
  )
}
