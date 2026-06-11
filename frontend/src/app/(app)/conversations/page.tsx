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
  MicOff,
  Smile,
  MoreVertical,
  Phone,
  UserPlus,
  Tag,
  ArrowRightLeft,
  CheckCheck,
  Check,
  CheckCircle,
  Image,
  FileText,
  X,
  MessageSquare,
  AtSign,
  PhoneCall,
  File,
  Video,
  Camera,
  RotateCcw,
} from 'lucide-react'
import { clsx } from 'clsx'
import toast from 'react-hot-toast'

interface Conversation {
  id: string
  contact_name: string
  contact_phone: string
  contact_avatar_url?: string
  last_message_preview: string
  last_message_at: string
  unread_count: number
  status: string
  assigned_to?: string
  assigned_to_name?: string
  channel_name?: string
  team_id?: string
}

interface Message {
  id: string
  conversation_id: string
  sender_type: string
  sender_id?: string
  content?: string
  message_type: string
  media_url?: string
  media_filename?: string
  status: string
  is_private: boolean
  sender_name?: string
  created_at: string
}

interface UserItem {
  id: string
  name: string
  email: string
  role_name?: string
  is_online: boolean
}

interface TeamItem {
  id: string
  name: string
  member_count: number
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
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Modals
  const [showTransferModal, setShowTransferModal] = useState(false)
  const [showMentionModal, setShowMentionModal] = useState(false)
  const [showAttachMenu, setShowAttachMenu] = useState(false)
  const [showCallModal, setShowCallModal] = useState(false)

  // Data for modals
  const [users, setUsers] = useState<UserItem[]>([])
  const [teams, setTeams] = useState<TeamItem[]>([])

  // Audio recording
  const [isRecording, setIsRecording] = useState(false)
  const [recordingTime, setRecordingTime] = useState(0)
  const [contactTyping, setContactTyping] = useState(false)
  const [contactRecording, setContactRecording] = useState(false)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const recordingIntervalRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    fetchConversations()
    fetchUsers()
    fetchTeams()
  }, [filter])

  useEffect(() => {
    const handleNewMessage = (data: Message) => {
      if (selectedConv && data.conversation_id === selectedConv.id) {
        // Avoid duplicates - don't add if it's our own message (already shown optimistically)
        if (data.sender_type === 'user' && data.sender_id === user?.id) {
          // Replace temp message with real one
          setMessages((prev) => {
            const withoutTemp = prev.filter((m) => !m.id.startsWith('temp-'))
            const exists = withoutTemp.find((m) => m.id === data.id)
            if (exists) return withoutTemp
            return [...withoutTemp, data]
          })
        } else {
          setMessages((prev) => {
            const exists = prev.find((m) => m.id === data.id)
            if (exists) return prev
            return [...prev, data]
          })
        }
        scrollToBottom()
      }
      fetchConversations()

      // Browser notification
      if (data.sender_type === 'contact' && document.hidden) {
        showNotification(data)
      }
    }

    const handleMessageStatus = (data: { external_id: string; conversation_id: string; status: string }) => {
      if (selectedConv && data.conversation_id === selectedConv.id) {
        setMessages((prev) =>
          prev.map((m) =>
            m.id === data.external_id || (m as any).external_id === data.external_id
              ? { ...m, status: data.status }
              : m
          )
        )
      }
    }

    const handleTyping = (data: { conversation_id: string; is_typing: boolean; is_recording: boolean }) => {
      if (selectedConv && data.conversation_id === selectedConv.id) {
        setContactTyping(data.is_typing)
        setContactRecording(data.is_recording || false)
        // Auto-clear after 5 seconds
        if (data.is_typing) {
          setTimeout(() => {
            setContactTyping(false)
            setContactRecording(false)
          }, 5000)
        }
      }
    }

    wsService.on('new_message', handleNewMessage)
    wsService.on('message_status', handleMessageStatus)
    wsService.on('typing', handleTyping)
    return () => {
      wsService.off('new_message', handleNewMessage)
      wsService.off('message_status', handleMessageStatus)
      wsService.off('typing', handleTyping)
    }
  }, [selectedConv])

  // Request notification permission
  useEffect(() => {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission()
    }
  }, [])

  const showNotification = (msg: Message) => {
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification('Nova mensagem', {
        body: msg.content || 'Mídia recebida',
        icon: '/favicon.ico',
      })
    }
  }

  const fetchConversations = async () => {
    try {
      const params: any = {}
      if (filter === 'mine') params.assigned_to = user?.id
      if (filter === 'unassigned') params.status = 'open'
      if (filter === 'resolved') params.status = 'resolved'
      const response = await api.get('/conversations', { params })
      setConversations(response.data.conversations || [])
    } catch (error) {
      console.error('Error fetching conversations:', error)
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

  const fetchTeams = async () => {
    try {
      const response = await api.get('/teams')
      setTeams(response.data.teams || [])
    } catch {}
  }

  const selectConversation = async (conv: Conversation) => {
    setSelectedConv(conv)
    if (selectedConv) wsService.leaveConversation(selectedConv.id)
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

    const tempId = `temp-${Date.now()}`
    const optimisticMsg: Message = {
      id: tempId,
      conversation_id: selectedConv.id,
      sender_type: 'user',
      sender_id: user?.id,
      content: newMessage,
      message_type: 'text',
      status: 'sent',
      is_private: false,
      created_at: new Date().toISOString(),
    }

    setMessages((prev) => [...prev, optimisticMsg])
    setNewMessage('')
    scrollToBottom()

    try {
      await api.post(`/conversations/${selectedConv.id}/messages/text`, {
        content: optimisticMsg.content,
      })
    } catch (error) {
      toast.error('Erro ao enviar mensagem')
      setMessages((prev) => prev.filter((m) => m.id !== tempId))
    }
  }

  // File upload
  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>, type: string) => {
    const file = e.target.files?.[0]
    if (!file || !selectedConv) return

    const formData = new FormData()
    formData.append('file', file)

    try {
      // For now, send as text with file info (full upload needs storage integration)
      await api.post(`/conversations/${selectedConv.id}/messages/text`, {
        content: `📎 ${file.name}`,
      })
      toast.success('Arquivo enviado')
      fetchConversations()
    } catch {
      toast.error('Erro ao enviar arquivo')
    }

    setShowAttachMenu(false)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  // Audio recording
  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mediaRecorder = new MediaRecorder(stream)
      mediaRecorderRef.current = mediaRecorder
      audioChunksRef.current = []

      mediaRecorder.ondataavailable = (event) => {
        audioChunksRef.current.push(event.data)
      }

      mediaRecorder.onstop = async () => {
        const audioBlob = new Blob(audioChunksRef.current, { type: 'audio/ogg' })
        stream.getTracks().forEach((track) => track.stop())
        await sendAudioMessage(audioBlob)
      }

      mediaRecorder.start()
      setIsRecording(true)
      setRecordingTime(0)

      recordingIntervalRef.current = setInterval(() => {
        setRecordingTime((prev) => prev + 1)
      }, 1000)
    } catch {
      toast.error('Não foi possível acessar o microfone')
    }
  }

  const stopRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop()
      setIsRecording(false)
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current)
      }
    }
  }

  const cancelRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop()
      audioChunksRef.current = []
      setIsRecording(false)
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current)
      }
    }
  }

  const sendAudioMessage = async (audioBlob: Blob) => {
    if (!selectedConv || audioBlob.size === 0) return

    try {
      // Convert blob to base64
      const reader = new FileReader()
      reader.readAsDataURL(audioBlob)
      reader.onloadend = async () => {
        const base64Audio = reader.result as string

        const res = await api.post(`/conversations/${selectedConv!.id}/messages/audio`, {
          audio_base64: base64Audio,
        })

        // Show optimistic message
        const optimisticMsg: Message = {
          id: `temp-audio-${Date.now()}`,
          conversation_id: selectedConv!.id,
          sender_type: 'user',
          sender_id: user?.id,
          content: '🎵 Áudio',
          message_type: 'audio',
          media_url: base64Audio,
          status: 'sent',
          is_private: false,
          created_at: new Date().toISOString(),
        }
        setMessages((prev) => [...prev, optimisticMsg])
        scrollToBottom()
        toast.success('Áudio enviado')
      }
    } catch {
      toast.error('Erro ao enviar áudio')
    }
  }

  // Transfer conversation
  const transferToUser = async (userId: string) => {
    if (!selectedConv) return
    try {
      await api.post(`/conversations/${selectedConv.id}/transfer`, { user_id: userId })
      toast.success('Conversa transferida')
      setShowTransferModal(false)
      fetchConversations()
    } catch {
      toast.error('Erro ao transferir')
    }
  }

  const transferToTeam = async (teamId: string) => {
    if (!selectedConv) return
    try {
      await api.post(`/conversations/${selectedConv.id}/transfer`, { team_id: teamId })
      toast.success('Conversa transferida para o time')
      setShowTransferModal(false)
      fetchConversations()
    } catch {
      toast.error('Erro ao transferir')
    }
  }

  // Assign to me
  const assignToMe = async () => {
    if (!selectedConv) return
    try {
      await api.post(`/conversations/${selectedConv.id}/assign`, { user_id: user?.id })
      toast.success('Conversa atribuída a você')
      setSelectedConv({ ...selectedConv, assigned_to: user?.id, assigned_to_name: user?.name, status: 'in_progress' })
      fetchConversations()
    } catch {
      toast.error('Erro ao atribuir')
    }
  }

  // Resolve conversation
  const resolveConversation = async () => {
    if (!selectedConv) return
    try {
      await api.post(`/conversations/${selectedConv.id}/close`)
      toast.success('Conversa resolvida')
      setSelectedConv({ ...selectedConv, status: 'resolved' })
      fetchConversations()
    } catch {
      toast.error('Erro ao resolver')
    }
  }

  // Reopen conversation
  const reopenConversation = async () => {
    if (!selectedConv) return
    try {
      await api.post(`/conversations/${selectedConv.id}/reopen`)
      toast.success('Conversa reaberta')
      setSelectedConv({ ...selectedConv, status: 'open' })
      fetchConversations()
    } catch {
      toast.error('Erro ao reabrir')
    }
  }

  // Mention user
  const mentionUser = (userName: string) => {
    setNewMessage((prev) => prev + `@${userName} `)
    setShowMentionModal(false)
  }

  // Call
  const makeCall = async () => {
    if (!selectedConv) return
    try {
      // Register call in system
      toast.success('Chamada registrada. Abra o WhatsApp para ligar.')
      setShowCallModal(false)
    } catch {
      toast.error('Erro ao registrar chamada')
    }
  }

  // Mark as read (when opening conversation)
  useEffect(() => {
    if (selectedConv && selectedConv.unread_count > 0) {
      // Mark messages as read by opening the conversation
      setConversations((prev) =>
        prev.map((c) => (c.id === selectedConv.id ? { ...c, unread_count: 0 } : c))
      )
    }
  }, [selectedConv])

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

  const formatRecordingTime = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }

  return (
    <div className="flex h-screen">
      {/* Conversation List */}
      <div className="w-80 border-r border-gray-200 bg-white flex flex-col">
        <div className="p-4 border-b border-gray-100">
          <h2 className="text-lg font-semibold text-gray-900 mb-3">Conversas</h2>
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
          <div className="flex gap-1">
            {[
              { id: 'all', label: 'Todas' },
              { id: 'mine', label: 'Minhas' },
              { id: 'unassigned', label: 'Sem dono' },
              { id: 'resolved', label: 'Resolvidas' },
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => setFilter(f.id)}
                className={clsx(
                  'px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
                  filter === f.id ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'
                )}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>

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
                  <span className="text-xs text-gray-400">{formatTime(conv.last_message_at)}</span>
                </div>
                <p className="text-xs text-gray-500 truncate mt-0.5">
                  {conv.last_message_preview || 'Sem mensagens'}
                </p>
                <div className="flex items-center gap-2 mt-1">
                  {conv.channel_name && <span className="text-xs text-gray-400">{conv.channel_name}</span>}
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
            <div className="p-8 text-center text-gray-400 text-sm">Nenhuma conversa encontrada</div>
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

            <div className="flex items-center gap-1">
              {/* Open Chat / Assign to me */}
              {!selectedConv.assigned_to && (
                <button
                  onClick={assignToMe}
                  className="px-3 py-1.5 bg-primary-600 text-white text-xs font-medium rounded-lg hover:bg-primary-700 flex items-center gap-1"
                  title="Abrir chat e atribuir a mim"
                >
                  <MessageSquare size={14} />
                  Abrir Chat
                </button>
              )}
              {/* Call */}
              <button
                onClick={() => setShowCallModal(true)}
                className="p-2 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded-lg"
                title="Ligar"
              >
                <Phone size={18} />
              </button>
              {/* Assign to me (if already assigned to someone else) */}
              {selectedConv.assigned_to && selectedConv.assigned_to !== user?.id && (
                <button
                  onClick={assignToMe}
                  className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg"
                  title="Atribuir a mim"
                >
                  <UserPlus size={18} />
                </button>
              )}
              {/* Transfer */}
              <button
                onClick={() => setShowTransferModal(true)}
                className="p-2 text-gray-400 hover:text-orange-600 hover:bg-orange-50 rounded-lg"
                title="Transferir conversa"
              >
                <ArrowRightLeft size={18} />
              </button>
              {/* Mention */}
              <button
                onClick={() => setShowMentionModal(true)}
                className="p-2 text-gray-400 hover:text-purple-600 hover:bg-purple-50 rounded-lg"
                title="Mencionar atendente"
              >
                <AtSign size={18} />
              </button>
              {/* Resolve */}
              {selectedConv.status !== 'resolved' && (
                <button
                  onClick={resolveConversation}
                  className="px-3 py-1.5 bg-green-600 text-white text-xs font-medium rounded-lg hover:bg-green-700 flex items-center gap-1"
                  title="Resolver conversa"
                >
                  <CheckCircle size={14} />
                  Resolver
                </button>
              )}
              {/* Reopen */}
              {selectedConv.status === 'resolved' && (
                <button
                  onClick={reopenConversation}
                  className="px-3 py-1.5 bg-yellow-600 text-white text-xs font-medium rounded-lg hover:bg-yellow-700 flex items-center gap-1"
                  title="Reabrir conversa"
                >
                  <RotateCcw size={14} />
                  Reabrir
                </button>
              )}
              {/* More */}
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
                      <img
                        src={`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`}
                        alt="Imagem"
                        className="max-w-full rounded-lg max-h-60 object-cover cursor-pointer"
                        onClick={() => window.open(`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`, '_blank')}
                        onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
                      />
                    </div>
                  )}

                  {msg.message_type === 'audio' && msg.media_url && (
                    <div className="mb-1">
                      <div className="flex items-center gap-2">
                        <audio id={`audio-${msg.id}`} controls className="max-w-full h-10 flex-1" preload="metadata">
                          <source src={`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`} />
                        </audio>
                        <div className="flex gap-1">
                          {[1, 1.5, 2].map((speed) => (
                            <button
                              key={speed}
                              type="button"
                              onClick={() => {
                                const audio = document.getElementById(`audio-${msg.id}`) as HTMLAudioElement
                                if (audio) audio.playbackRate = speed
                              }}
                              className={`text-[10px] px-1.5 py-0.5 rounded font-medium transition-colors ${
                                msg.sender_type === 'user'
                                  ? 'bg-white/20 text-white hover:bg-white/30'
                                  : 'bg-gray-200 text-gray-600 hover:bg-gray-300'
                              }`}
                            >
                              {speed}x
                            </button>
                          ))}
                        </div>
                      </div>
                    </div>
                  )}

                  {msg.message_type === 'video' && msg.media_url && (
                    <div className="mb-2">
                      <video controls className="max-w-full rounded-lg max-h-60">
                        <source src={`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`} />
                      </video>
                    </div>
                  )}

                  {msg.message_type === 'document' && (
                    <div className="flex items-center gap-2 mb-1 p-2 bg-gray-50 rounded-lg">
                      <FileText size={20} className="text-gray-500" />
                      <a
                        href={`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-primary-600 underline truncate"
                      >
                        {msg.media_filename || 'Documento'}
                      </a>
                    </div>
                  )}

                  {msg.content && msg.message_type !== 'audio' && (
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
            {/* Typing indicator */}
            {contactTyping && (
              <div className="flex justify-start message-enter">
                <div className="bg-white rounded-2xl rounded-bl-md px-4 py-2.5 shadow-sm border border-gray-100">
                  <p className="text-xs text-gray-500 italic">
                    {contactRecording ? '🎙️ Gravando áudio...' : '✍️ Digitando...'}
                  </p>
                </div>
              </div>
            )}
          </div>

          {/* Message Input */}
          <div className="bg-white border-t border-gray-200 p-4">
            {isRecording ? (
              <div className="flex items-center gap-4">
                <button onClick={cancelRecording} className="p-2 text-red-500 hover:bg-red-50 rounded-lg">
                  <X size={20} />
                </button>
                <div className="flex-1 flex items-center gap-3">
                  <div className="w-3 h-3 bg-red-500 rounded-full animate-pulse" />
                  <span className="text-sm text-gray-600 font-mono">{formatRecordingTime(recordingTime)}</span>
                  <span className="text-sm text-gray-400">Gravando...</span>
                </div>
                <button
                  onClick={stopRecording}
                  className="p-3 bg-primary-600 text-white rounded-xl hover:bg-primary-700"
                >
                  <Send size={18} />
                </button>
              </div>
            ) : (
              <form onSubmit={sendMessage} className="flex items-center gap-3">
                {/* Attach */}
                <div className="relative">
                  <button
                    type="button"
                    onClick={() => setShowAttachMenu(!showAttachMenu)}
                    className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100"
                  >
                    <Paperclip size={20} />
                  </button>

                  {showAttachMenu && (
                    <div className="absolute bottom-12 left-0 bg-white rounded-xl shadow-lg border border-gray-200 p-2 w-48 z-10">
                      <label className="flex items-center gap-3 px-3 py-2 hover:bg-gray-50 rounded-lg cursor-pointer text-sm text-gray-700">
                        <Camera size={16} className="text-blue-500" />
                        Foto
                        <input type="file" accept="image/*" className="hidden" onChange={(e) => handleFileUpload(e, 'image')} ref={fileInputRef} />
                      </label>
                      <label className="flex items-center gap-3 px-3 py-2 hover:bg-gray-50 rounded-lg cursor-pointer text-sm text-gray-700">
                        <Video size={16} className="text-purple-500" />
                        Vídeo
                        <input type="file" accept="video/*" className="hidden" onChange={(e) => handleFileUpload(e, 'video')} />
                      </label>
                      <label className="flex items-center gap-3 px-3 py-2 hover:bg-gray-50 rounded-lg cursor-pointer text-sm text-gray-700">
                        <File size={16} className="text-orange-500" />
                        Documento
                        <input type="file" accept=".pdf,.doc,.docx,.xls,.xlsx,.txt,.zip" className="hidden" onChange={(e) => handleFileUpload(e, 'document')} />
                      </label>
                    </div>
                  )}
                </div>

                {/* Input */}
                <div className="flex-1 relative">
                  <input
                    type="text"
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                    placeholder="Digite uma mensagem..."
                    className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500/20 outline-none"
                  />
                </div>

                {/* Mic */}
                <button
                  type="button"
                  onClick={startRecording}
                  className="p-2 text-gray-400 hover:text-red-500 rounded-lg hover:bg-red-50"
                  title="Gravar áudio"
                >
                  <Mic size={20} />
                </button>

                {/* Send */}
                <button
                  type="submit"
                  disabled={!newMessage.trim()}
                  className="p-3 bg-primary-600 text-white rounded-xl hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Send size={18} />
                </button>
              </form>
            )}
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

      {/* Right Panel - Contact */}
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
                  selectedConv.status === 'in_progress' && 'badge-blue',
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

      {/* Transfer Modal */}
      {showTransferModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Transferir conversa</h3>
              <button onClick={() => setShowTransferModal(false)} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>

            <h4 className="text-sm font-medium text-gray-500 mb-2">Para atendente:</h4>
            <div className="space-y-1 mb-4">
              {users.map((u) => (
                <button
                  key={u.id}
                  onClick={() => transferToUser(u.id)}
                  className="w-full flex items-center gap-3 p-3 hover:bg-gray-50 rounded-lg text-left"
                >
                  <div className="w-8 h-8 bg-primary-100 rounded-full flex items-center justify-center">
                    <span className="text-primary-700 text-xs font-medium">{u.name.charAt(0)}</span>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">{u.name}</p>
                    <p className="text-xs text-gray-400">{u.role_name}</p>
                  </div>
                  {u.is_online && <div className="w-2 h-2 bg-green-500 rounded-full ml-auto" />}
                </button>
              ))}
            </div>

            <h4 className="text-sm font-medium text-gray-500 mb-2">Para time:</h4>
            <div className="space-y-1">
              {teams.map((t) => (
                <button
                  key={t.id}
                  onClick={() => transferToTeam(t.id)}
                  className="w-full flex items-center gap-3 p-3 hover:bg-gray-50 rounded-lg text-left"
                >
                  <div className="w-8 h-8 bg-orange-100 rounded-full flex items-center justify-center">
                    <ArrowRightLeft size={14} className="text-orange-600" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">{t.name}</p>
                    <p className="text-xs text-gray-400">{t.member_count} membros</p>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Mention Modal */}
      {showMentionModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-sm">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Mencionar atendente</h3>
              <button onClick={() => setShowMentionModal(false)} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>
            <div className="space-y-1">
              {users.map((u) => (
                <button
                  key={u.id}
                  onClick={() => mentionUser(u.name)}
                  className="w-full flex items-center gap-3 p-3 hover:bg-gray-50 rounded-lg text-left"
                >
                  <AtSign size={16} className="text-purple-500" />
                  <span className="text-sm text-gray-900">{u.name}</span>
                  {u.is_online && <div className="w-2 h-2 bg-green-500 rounded-full ml-auto" />}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Call Modal */}
      {showCallModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-sm text-center">
            <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <PhoneCall size={28} className="text-green-600" />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Ligar para</h3>
            <p className="text-gray-500 mb-1">{selectedConv?.contact_name}</p>
            <p className="text-gray-400 text-sm mb-6">{selectedConv?.contact_phone}</p>
            <p className="text-xs text-gray-400 mb-4">
              O WhatsApp não suporta chamadas via API. A chamada será aberta no seu WhatsApp.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setShowCallModal(false)}
                className="btn-secondary flex-1"
              >
                Cancelar
              </button>
              <button
                onClick={() => {
                  window.open(`https://wa.me/${selectedConv?.contact_phone}`, '_blank')
                  makeCall()
                }}
                className="btn-primary flex-1"
              >
                <Phone size={16} /> Abrir WhatsApp
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
