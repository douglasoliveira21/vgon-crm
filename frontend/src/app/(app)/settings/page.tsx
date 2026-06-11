'use client'

import { useState, useEffect } from 'react'
import { useAuthStore } from '@/store/auth'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { User, Building, Shield, Bell, Palette, MessageSquare, Tag } from 'lucide-react'

export default function SettingsPage() {
  const { user } = useAuthStore()

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-8">Configurações</h1>

      <div className="space-y-6">
        {/* Profile */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <User size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Meu perfil</h2>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nome</label>
              <input type="text" defaultValue={user?.name} className="input" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">E-mail</label>
              <input type="email" defaultValue={user?.email} className="input" disabled />
            </div>
          </div>

          <button className="btn-primary mt-4">Salvar alterações</button>
        </div>

        {/* Company */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Building size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Empresa</h2>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nome da empresa</label>
              <input type="text" className="input" placeholder="Minha Empresa LTDA" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">CNPJ</label>
              <input type="text" className="input" placeholder="00.000.000/0001-00" />
            </div>
          </div>
        </div>

        {/* Security */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Shield size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Segurança</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Senha atual</label>
              <input type="password" className="input" placeholder="••••••••" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nova senha</label>
                <input type="password" className="input" placeholder="••••••••" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Confirmar nova senha</label>
                <input type="password" className="input" placeholder="••••••••" />
              </div>
            </div>
          </div>

          <button className="btn-primary mt-4">Alterar senha</button>
        </div>

        {/* Notifications */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Bell size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Notificações</h2>
          </div>

          <div className="space-y-3">
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Nova conversa atribuída</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Nova mensagem recebida</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Menção com @nome</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Recados internos</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
          </div>
        </div>

        {/* Channels / WhatsApp */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <MessageSquare size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">WhatsApp / Canais</h2>
          </div>

          <div className="space-y-3">
            <label className="flex items-center justify-between">
              <div>
                <span className="text-sm text-gray-700">Receber mensagens de grupos</span>
                <p className="text-xs text-gray-400">Se desativado, mensagens de grupos do WhatsApp serão ignoradas</p>
              </div>
              <input
                type="checkbox"
                defaultChecked={false}
                onChange={async (e) => {
                  try {
                    await api.put('/settings', { receive_group_messages: e.target.checked })
                    toast.success(e.target.checked ? 'Mensagens de grupos ativadas' : 'Mensagens de grupos desativadas')
                  } catch {}
                }}
                className="rounded border-gray-300"
              />
            </label>
            <label className="flex items-center justify-between">
              <div>
                <span className="text-sm text-gray-700">Ignorar mensagens de status/stories</span>
                <p className="text-xs text-gray-400">Não receber atualizações de status do WhatsApp</p>
              </div>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <div>
                <span className="text-sm text-gray-700">Enviar confirmação de leitura</span>
                <p className="text-xs text-gray-400">Marcar mensagens como lidas ao visualizar no CRM</p>
              </div>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
          </div>
        </div>

        {/* Tags */}
        <TagsManager />
      </div>
    </div>
  )
}

function TagsManager() {
  const [tags, setTags] = useState<Array<{id: string; name: string; color: string}>>([])
  const [newTag, setNewTag] = useState('')
  const [newColor, setNewColor] = useState('#3B82F6')

  useEffect(() => {
    fetchTags()
  }, [])

  const fetchTags = async () => {
    try {
      const res = await api.get('/tags')
      setTags(res.data.tags || [])
    } catch {}
  }

  const createTag = async () => {
    if (!newTag.trim()) return
    try {
      await api.post('/tags', { name: newTag, color: newColor })
      toast.success('Tag criada')
      setNewTag('')
      fetchTags()
    } catch {
      toast.error('Erro ao criar tag')
    }
  }

  const deleteTag = async (id: string) => {
    if (!confirm('Remover esta tag?')) return
    try {
      await api.delete(`/tags/${id}`)
      setTags(tags.filter(t => t.id !== id))
      toast.success('Tag removida')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  return (
    <div className="card p-6">
      <div className="flex items-center gap-3 mb-4">
        <Tag size={20} className="text-gray-400" />
        <h2 className="text-lg font-semibold text-gray-900">Tags / Etiquetas</h2>
      </div>

      {/* Create tag */}
      <div className="flex items-center gap-2 mb-4">
        <input
          type="color"
          value={newColor}
          onChange={(e) => setNewColor(e.target.value)}
          className="w-9 h-9 rounded-lg border border-gray-200 cursor-pointer"
        />
        <input
          type="text"
          value={newTag}
          onChange={(e) => setNewTag(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') createTag() }}
          className="input flex-1"
          placeholder="Nome da nova tag..."
        />
        <button onClick={createTag} disabled={!newTag.trim()} className="btn-primary text-sm py-2.5">
          Criar
        </button>
      </div>

      {/* Tags list */}
      <div className="flex flex-wrap gap-2">
        {tags.map((tag) => (
          <div
            key={tag.id}
            className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium border"
            style={{ backgroundColor: tag.color + '20', borderColor: tag.color, color: tag.color }}
          >
            <span className="w-2 h-2 rounded-full" style={{ backgroundColor: tag.color }} />
            {tag.name}
            <button
              onClick={() => deleteTag(tag.id)}
              className="ml-1 opacity-50 hover:opacity-100"
            >
              ×
            </button>
          </div>
        ))}
        {tags.length === 0 && (
          <p className="text-sm text-gray-400">Nenhuma tag criada. Crie tags para organizar seus contatos.</p>
        )}
      </div>
    </div>
  )
}
