'use client'

import { useEffect, useRef } from 'react'
import toast from 'react-hot-toast'
import api from '@/lib/api'
import wsService from '@/lib/websocket'
import { useAuthStore } from '@/store/auth'

type IncomingMessage = {
  id: string
  conversation_id: string
  sender_type: string
  sender_id?: string
  content?: string
  message_type?: string
}

type ConversationInfo = {
  id: string
  assigned_to?: string | null
}

const getJsonSetting = (key: string) => {
  try {
    return JSON.parse(localStorage.getItem(key) || '{}')
  } catch {
    return {}
  }
}

const getMessagePreview = (msg: IncomingMessage) => {
  if (msg.content) return msg.content
  if (msg.message_type === 'image') return 'Imagem recebida'
  if (msg.message_type === 'gif') return 'GIF recebido'
  if (msg.message_type === 'audio') return 'Áudio recebido'
  if (msg.message_type === 'video') return 'Vídeo recebido'
  return 'Mídia recebida'
}

const playNotificationSound = () => {
  try {
    const settings = getJsonSetting('notification_settings')
    if (settings.enabled === false) return

    const soundFile = settings.sound || 'notification-1'
    const volume = settings.volume != null ? settings.volume / 100 : 0.5
    const audio = new Audio(`/sounds/${soundFile}.wav`)
    audio.volume = Math.min(Math.max(volume, 0), 1)
    audio.preload = 'auto'
    audio.play().catch(() => {})
  } catch {}
}

const showBrowserNotification = (msg: IncomingMessage, preview: string) => {
  try {
    const settings = getJsonSetting('notification_settings')
    if (settings.browserEnabled === false) return
    if (!('Notification' in window) || Notification.permission !== 'granted') return

    const notification = new Notification('Nova mensagem no VGON Chat', {
      body: preview,
      icon: '/favicon.png',
      badge: '/favicon.png',
      tag: msg.conversation_id,
      silent: false,
      requireInteraction: document.visibilityState !== 'visible',
    })

    notification.onclick = () => {
      window.focus()
      window.location.href = `/conversations?conversation=${msg.conversation_id}`
      notification.close()
    }
  } catch {}
}

export function GlobalNotifications() {
  const { isAuthenticated, user } = useAuthStore()
  const seenMessagesRef = useRef<Set<string>>(new Set())
  const conversationsRef = useRef<Map<string, ConversationInfo>>(new Map())

  useEffect(() => {
    if (!isAuthenticated) return

    const settings = getJsonSetting('notification_settings')
    if (settings.browserEnabled === false) return

    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission().catch(() => {})
    }
  }, [isAuthenticated])

  useEffect(() => {
    if (!isAuthenticated || !user?.id) return

    const shouldAlert = async (msg: IncomingMessage) => {
      const alertEvents = getJsonSetting('notification_alert_events')
      const alertMine = alertEvents.assigned_to_me !== false
      const alertUnassigned = alertEvents.unassigned !== false
      const alertOthers = alertEvents.assigned_to_others !== false

      if (alertMine && alertUnassigned && alertOthers) return true

      let conversation = conversationsRef.current.get(msg.conversation_id)
      if (!conversation) {
        try {
          const response = await api.get(`/conversations/${msg.conversation_id}`)
          conversation = response.data as ConversationInfo
          conversationsRef.current.set(msg.conversation_id, conversation)
        } catch {
          return true
        }
      }

      if (conversation.assigned_to === user.id) return alertMine
      if (!conversation.assigned_to) return alertUnassigned
      return alertOthers
    }

    const handleNewMessage = async (msg: IncomingMessage) => {
      if (!msg?.id || seenMessagesRef.current.has(msg.id)) return
      seenMessagesRef.current.add(msg.id)

      if (msg.sender_type !== 'contact') return
      if (!(await shouldAlert(msg))) return

      const preview = getMessagePreview(msg)

      toast(preview, {
        duration: 5000,
      })

      showBrowserNotification(msg, preview)
      playNotificationSound()
    }

    wsService.on('new_message', handleNewMessage)
    return () => wsService.off('new_message', handleNewMessage)
  }, [isAuthenticated, user?.id])

  return null
}
