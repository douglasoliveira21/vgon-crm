'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Users, Edit2, Trash2, UserPlus, Shield } from 'lucide-react'

interface Team {
  id: string
  name: string
  description?: string
  distribution_rule: string
  is_active: boolean
  member_count: number
}

export default function TeamsPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)

  useEffect(() => {
    fetchTeams()
  }, [])

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

  const deleteTeam = async (id: string) => {
    if (!confirm('Remover este time?')) return
    try {
      await api.delete(`/teams/${id}`)
      setTeams((prev) => prev.filter((t) => t.id !== id))
      toast.success('Time removido')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  const distributionLabels: Record<string, string> = {
    'round-robin': 'Round Robin',
    'manual': 'Manual',
    'least-busy': 'Menor fila',
    'first-available': 'Primeiro disponível',
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Times</h1>
          <p className="text-gray-500 mt-1">Organize seus atendentes em equipes</p>
        </div>
        <button onClick={() => setShowForm(true)} className="btn-primary">
          <Plus size={18} />
          Novo time
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {teams.map((team) => (
          <div key={team.id} className="card p-5">
            <div className="flex items-start justify-between mb-3">
              <div className="w-10 h-10 bg-primary-100 rounded-xl flex items-center justify-center">
                <Users size={20} className="text-primary-600" />
              </div>
              <div className="flex gap-1">
                <button className="p-1.5 text-gray-400 hover:text-primary-600 rounded">
                  <Edit2 size={14} />
                </button>
                <button
                  onClick={() => deleteTeam(team.id)}
                  className="p-1.5 text-gray-400 hover:text-red-600 rounded"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>

            <h3 className="font-semibold text-gray-900">{team.name}</h3>
            {team.description && (
              <p className="text-sm text-gray-500 mt-1">{team.description}</p>
            )}

            <div className="mt-4 flex items-center justify-between">
              <span className="text-xs text-gray-400 flex items-center gap-1">
                <UserPlus size={12} /> {team.member_count} membros
              </span>
              <span className="badge badge-blue">
                {distributionLabels[team.distribution_rule] || team.distribution_rule}
              </span>
            </div>
          </div>
        ))}
      </div>

      {teams.length === 0 && !loading && (
        <div className="card p-12 text-center">
          <Users size={40} className="text-gray-300 mx-auto mb-4" />
          <p className="text-gray-500 mb-4">Nenhum time criado ainda</p>
          <button onClick={() => setShowForm(true)} className="btn-primary inline-flex">
            <Plus size={18} /> Criar primeiro time
          </button>
        </div>
      )}
    </div>
  )
}
