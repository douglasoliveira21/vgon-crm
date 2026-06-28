'use client'

import { useEffect, useState, useRef } from 'react'
import { useSearchParams } from 'next/navigation'
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
  UsersRound,
} from 'lucide-react'
import { clsx } from 'clsx'
import toast from 'react-hot-toast'

interface Conversation {
  id: string
  contact_id?: string
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
  reply_to_content?: string
  reply_to_sender?: string
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
  const searchParams = useSearchParams()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selectedConv, setSelectedConv] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [newMessage, setNewMessage] = useState('')
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState('mine')
  const [statusFilter, setStatusFilter] = useState('')
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
  const [isDragging, setIsDragging] = useState(false)

  // Attachment preview
  const [pendingFile, setPendingFile] = useState<{ file: File; preview: string; type: string } | null>(null)
  const [pendingCaption, setPendingCaption] = useState('')

  // Tab unread counts
  const [tabUnreadCounts, setTabUnreadCounts] = useState<{ mine: number; unassigned: number; all: number }>({ mine: 0, unassigned: 0, all: 0 })
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)

  // Context menu & reply
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; message: Message } | null>(null)
  const [replyingTo, setReplyingTo] = useState<Message | null>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const recordingIntervalRef = useRef<NodeJS.Timeout | null>(null)

  // Quick replies
  const [quickReplies, setQuickReplies] = useState<{ id: string; shortcut: string; title: string | null; content: string; category: string | null }[]>([])
  const [showQuickReplies, setShowQuickReplies] = useState(false)
  const [quickReplyFilter, setQuickReplyFilter] = useState('')
  const [selectedQuickReplyIndex, setSelectedQuickReplyIndex] = useState(0)

  useEffect(() => {
    fetchConversations()
    fetchTabUnreadCounts()
    fetchUsers()
    fetchTeams()
    fetchQuickReplies()
  }, [filter, statusFilter, searchParams])

  useEffect(() => {
    const handleNewMessage = (data: Message) => {
      if (selectedConv && data.conversation_id === selectedConv.id) {
        // User is viewing this chat - add message and don't count as unread
        if (data.sender_type === 'user' && data.sender_id === user?.id) {
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

        // Update sidebar - no unread since user is viewing
        setConversations((prev) =>
          prev.map((c) =>
            c.id === data.conversation_id
              ? { ...c, last_message_preview: data.content || '📎 Mídia', last_message_at: data.created_at, unread_count: 0 }
              : c
          )
        )
      } else {
        // Message for another chat - update sidebar with unread count
        setConversations((prev) => {
          const exists = prev.find((c) => c.id === data.conversation_id)
          if (exists) {
            return prev.map((c) =>
              c.id === data.conversation_id
                ? { ...c, last_message_preview: data.content || '📎 Mídia', last_message_at: data.created_at, unread_count: (c.unread_count || 0) + 1 }
                : c
            )
          }
          // New conversation not in list - refetch
          fetchConversations()
          return prev
        })

        // Browser notification only when not viewing this chat
        if (data.sender_type === 'contact') {
          showNotification(data)
        }
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
    // Get notification preferences
    const notifSettings = JSON.parse(localStorage.getItem('notification_settings') || '{}')
    const alertEvents = JSON.parse(localStorage.getItem('notification_alert_events') || '{}')

    // Find the conversation this message belongs to
    const conv = conversations.find(c => c.id === msg.conversation_id)

    // Check if we should alert based on conversation assignment rules
    const shouldAlert = checkShouldAlert(conv, alertEvents)
    if (!shouldAlert) return

    // Browser notification (always if permission granted)
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification('Nova mensagem', {
        body: msg.content || 'Mídia recebida',
        icon: '/favicon.ico',
      })
    }

    // Play notification sound
    // If tab is not active: always play
    // If tab is active: play only if not viewing this specific conversation
    if (document.hidden || !selectedConv || selectedConv.id !== msg.conversation_id) {
      playNotificationSound()
    }
  }

  const checkShouldAlert = (conv: Conversation | undefined, alertEvents: any) => {
    // Default: alert for all if no settings configured
    const alertMine = alertEvents.assigned_to_me !== false
    const alertUnassigned = alertEvents.unassigned !== false
    const alertOthers = alertEvents.assigned_to_others !== false

    // If conversation not in current filtered list, still alert (could be from another tab)
    if (!conv) return true

    if (conv.assigned_to === user?.id) {
      return alertMine
    }
    if (!conv.assigned_to) {
      return alertUnassigned
    }
    // Assigned to someone else
    return alertOthers
  }

  const playNotificationSound = () => {
    try {
      const settings = JSON.parse(localStorage.getItem('notification_settings') || '{}')
      if (settings.enabled === false) return
      const soundFile = settings.sound || 'notification-1'
      const volume = settings.volume != null ? settings.volume / 100 : 0.5
      const audio = new Audio(`/sounds/${soundFile}.wav`)
      audio.volume = volume
      audio.play().catch(() => {})
    } catch {}
  }

  const fetchConversations = async () => {
    try {
      const params: any = {}
      const channelParam = searchParams.get('channel')
      if (channelParam) params.channel = channelParam

      // Tab filter
      if (filter === 'mine') params.assigned_to = user?.id
      if (filter === 'unassigned') params.unassigned = 'true'

      // Status dropdown filter overrides
      if (statusFilter && statusFilter !== 'all_status') {
        params.status = statusFilter
      }

      // Default: "Todas" and "Minhas" show only non-resolved
      if (filter === 'all' && !statusFilter) {
        params.status = 'open,in_progress,pending'
      }
      if (filter === 'mine' && !statusFilter) {
        params.status = 'open,in_progress,pending'
      }
      if (filter === 'unassigned' && !statusFilter) {
        params.status = 'open,in_progress,pending'
      }

      // all_status means no status filter
      if (statusFilter === 'all_status') {
        delete params.status
      }

      const response = await api.get('/conversations', { params })
      setConversations(response.data.conversations || [])
    } catch (error) {
      console.error('Error fetching conversations:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchTabUnreadCounts = async () => {
    try {
      const [mineRes, unassignedRes, allRes] = await Promise.all([
        api.get('/conversations', { params: { assigned_to: user?.id, status: 'open,in_progress,pending', channel: searchParams.get('channel') || undefined } }),
        api.get('/conversations', { params: { unassigned: 'true', status: 'open,in_progress,pending', channel: searchParams.get('channel') || undefined } }),
        api.get('/conversations', { params: { status: 'open,in_progress,pending', channel: searchParams.get('channel') || undefined } }),
      ])
      const mineConvs: Conversation[] = mineRes.data.conversations || []
      const unassignedConvs: Conversation[] = unassignedRes.data.conversations || []
      const allConvs: Conversation[] = allRes.data.conversations || []

      setTabUnreadCounts({
        mine: mineConvs.reduce((sum, c) => sum + (c.unread_count || 0), 0),
        unassigned: unassignedConvs.reduce((sum, c) => sum + (c.unread_count || 0), 0),
        all: allConvs.reduce((sum, c) => sum + (c.unread_count || 0), 0),
      })
    } catch {}
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

  const fetchQuickReplies = async () => {
    try {
      const response = await api.get('/quick-replies')
      setQuickReplies(response.data.quick_replies || [])
    } catch {}
  }

  // Auto-refresh messages when chat is open (fallback for WebSocket)
  useEffect(() => {
    if (!selectedConv) return

    const interval = setInterval(async () => {
      try {
        const response = await api.get(`/conversations/${selectedConv.id}/messages`, { params: { limit: 200 } })
        const newMessages: Message[] = response.data.messages || []
        setMessages((prev) => {
          // Count real messages (not temp)
          const realPrev = prev.filter((m) => !m.id.startsWith('temp-'))
          // Only update if server has more messages
          if (newMessages.length > realPrev.length) {
            return newMessages
          }
          // Also update if statuses changed
          const hasStatusChange = newMessages.some((nm, i) => {
            const existing = realPrev.find(m => m.id === nm.id)
            return existing && existing.status !== nm.status
          })
          if (hasStatusChange) {
            return newMessages
          }
          return prev
        })
      } catch {}
    }, 3000)

    return () => clearInterval(interval)
  }, [selectedConv?.id])

  // Auto-refresh conversation list
  useEffect(() => {
    const interval = setInterval(() => {
      fetchConversations()
      fetchTabUnreadCounts()
    }, 10000) // Every 10 seconds

    return () => clearInterval(interval)
  }, [filter, statusFilter])

  const selectConversation = async (conv: Conversation) => {
    setSelectedConv(conv)
    if (selectedConv) wsService.leaveConversation(selectedConv.id)
    wsService.joinConversation(conv.id)

    // Mark as read in backend
    if (conv.unread_count > 0) {
      api.post(`/conversations/${conv.id}/read`).catch(() => {})
    }

    // Update local state immediately
    setConversations((prev) =>
      prev.map((c) => (c.id === conv.id ? { ...c, unread_count: 0 } : c))
    )

    try {
      const response = await api.get(`/conversations/${conv.id}/messages`, { params: { limit: 200 } })
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
      sender_name: user?.name,
      content: newMessage,
      message_type: 'text',
      status: 'sent',
      is_private: false,
      reply_to_content: replyingTo?.content || undefined,
      reply_to_sender: replyingTo ? (replyingTo.sender_type === 'user' ? 'Você' : replyingTo.sender_name || 'Contato') : undefined,
      created_at: new Date().toISOString(),
    }

    setMessages((prev) => [...prev, optimisticMsg])
    const msgContent = newMessage
    setNewMessage('')
    setReplyingTo(null)
    scrollToBottom()

    try {
      await api.post(`/conversations/${selectedConv.id}/messages/text`, {
        content: msgContent,
        reply_to_id: replyingTo?.id || null,
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

    // Show preview modal (same as drag/drop)
    const preview = file.type.startsWith('image/') ? URL.createObjectURL(file) : ''
    setPendingFile({ file, preview, type })
    setPendingCaption('')

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
          sender_name: user?.name,
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

  // Unassign conversation
  const unassignConversation = async () => {
    if (!selectedConv) return
    try {
      await api.post(`/conversations/${selectedConv.id}/unassign`)
      toast.success('Conversa desatribuída')
      setSelectedConv({ ...selectedConv, assigned_to: undefined, assigned_to_name: undefined, status: 'open' })
      fetchConversations()
    } catch {
      toast.error('Erro ao desatribuir')
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

  // Handle paste (Ctrl+V) with files
  const handlePaste = (e: React.ClipboardEvent) => {
    const items = e.clipboardData?.items
    if (!items || !selectedConv) return

    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.type.startsWith('image/') || item.type.startsWith('video/') || item.type.startsWith('audio/') || item.type.startsWith('application/')) {
        e.preventDefault()
        const file = item.getAsFile()
        if (file) {
          sendFileToChat(file)
        }
        return
      }
    }
  }

  // Handle drag & drop
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    if (!selectedConv) return

    const files = e.dataTransfer?.files
    if (files && files.length > 0) {
      sendFileToChat(files[0])
    }
  }

  // Send file from paste/drop - show preview first
  const sendFileToChat = (file: File) => {
    if (!selectedConv) return

    let mediaType = 'document'
    if (file.type.startsWith('image/')) mediaType = 'image'
    else if (file.type.startsWith('video/')) mediaType = 'video'
    else if (file.type.startsWith('audio/')) mediaType = 'audio'

    const preview = file.type.startsWith('image/') ? URL.createObjectURL(file) : ''
    setPendingFile({ file, preview, type: mediaType })
    setPendingCaption('')
  }

  // Confirm and send the pending file
  const confirmSendFile = async () => {
    if (!pendingFile || !selectedConv) return

    const reader = new FileReader()
    reader.readAsDataURL(pendingFile.file)
    reader.onloadend = async () => {
      const base64File = reader.result as string

      try {
        await api.post(`/conversations/${selectedConv!.id}/messages/media`, {
          media_base64: base64File,
          media_type: pendingFile.type,
          file_name: pendingFile.file.name,
          caption: pendingCaption,
        })

        const optimisticMsg: Message = {
          id: `temp-file-${Date.now()}`,
          conversation_id: selectedConv!.id,
          sender_type: 'user',
          sender_id: user?.id,
          sender_name: user?.name,
          content: pendingCaption || pendingFile.file.name,
          message_type: pendingFile.type,
          media_url: base64File,
          media_filename: pendingFile.file.name,
          status: 'sent',
          is_private: false,
          created_at: new Date().toISOString(),
        }
        setMessages((prev) => [...prev, optimisticMsg])
        scrollToBottom()
        toast.success('Arquivo enviado')
      } catch {
        toast.error('Erro ao enviar arquivo')
      }

      setPendingFile(null)
      setPendingCaption('')
    }
  }

  const cancelPendingFile = () => {
    if (pendingFile?.preview) URL.revokeObjectURL(pendingFile.preview)
    setPendingFile(null)
    setPendingCaption('')
  }

  // Context menu (right click on message)
  const handleMessageContextMenu = (e: React.MouseEvent, msg: Message) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, message: msg })
  }

  const closeContextMenu = () => setContextMenu(null)

  // Reply to message
  const replyToMessage = (msg: Message) => {
    setReplyingTo(msg)
    setContextMenu(null)
  }

  // Delete message
  const deleteMessage = async (msg: Message) => {
    setContextMenu(null)
    if (!confirm('Apagar esta mensagem?')) return

    try {
      await api.delete(`/conversations/${msg.conversation_id}/messages/${msg.id}`)
      setMessages((prev) => prev.filter((m) => m.id !== msg.id))
      toast.success('Mensagem apagada')
    } catch {
      toast.error('Erro ao apagar mensagem')
    }
  }

  return (
    <div className="flex h-screen">
      {/* Conversation List */}
      <div className="w-96 border-r border-gray-200 bg-white flex flex-col">
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
          <div className="flex flex-wrap gap-1">
            {[
              { id: 'mine', label: 'Minhas' },
              { id: 'unassigned', label: 'Não atribuídas' },
              { id: 'all', label: 'Todas' },
            ].map((f) => {
              const tabUnread = tabUnreadCounts[f.id as keyof typeof tabUnreadCounts] || 0

              return (
                <button
                  key={f.id}
                  onClick={() => setFilter(f.id)}
                  className={clsx(
                    'px-3 py-1.5 text-xs font-medium rounded-md transition-colors flex items-center gap-1.5',
                    filter === f.id ? 'bg-primary-100 text-primary-700' : 'text-gray-500 hover:bg-gray-100'
                  )}
                >
                  {f.label}
                  {tabUnread > 0 && (
                    <span className="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 bg-primary-600 text-white text-[10px] font-bold rounded-full">
                      {tabUnread > 99 ? '99+' : tabUnread}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
          {/* Status filter */}
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="mt-2 w-full text-xs border border-gray-200 rounded-lg px-2 py-1.5 text-gray-600 bg-gray-50 focus:border-primary-500 outline-none"
          >
            <option value="">Filtrar por status...</option>
            <option value="open">🟢 Abertos</option>
            <option value="in_progress">🔵 Em atendimento</option>
            <option value="pending">🟡 Pendentes</option>
            <option value="resolved">✅ Resolvidos</option>
            <option value="all_status">📋 Todos os status</option>
          </select>
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
              <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center flex-shrink-0 overflow-hidden">
                {conv.contact_avatar_url ? (
                  <img
                    src={conv.contact_avatar_url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${conv.contact_avatar_url}` : conv.contact_avatar_url}
                    alt=""
                    className="w-full h-full object-cover"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; (e.target as HTMLImageElement).nextElementSibling?.classList.remove('hidden') }}
                  />
                ) : (
                  <span className="text-primary-700 font-medium text-sm">
                    {conv.contact_name?.charAt(0)?.toUpperCase() || '?'}
                  </span>
                )}
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
        <div
          className={clsx('flex-1 flex flex-col bg-gray-50 relative', isDragging && 'ring-2 ring-primary-500 ring-inset')}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onPaste={handlePaste}
        >
          {/* Drag overlay */}
          {isDragging && (
            <div className="absolute inset-0 bg-primary-500/10 z-10 flex items-center justify-center pointer-events-none">
              <div className="bg-white rounded-xl shadow-lg p-6 text-center">
                <Paperclip size={32} className="text-primary-600 mx-auto mb-2" />
                <p className="text-gray-700 font-medium">Solte o arquivo aqui</p>
              </div>
            </div>
          )}
          {/* Chat Header */}
          <div className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center overflow-hidden">
                {selectedConv.contact_avatar_url ? (
                  <img
                    src={selectedConv.contact_avatar_url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${selectedConv.contact_avatar_url}` : selectedConv.contact_avatar_url}
                    alt="" className="w-full h-full object-cover"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
                  />
                ) : (
                  <span className="text-primary-700 font-medium">
                    {selectedConv.contact_name?.charAt(0)?.toUpperCase() || '?'}
                  </span>
                )}
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
                onContextMenu={(e) => handleMessageContextMenu(e, msg)}
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
                  {/* Sender name */}
                  {msg.sender_type === 'user' && msg.sender_name && (
                    <p className={`text-xs font-bold mb-0.5 ${msg.sender_type === 'user' ? 'text-white/90' : 'text-gray-700'}`}>
                      {msg.sender_name}
                    </p>
                  )}
                  {msg.sender_type === 'bot' && (
                    <p className="text-xs font-bold mb-0.5 text-purple-600">
                      🤖 {msg.sender_name || 'Bot'}
                    </p>
                  )}

                  {msg.is_private && (
                    <p className="text-xs font-medium text-yellow-600 mb-1">🔒 Nota interna</p>
                  )}

                  {/* Reply quote */}
                  {msg.reply_to_content && (
                    <div className={`mb-2 p-2 rounded-lg border-l-3 ${
                      msg.sender_type === 'user'
                        ? 'bg-white/10 border-l-white/50'
                        : 'bg-gray-100 border-l-primary-400'
                    }`}>
                      <p className={`text-xs font-medium ${msg.sender_type === 'user' ? 'text-white/80' : 'text-primary-600'}`}>
                        {msg.reply_to_sender || 'Contato'}
                      </p>
                      <p className={`text-xs truncate ${msg.sender_type === 'user' ? 'text-white/60' : 'text-gray-500'}`}>
                        {msg.reply_to_content}
                      </p>
                    </div>
                  )}

                  {msg.message_type === 'image' && msg.media_url && (
                    <div className="mb-2">
                      {msg.media_url.startsWith('/uploads/') ? (
                        <img
                          src={`${process.env.NEXT_PUBLIC_API_URL}${msg.media_url}`}
                          alt="Imagem"
                          className="max-w-full rounded-lg max-h-60 object-cover cursor-pointer"
                          onClick={() => window.open(`${process.env.NEXT_PUBLIC_API_URL}${msg.media_url}`, '_blank')}
                        />
                      ) : msg.media_url === 'sent' ? (
                        <div className="flex items-center gap-2 text-xs opacity-70">
                          <Camera size={16} /> Imagem enviada
                        </div>
                      ) : (
                        <img
                          src={`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`}
                          alt="Imagem"
                          className="max-w-full rounded-lg max-h-60 object-cover cursor-pointer"
                          onClick={() => window.open(`${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`, '_blank')}
                          onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
                        />
                      )}
                    </div>
                  )}

                  {msg.message_type === 'audio' && msg.media_url && (
                    <div className="mb-1">
                      {msg.media_url === 'sent' ? (
                        <div className="flex items-center gap-2 text-xs opacity-70">
                          <Mic size={16} /> Áudio enviado ✓
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <audio id={`audio-${msg.id}`} controls className="max-w-full h-10 flex-1" preload="metadata">
                            <source src={msg.media_url.startsWith('/uploads/') ? `${process.env.NEXT_PUBLIC_API_URL}${msg.media_url}` : `${process.env.NEXT_PUBLIC_API_URL}/media/${msg.id}`} />
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
                      )}
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
          <div className="bg-white border-t border-gray-200">
            {/* Reply bar */}
            {replyingTo && (
              <div className="px-4 pt-3 flex items-center gap-3 border-b border-gray-100 pb-2">
                <div className="flex-1 bg-gray-50 rounded-lg p-2 border-l-4 border-primary-500">
                  <p className="text-xs font-medium text-primary-600">
                    {replyingTo.sender_type === 'user' ? 'Você' : replyingTo.sender_name || 'Contato'}
                  </p>
                  <p className="text-xs text-gray-500 truncate">{replyingTo.content || '📎 Mídia'}</p>
                </div>
                <button onClick={() => setReplyingTo(null)} className="text-gray-400 hover:text-gray-600">
                  <X size={16} />
                </button>
              </div>
            )}
            <div className="p-4">
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
                    onChange={(e) => {
                      const val = e.target.value
                      setNewMessage(val)
                      // Quick reply trigger
                      if (val.startsWith('/')) {
                        setShowQuickReplies(true)
                        setQuickReplyFilter(val.slice(1).toLowerCase())
                        setSelectedQuickReplyIndex(0)
                      } else {
                        setShowQuickReplies(false)
                      }
                    }}
                    onKeyDown={(e) => {
                      if (showQuickReplies) {
                        const filtered = quickReplies.filter(
                          (r) =>
                            r.shortcut.toLowerCase().includes(quickReplyFilter) ||
                            r.content.toLowerCase().includes(quickReplyFilter) ||
                            (r.title && r.title.toLowerCase().includes(quickReplyFilter))
                        )
                        if (e.key === 'ArrowDown') {
                          e.preventDefault()
                          setSelectedQuickReplyIndex((prev) => Math.min(prev + 1, filtered.length - 1))
                        } else if (e.key === 'ArrowUp') {
                          e.preventDefault()
                          setSelectedQuickReplyIndex((prev) => Math.max(prev - 1, 0))
                        } else if (e.key === 'Enter' && filtered.length > 0) {
                          e.preventDefault()
                          setNewMessage(filtered[selectedQuickReplyIndex].content)
                          setShowQuickReplies(false)
                        } else if (e.key === 'Escape') {
                          setShowQuickReplies(false)
                        }
                      }
                    }}
                    placeholder="Digite uma mensagem... (/ para atalhos)"
                    className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500/20 outline-none pr-10"
                  />

                  {/* Quick Replies Dropdown */}
                  {showQuickReplies && (() => {
                    const filtered = quickReplies.filter(
                      (r) =>
                        r.shortcut.toLowerCase().includes(quickReplyFilter) ||
                        r.content.toLowerCase().includes(quickReplyFilter) ||
                        (r.title && r.title.toLowerCase().includes(quickReplyFilter))
                    )
                    if (filtered.length === 0) return null
                    return (
                      <div className="absolute bottom-14 left-0 w-full bg-white rounded-xl shadow-xl border border-gray-200 max-h-60 overflow-y-auto z-50">
                        <div className="p-2 border-b border-gray-100">
                          <span className="text-xs text-gray-400 font-medium px-2">Respostas Rápidas</span>
                        </div>
                        {filtered.map((reply, idx) => (
                          <button
                            key={reply.id}
                            type="button"
                            onClick={() => {
                              setNewMessage(reply.content)
                              setShowQuickReplies(false)
                            }}
                            className={clsx(
                              'w-full text-left px-3 py-2 hover:bg-gray-50 transition-colors flex items-start gap-3',
                              idx === selectedQuickReplyIndex && 'bg-primary-50'
                            )}
                          >
                            <code className="text-xs bg-gray-100 text-primary-600 px-1.5 py-0.5 rounded font-mono shrink-0 mt-0.5">
                              {reply.shortcut}
                            </code>
                            <div className="flex-1 min-w-0">
                              {reply.title && (
                                <p className="text-sm font-medium text-gray-700 truncate">{reply.title}</p>
                              )}
                              <p className="text-xs text-gray-500 truncate">{reply.content}</p>
                            </div>
                          </button>
                        ))}
                      </div>
                    )
                  })()}
                  <button
                    type="button"
                    onClick={() => setShowEmojiPicker(!showEmojiPicker)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-yellow-500"
                  >
                    <Smile size={20} />
                  </button>

                  {/* Emoji Picker */}
                  {showEmojiPicker && (
                    <div className="absolute bottom-14 right-0 z-50">
                      <div className="bg-white rounded-xl shadow-xl border border-gray-200 p-3 w-80 max-h-72 overflow-y-auto">
                        <div className="grid grid-cols-8 gap-1">
                          {['😀','😂','🥹','😍','🥰','😘','😜','🤪','😎','🤩','🥳','😇','🙃','🤔','🫡','🤫','🫣','😬','😅','😢','😭','😤','🤬','🥺','😱','🫠','🤗','🤭','👍','👎','👋','🤝','🙏','💪','❤️','🔥','✨','💯','🎉','🎊','👏','🙌','💐','🌹','🌟','⭐','🌈','☀️','🌙','💬','📩','📎','📋','✅','❌','⚠️','💡','🔔','🕐','📅','💰','🏆','🎯','🚀','💻','📱','🎵','📸','🎬','📊','📈','🏢','👤','👥','🤖','💼','📞','✉️','🔗','🔒','🛒'].map((emoji) => (
                            <button
                              key={emoji}
                              type="button"
                              onClick={() => {
                                setNewMessage((prev) => prev + emoji)
                                setShowEmojiPicker(false)
                              }}
                              className="p-1.5 text-xl hover:bg-gray-100 rounded-lg transition-colors"
                            >
                              {emoji}
                            </button>
                          ))}
                        </div>
                      </div>
                    </div>
                  )}
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
        <ContactPanel conversation={selectedConv} users={users} teams={teams} onAssignUser={transferToUser} onAssignTeam={transferToTeam} onUnassign={unassignConversation} />
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

      {/* Context Menu (right click) */}
      {contextMenu && (
        <div
          className="fixed z-50"
          style={{ top: contextMenu.y, left: contextMenu.x }}
        >
          <div className="bg-white rounded-lg shadow-lg border border-gray-200 py-1 min-w-[160px]" onClick={closeContextMenu}>
            <button
              onClick={() => replyToMessage(contextMenu.message)}
              className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-2"
            >
              ↩️ Responder
            </button>
            {contextMenu.message.sender_type === 'user' && (
              <button
                onClick={() => deleteMessage(contextMenu.message)}
                className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 flex items-center gap-2"
              >
                🗑️ Apagar mensagem
              </button>
            )}
          </div>
          <div className="fixed inset-0 -z-10" onClick={closeContextMenu} />
        </div>
      )}

      {/* File Preview Modal */}
      {pendingFile && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Enviar arquivo</h3>
              <button onClick={cancelPendingFile} className="text-gray-400 hover:text-gray-600">
                <X size={20} />
              </button>
            </div>

            {/* Preview */}
            <div className="mb-4 flex justify-center bg-gray-50 rounded-lg p-4 min-h-[200px] items-center">
              {pendingFile.type === 'image' && pendingFile.preview ? (
                <img src={pendingFile.preview} alt="Preview" className="max-h-60 rounded-lg object-contain" />
              ) : pendingFile.type === 'video' ? (
                <div className="text-center">
                  <Video size={48} className="text-purple-400 mx-auto mb-2" />
                  <p className="text-sm text-gray-600">{pendingFile.file.name}</p>
                  <p className="text-xs text-gray-400">{(pendingFile.file.size / 1024 / 1024).toFixed(2)} MB</p>
                </div>
              ) : (
                <div className="text-center">
                  <FileText size={48} className="text-orange-400 mx-auto mb-2" />
                  <p className="text-sm text-gray-600">{pendingFile.file.name}</p>
                  <p className="text-xs text-gray-400">{(pendingFile.file.size / 1024 / 1024).toFixed(2)} MB</p>
                </div>
              )}
            </div>

            {/* Caption */}
            <div className="mb-4">
              <input
                type="text"
                value={pendingCaption}
                onChange={(e) => setPendingCaption(e.target.value)}
                placeholder="Adicionar legenda (opcional)..."
                className="input"
                onKeyDown={(e) => { if (e.key === 'Enter') confirmSendFile() }}
                autoFocus
              />
            </div>

            <div className="flex gap-3">
              <button onClick={cancelPendingFile} className="btn-secondary flex-1">
                Cancelar
              </button>
              <button onClick={confirmSendFile} className="btn-primary flex-1">
                <Send size={16} /> Enviar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}


// Contact Panel with Tags and Funnel
function ContactPanel({ conversation, users, teams, onAssignUser, onAssignTeam, onUnassign }: { conversation: Conversation; users: UserItem[]; teams: TeamItem[]; onAssignUser: (userId: string) => void; onAssignTeam: (teamId: string) => void; onUnassign: () => void }) {
  const [tags, setTags] = useState<Array<{id: string; name: string; color: string}>>([])
  const [contactTags, setContactTags] = useState<Array<{id: string; name: string; color: string}>>([])
  const [funnels, setFunnels] = useState<Array<{id: string; name: string; stages: Array<{id: string; name: string}>}>>([])
  const [showTagSelect, setShowTagSelect] = useState(false)
  const [showFunnelSelect, setShowFunnelSelect] = useState(false)
  const [showAssignUser, setShowAssignUser] = useState(false)
  const [showAssignTeam, setShowAssignTeam] = useState(false)

  useEffect(() => {
    api.get('/tags').then(res => setTags(res.data.tags || [])).catch(() => {})
    api.get('/funnels').then(res => setFunnels(res.data.funnels || [])).catch(() => {})
    // Fetch contact tags
    if (conversation.contact_id) {
      api.get(`/contacts/${conversation.contact_id}`).then(res => {
        setContactTags(res.data.tags || [])
      }).catch(() => {})
    }
  }, [conversation.id])

  const addTag = async (tagId: string) => {
    try {
      await api.post(`/contacts/${conversation.contact_id}/tags`, { tag_id: tagId })
      const tag = tags.find(t => t.id === tagId)
      if (tag) setContactTags(prev => [...prev, tag])
      toast.success('Tag adicionada')
      setShowTagSelect(false)
    } catch {
      toast.error('Erro ao adicionar tag')
    }
  }

  const removeTag = async (tagId: string) => {
    try {
      await api.delete(`/contacts/${conversation.contact_id}/tags/${tagId}`)
      setContactTags(prev => prev.filter(t => t.id !== tagId))
      toast.success('Tag removida')
    } catch {}
  }

  const addToFunnel = async (funnelId: string, stageId: string) => {
    try {
      await api.post('/deals', {
        funnel_id: funnelId,
        stage_id: stageId,
        title: `${conversation.contact_name || 'Contato'} - WhatsApp`,
        value: 0,
        contact_id: conversation.contact_id || '',
      })
      toast.success('Adicionado ao funil!')
      setShowFunnelSelect(false)
    } catch {
      toast.error('Erro ao adicionar ao funil')
    }
  }

  return (
    <div className="w-72 border-l border-gray-200 bg-white p-4 overflow-y-auto hidden xl:block">
      {/* Avatar and name */}
      <div className="text-center mb-4">
        <div className="w-14 h-14 bg-primary-100 rounded-full flex items-center justify-center mx-auto mb-2 overflow-hidden">
          {conversation.contact_avatar_url ? (
            <img
              src={conversation.contact_avatar_url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${conversation.contact_avatar_url}` : conversation.contact_avatar_url}
              alt="" className="w-full h-full object-cover"
              onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
            />
          ) : (
            <span className="text-primary-700 font-bold text-lg">
              {conversation.contact_name?.charAt(0)?.toUpperCase() || '?'}
            </span>
          )}
        </div>
        <h3 className="font-semibold text-gray-900 text-sm">{conversation.contact_name}</h3>
        <p className="text-xs text-gray-500">{conversation.contact_phone}</p>
      </div>

      <div className="space-y-4">
        {/* Status */}
        <div>
          <label className="text-xs font-medium text-gray-400 uppercase">Status</label>
          <p className="text-sm mt-1">
            <span className={`badge ${
              conversation.status === 'open' ? 'badge-green' :
              conversation.status === 'in_progress' ? 'badge-blue' :
              conversation.status === 'pending' ? 'badge-yellow' : 'badge-gray'
            }`}>
              {conversation.status === 'open' && 'Aberta'}
              {conversation.status === 'pending' && 'Pendente'}
              {conversation.status === 'resolved' && 'Resolvida'}
              {conversation.status === 'in_progress' && 'Em atendimento'}
            </span>
          </p>
        </div>

        {conversation.assigned_to_name && (
          <div>
            <label className="text-xs font-medium text-gray-400 uppercase">Atendente</label>
            <div className="flex items-center justify-between mt-1">
              <p className="text-sm text-gray-700">{conversation.assigned_to_name}</p>
              <button
                onClick={onUnassign}
                className="text-xs text-red-500 hover:text-red-700 font-medium"
              >
                Desatribuir
              </button>
            </div>
          </div>
        )}

        {/* Assign to User */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-medium text-gray-400 uppercase">Atribuir a Técnico</label>
            <button
              onClick={() => { setShowAssignUser(!showAssignUser); setShowAssignTeam(false) }}
              className="text-xs text-primary-600 hover:text-primary-700"
            >
              {showAssignUser ? 'Fechar' : 'Selecionar'}
            </button>
          </div>
          {showAssignUser && (
            <div className="p-2 bg-gray-50 rounded-lg space-y-1 max-h-40 overflow-y-auto">
              {users.map(u => (
                <button
                  key={u.id}
                  onClick={() => { onAssignUser(u.id); setShowAssignUser(false) }}
                  className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-white flex items-center gap-2"
                >
                  <div className="w-6 h-6 bg-primary-100 rounded-full flex items-center justify-center flex-shrink-0">
                    <span className="text-primary-700 text-[10px] font-medium">{u.name.charAt(0)}</span>
                  </div>
                  <span className="truncate">{u.name}</span>
                  {u.is_online && <div className="w-2 h-2 bg-green-500 rounded-full ml-auto flex-shrink-0" />}
                </button>
              ))}
              {users.length === 0 && (
                <p className="text-xs text-gray-400 px-2">Nenhum técnico disponível</p>
              )}
            </div>
          )}
        </div>

        {/* Assign to Team */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-medium text-gray-400 uppercase">Atribuir a Time</label>
            <button
              onClick={() => { setShowAssignTeam(!showAssignTeam); setShowAssignUser(false) }}
              className="text-xs text-primary-600 hover:text-primary-700"
            >
              {showAssignTeam ? 'Fechar' : 'Selecionar'}
            </button>
          </div>
          {showAssignTeam && (
            <div className="p-2 bg-gray-50 rounded-lg space-y-1 max-h-40 overflow-y-auto">
              {teams.map(t => (
                <button
                  key={t.id}
                  onClick={() => { onAssignTeam(t.id); setShowAssignTeam(false) }}
                  className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-white flex items-center gap-2"
                >
                  <div className="w-6 h-6 bg-orange-100 rounded-full flex items-center justify-center flex-shrink-0">
                    <UsersRound size={12} className="text-orange-600" />
                  </div>
                  <span className="truncate">{t.name}</span>
                  <span className="text-gray-400 ml-auto text-[10px]">{t.member_count} membros</span>
                </button>
              ))}
              {teams.length === 0 && (
                <p className="text-xs text-gray-400 px-2">Nenhum time criado</p>
              )}
            </div>
          )}
        </div>

        <div>
          <label className="text-xs font-medium text-gray-400 uppercase">Canal</label>
          <p className="text-sm text-gray-700 mt-1">{conversation.channel_name || 'WhatsApp'}</p>
        </div>

        {/* Tags */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-medium text-gray-400 uppercase">Tags</label>
            <button
              onClick={() => setShowTagSelect(!showTagSelect)}
              className="text-xs text-primary-600 hover:text-primary-700"
            >
              + Adicionar
            </button>
          </div>

          {/* Tag selector */}
          {showTagSelect && (
            <div className="mb-2 p-2 bg-gray-50 rounded-lg space-y-1 max-h-32 overflow-y-auto">
              {tags.filter(t => !contactTags.find(ct => ct.id === t.id)).map(tag => (
                <button
                  key={tag.id}
                  onClick={() => addTag(tag.id)}
                  className="w-full text-left px-2 py-1 text-xs rounded hover:bg-white flex items-center gap-2"
                >
                  <span className="w-2 h-2 rounded-full" style={{ backgroundColor: tag.color }} />
                  {tag.name}
                </button>
              ))}
              {tags.filter(t => !contactTags.find(ct => ct.id === t.id)).length === 0 && (
                <p className="text-xs text-gray-400 px-2">Todas as tags já adicionadas</p>
              )}
            </div>
          )}

          {/* Contact tags */}
          <div className="flex flex-wrap gap-1">
            {contactTags.map(tag => (
              <span
                key={tag.id}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
                style={{ backgroundColor: tag.color + '20', color: tag.color }}
              >
                {tag.name}
                <button onClick={() => removeTag(tag.id)} className="opacity-50 hover:opacity-100">×</button>
              </span>
            ))}
            {contactTags.length === 0 && !showTagSelect && (
              <p className="text-xs text-gray-400">Sem tags</p>
            )}
          </div>
        </div>

        {/* Funnel */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-medium text-gray-400 uppercase">Funil de Vendas</label>
            <button
              onClick={() => setShowFunnelSelect(!showFunnelSelect)}
              className="text-xs text-primary-600 hover:text-primary-700"
            >
              + Adicionar
            </button>
          </div>

          {showFunnelSelect && (
            <div className="p-2 bg-gray-50 rounded-lg space-y-2 max-h-48 overflow-y-auto">
              {funnels.map(funnel => (
                <div key={funnel.id}>
                  <p className="text-xs font-medium text-gray-700 mb-1">{funnel.name}</p>
                  <div className="space-y-0.5 pl-2">
                    {(funnel.stages || []).map(stage => (
                      <button
                        key={stage.id}
                        onClick={() => addToFunnel(funnel.id, stage.id)}
                        className="w-full text-left px-2 py-1 text-xs rounded hover:bg-white text-gray-600"
                      >
                        → {stage.name}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
              {funnels.length === 0 && (
                <p className="text-xs text-gray-400">Nenhum funil criado</p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
