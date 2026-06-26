'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Pencil, Trash2, Search, Zap, X } from 'lucide-react'

interface QuickReply {
  id: string
  shortcut: string
  title: string | null
  content: string
  category: string | null
  created_at: string
}

export default function QuickRepliesPage() {
  const [replies, setReplies] = useState<QuickReply[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingReply, setEditingReply] = useState<QuickReply | null>(null)
  const [form, setForm] = useState({ shortcut: '', title: '', content: '', category: '' })

  useEffect(() => {
    fetchReplies()
  }, [])

  const fetchReplies = async () => {
    try {
      const { data } = await api.get('/quick-replies')
      setReplies(data.quick_replies || [])
    } catch {
      toast.error('Erro ao carregar respostas rápidas')
    } finally {
      setLoading(false)
    }
  }

  const openCreate = () => {
    setEditingReply(null)
    setForm({ shortcut: '', title: '', content: '', category: '' })
    setShowModal(true)
  }

  const openEdit = (reply: QuickReply) => {
    setEditingReply(reply)
    setForm({
      shortcut: reply.shortcut,
      title: reply.title || '',
      content: reply.content,
      category: reply.category || '',
    })
    setShowModal(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!form.shortcut.trim() || !form.content.trim()) {
      toast.error('Atalho e conteúdo são obrigatórios')
      return
    }

    try {
      if (editingReply) {
        await api.put(`/quick-replies/${editingReply.id}`, form)
        toast.success('Resposta rápida atualizada')
      } else {
        await api.post('/quick-replies', form)
        toast.success('Resposta rápida criada')
      }
      setShowModal(false)
      fetchReplies()
    } catch {
      toast.error('Erro ao salvar resposta rápida')
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Tem certeza que deseja excluir esta resposta rápida?')) return

    try {
      await api.delete(`/quick-replies/${id}`)
      toast.success('Resposta rápida excluída')
      fetchReplies()
    } catch {
      toast.error('Erro ao excluir')
    }
  }

  const filteredReplies = replies.filter(
    (r) =>
      r.shortcut.toLowerCase().includes(search.toLowerCase()) ||
      r.content.toLowerCase().includes(search.toLowerCase()) ||
      (r.title && r.title.toLowerCase().includes(search.toLowerCase())) ||
      (r.category && r.category.toLowerCase().includes(search.toLowerCase()))
  )

  const categories = [...new Set(replies.map((r) => r.category).filter(Boolean))]

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Respostas Rápidas</h1>
          <p className="text-gray-500 text-sm mt-1">
            Gerencie atalhos de texto para agilizar o atendimento. Use "/" no chat para ativar.
          </p>
        </div>
        <button onClick={openCreate} className="btn-primary flex items-center gap-2">
          <Plus size={18} />
          Nova Resposta
        </button>
      </div>

      {/* Search */}
      <div className="relative mb-6">
        <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Buscar por atalho, conteúdo ou categoria..."
          className="input pl-10 w-full"
        />
      </div>

      {/* Categories filter */}
      {categories.length > 0 && (
        <div className="flex gap-2 mb-4 flex-wrap">
          <button
            onClick={() => setSearch('')}
            className={`badge ${!search ? 'bg-primary-100 text-primary-700' : 'bg-gray-100 text-gray-600'} cursor-pointer hover:opacity-80`}
          >
            Todas
          </button>
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setSearch(cat || '')}
              className={`badge ${search === cat ? 'bg-primary-100 text-primary-700' : 'bg-gray-100 text-gray-600'} cursor-pointer hover:opacity-80`}
            >
              {cat}
            </button>
          ))}
        </div>
      )}

      {/* List */}
      {loading ? (
        <div className="text-center py-12 text-gray-500">Carregando...</div>
      ) : filteredReplies.length === 0 ? (
        <div className="card text-center py-12">
          <Zap size={48} className="mx-auto text-gray-300 mb-4" />
          <p className="text-gray-500 text-lg">Nenhuma resposta rápida encontrada</p>
          <p className="text-gray-400 text-sm mt-1">
            Crie atalhos como "/saudacao" para responder rapidamente no chat
          </p>
        </div>
      ) : (
        <div className="grid gap-3">
          {filteredReplies.map((reply) => (
            <div key={reply.id} className="card p-4 hover:shadow-md transition-shadow">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3 mb-1">
                    <code className="bg-primary-50 text-primary-700 px-2 py-0.5 rounded text-sm font-mono font-medium">
                      {reply.shortcut}
                    </code>
                    {reply.title && (
                      <span className="text-sm font-medium text-gray-700">{reply.title}</span>
                    )}
                    {reply.category && (
                      <span className="badge bg-gray-100 text-gray-600 text-xs">
                        {reply.category}
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-gray-600 mt-1 line-clamp-2 whitespace-pre-wrap">
                    {reply.content}
                  </p>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <button
                    onClick={() => openEdit(reply)}
                    className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg transition-colors"
                    title="Editar"
                  >
                    <Pencil size={16} />
                  </button>
                  <button
                    onClick={() => handleDelete(reply.id)}
                    className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                    title="Excluir"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl w-full max-w-lg shadow-2xl">
            <div className="flex items-center justify-between p-5 border-b border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900">
                {editingReply ? 'Editar Resposta Rápida' : 'Nova Resposta Rápida'}
              </h2>
              <button
                onClick={() => setShowModal(false)}
                className="p-1 text-gray-400 hover:text-gray-600 rounded-lg"
              >
                <X size={20} />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="p-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Atalho <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={form.shortcut}
                  onChange={(e) => setForm({ ...form, shortcut: e.target.value })}
                  placeholder="/saudacao"
                  className="input w-full"
                />
                <p className="text-xs text-gray-400 mt-1">
                  Digite este atalho no chat para inserir a resposta automaticamente
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Título</label>
                <input
                  type="text"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  placeholder="Saudação inicial"
                  className="input w-full"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Conteúdo <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={form.content}
                  onChange={(e) => setForm({ ...form, content: e.target.value })}
                  placeholder="Olá! Como posso ajudá-lo hoje?"
                  rows={4}
                  className="input w-full resize-none"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Categoria</label>
                <input
                  type="text"
                  value={form.category}
                  onChange={(e) => setForm({ ...form, category: e.target.value })}
                  placeholder="Saudações, Suporte, Vendas..."
                  className="input w-full"
                />
              </div>

              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => setShowModal(false)} className="btn-secondary flex-1">
                  Cancelar
                </button>
                <button type="submit" className="btn-primary flex-1">
                  {editingReply ? 'Salvar' : 'Criar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
