'use client'

import { useEffect, useState, useRef } from 'react'
import api from '@/lib/api'
import wsService from '@/lib/websocket'
import { useAuthStore } from '@/store/auth'
import {
  Search,
  Send,
  Paperclip,
  Mic,
  Smile,
  MoreVertical,
  Phone,
  UserPlus,
  Tag,
  ArrowRightLeft,
  CheckCheck,
  Check,
  Image,
  FileText,
  X,
} from 'lucide-react'
import { clsx } from 'clsx'

interface Conversation {
  id: string
  contact_name: string
  contact_phone: string
  contact_avatar_url?: string
  last_message_preview: string
  last_message_at: string
  unread_count: number
  status: string
  assigned_to_name?: string
  channel_name?: string
}

interface Message {
  id: string
  conversation_id: string
  sender_type: string
  sender_id?: string
  content?: string
  message_type: string
  media_url?: string
  status: string
  is_private: boolean
  sender_name?: string
  created_at: string
}

export default function ConversationsPage() {
  const { user } = useAuthStore()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selectedConv, setSelectedConv] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [newMessage, setNewMessage] = useState('')
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState('all')
  const [loading, setLoading] = useState(true)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchConversations()
  }, [filter])

  useEffect(() => {
    // Listen for new messages via WebSocket
    const handleNewMessage = (data: Message) => {
      if (selectedConv && data.conversation_id === selectedConv.id) {
        setMessages((prev) => [...prev, data])
        scrollToBottom()
      }
      // Update conversation list
      fetchConversations()
    }

    wsService.on('new_message', handleNewMessage)
    return () => wsService.off('new_message', handleNewMessage)
  }, [selectedConv])

  const fetchConversations = async () => {
    try {
      const params: any = {}
      if (filter === 'mine') params.assigned_to = user?.id
      if (filter === 'unassigned') params.status = 'open'

      const response = await api.get('/conversations', { params })
      setConversations(response.data.conversations || [])
    } catch (error) {
      console.error('Error fetching conversations:', error)
    } finally {
      setLoading(false)
    }
  }

  const selectConversation = async (conv: Conversation) => {
    setSelectedConv(conv)

    // Leave previous room, join new one
    if (selectedConv) {
      wsService.leaveConversation(selectedConv.id)
    }
    wsService.joinConversation(conv.id)

    try {
      const response = await api.get(`/conversations/${conv.id}/messages`)
      setMessages(response.data.messages || [])
      scrollToBottom()
    } catch (error) {
      console.error('Error fetching messages:', error)
    }
  }

  const sendMessage = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newMessage.trim() || !selectedConv) return

    try {
      await api.post(`/conversations/${selectedConv.id}/messages/text`, {
        content: newMessage,
      })
      setNewMessage('')
      // Message will arrive via WebSocket
    } catch (error) {
      console.error('Error sending message:', error)
    }
  }

  const scrollToBottom = () => {
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, 100)
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'read':
        return <CheckCheck size={14} className="text-blue-500" />
      case 'delivered':
        return <CheckCheck size={14} className="text-gray-400" />
      default:
        return <Check size={14} className="text-gray-400" />
    }
  }

  const formatTime = (dateStr: string) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div className="flex h-screen">
      {/* Conversation List */}
      <div className="w-80 border-r border-gray-200 bg-white flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-gray-100">
          <h2 className="text-lg font-semibold text-gray-900 mb-3">Conversas</h2>

          {/* Search */}
          <div className="relative mb-3">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar conversa..."
              className="w-full pl-9 pr-3 py-2 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:border-primary-500 outline-none"
            />
          </div>

          {/* Filters */}
          <div className="flex gap-1">
            {[
              { id: 'all', label: 'Todas' },
              { id: 'mine', label: 'Minhas' },
              { id: 'unassigned', label: 'Sem dono' },
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => setFilter(f.id)}
                className={clsx(
                  'px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
                  filter === f.id
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-500 hover:bg-gray-100'
                )}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto">
          {conversations.map((conv) => (
            <button
              key={conv.id}
              onClick={() => selectConversation(conv)}
              className={clsx(
                'w-full p-4 flex items-start gap-3 border-b border-gray-50 hover:bg-gray-50 transition-colors text-left',
                selectedConv?.id === conv.id && 'bg-primary-50 border-l-2 border-l-primary-500'
              )}
            >
              {/* Avatar */}
              <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center flex-shrink-0">
                <span className="text-primary-700 font-medium text-sm">
                  {conv.contact_name?.charAt(0)?.toUpperCase() || '?'}
                </span>
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-gray-900 truncate">
                    {conv.contact_name || conv.contact_phone}
                  </span>
                  <span className="text-xs text-gray-400">
                    {formatTime(conv.last_message_at)}
                  </span>
                </div>
                <p className="text-xs text-gray-500 truncate mt-0.5">
                  {conv.last_message_preview || 'Sem mensagens'}
                </p>
                <div className="flex items-center gap-2 mt-1">
                  {conv.channel_name && (
                    <span className="text-xs text-gray-400">{conv.channel_name}</span>
                  )}
                  {conv.unread_count > 0 && (
                    <span className="inline-flex items-center justify-center w-5 h-5 bg-primary-600 text-white text-xs rounded-full">
                      {conv.unread_count}
                    </span>
                  )}
                </div>
              </div>
            </button>
          ))}

          {conversations.length === 0 && !loading && (
            <div className="p-8 text-center text-gray-400 text-sm">
              Nenhuma conversa encontrada
            </div>
          )}
        </div>
      </div>

      {/* Chat Area */}
      {selectedConv ? (
        <div className="flex-1 flex flex-col bg-gray-50">
          {/* Chat Header */}
          <div className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center">
                <span className="text-primary-700 font-medium">
                  {selectedConv.contact_name?.charAt(0)?.toUpperCase() || '?'}
                </span>
              </div>
              <div>
                <h3 className="font-medium text-gray-900">
                  {selectedConv.contact_name || selectedConv.contact_phone}
                </h3>
                <p className="text-xs text-gray-500">{selectedConv.contact_phone}</p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <button className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg">
                <Phone size={18} />
              </button>
              <button className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg">
                <UserPlus size={18} />
              </button>
              <button className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg">
                <ArrowRightLeft size={18} />
              </button>
              <button className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg">
                <Tag size={18} />
              </button>
              <button className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg">
                <MoreVertical size={18} />
              </button>
            </div>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto px-6 py-4 space-y-3">
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={clsx(
                  'message-enter flex',
                  msg.sender_type === 'user' ? 'justify-end' : 'justify-start'
                )}
              >
                <div
                  className={clsx(
                    'max-w-[70%] rounded-2xl px-4 py-2.5 shadow-sm',
                    msg.sender_type === 'user'
                      ? 'bg-primary-600 text-white rounded-br-md'
                      : 'bg-white text-gray-900 rounded-bl-md border border-gray-100',
                    msg.is_private && 'bg-yellow-50 border-yellow-200 text-yellow-900'
                  )}
                >
                  {msg.is_private && (
                    <p className="text-xs font-medium text-yellow-600 mb-1">🔒 Nota interna</p>
                  )}

                  {msg.message_type === 'image' && msg.media_url && (
                    <div className="mb-2">
                      <Image size={40} className="opacity-50" />
                      <span className="text-xs opacity-70">[Imagem]</span>
                    </div>
                  )}

                  {msg.message_type === 'document' && (
                    <div className="flex items-center gap-2 mb-1">
                      <FileText size={16} />
                      <span className="text-xs">[Documento]</span>
                    </div>
                  )}

                  {msg.content && (
                    <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
                  )}

                  <div className={clsx(
                    'flex items-center justify-end gap-1 mt-1',
                    msg.sender_type === 'user' ? 'text-white/70' : 'text-gray-400'
                  )}>
                    <span className="text-xs">{formatTime(msg.created_at)}</span>
                    {msg.sender_type === 'user' && getStatusIcon(msg.status)}
                  </div>
                </div>
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>

          {/* Message Input */}
          <div className="bg-white border-t border-gray-200 p-4">
            <form onSubmit={sendMessage} className="flex items-center gap-3">
              <button type="button" className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
                <Paperclip size={20} />
              </button>

              <div className="flex-1 relative">
                <input
                  type="text"
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                  placeholder="Digite uma mensagem..."
                  className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500/20 outline-none"
                />
                <button type="button" className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                  <Smile size={20} />
                </button>
              </div>

              <button type="button" className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
                <Mic size={20} />
              </button>

              <button
                type="submit"
                disabled={!newMessage.trim()}
                className="p-3 bg-primary-600 text-white rounded-xl hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Send size={18} />
              </button>
            </form>
          </div>
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center bg-gray-50">
          <div className="text-center">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <MessageSquare size={28} className="text-gray-400" />
            </div>
            <p className="text-gray-500 font-medium">Selecione uma conversa</p>
            <p className="text-gray-400 text-sm mt-1">Escolha uma conversa ao lado para começar</p>
          </div>
        </div>
      )}

      {/* Contact Panel (right side) */}
      {selectedConv && (
        <div className="w-72 border-l border-gray-200 bg-white p-4 overflow-y-auto hidden xl:block">
          <div className="text-center mb-6">
            <div className="w-16 h-16 bg-primary-100 rounded-full flex items-center justify-center mx-auto mb-3">
              <span className="text-primary-700 font-bold text-xl">
                {selectedConv.contact_name?.charAt(0)?.toUpperCase() || '?'}
              </span>
            </div>
            <h3 className="font-semibold text-gray-900">{selectedConv.contact_name}</h3>
            <p className="text-sm text-gray-500">{selectedConv.contact_phone}</p>
          </div>

          <div className="space-y-4">
            <div>
              <label className="text-xs font-medium text-gray-400 uppercase">Status</label>
              <p className="text-sm mt-1">
                <span className={clsx(
                  'badge',
                  selectedConv.status === 'open' && 'badge-green',
                  selectedConv.status === 'pending' && 'badge-yellow',
                  selectedConv.status === 'resolved' && 'badge-gray',
                )}>
                  {selectedConv.status === 'open' && 'Aberta'}
                  {selectedConv.status === 'pending' && 'Pendente'}
                  {selectedConv.status === 'resolved' && 'Resolvida'}
                  {selectedConv.status === 'in_progress' && 'Em atendimento'}
                </span>
              </p>
            </div>

            {selectedConv.assigned_to_name && (
              <div>
                <label className="text-xs font-medium text-gray-400 uppercase">Atendente</label>
                <p className="text-sm text-gray-700 mt-1">{selectedConv.assigned_to_name}</p>
              </div>
            )}

            <div>
              <label className="text-xs font-medium text-gray-400 uppercase">Canal</label>
              <p className="text-sm text-gray-700 mt-1">{selectedConv.channel_name || 'WhatsApp'}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
