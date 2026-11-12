'use client'

import { useEffect, useRef, useState } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Bell, HelpCircle, X, RefreshCw, MessageSquare } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '@/lib/api'
import wsService from '@/lib/websocket'
import { useAuthStore } from '@/store/auth'

type Announcement = { id: string; title: string; content: string; is_read: boolean; published_at?: string }
type Conversation = { id: string; contact_name: string; unread_count: number; last_message_preview?: string; last_message_at?: string }

const eventDateFormatter = new Intl.DateTimeFormat('pt-BR', {
  day: '2-digit', month: '2-digit', year: 'numeric',
  hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
})

function EventDate({ value, label }: { value?: string; label: string }) {
  const date = value ? new Date(value) : null
  if (!date || !Number.isFinite(date.getTime())) {
    return <p className="mt-2 text-xs text-gray-500">{label}: data indisponível</p>
  }
  return <time dateTime={date.toISOString()} title={`${label} — horário local`} className="mt-2 block text-xs text-gray-500 dark:text-gray-400">{label}: {eventDateFormatter.format(date)}</time>
}

export default function HeaderActions() {
  const [panel, setPanel] = useState<'notifications' | 'help' | null>(null)
  const [reading, setReading] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const bellRef = useRef<HTMLButtonElement>(null)
  const helpRef = useRef<HTMLButtonElement>(null)
  const pathname = usePathname()
  const user = useAuthStore((state) => state.user)
  const queryClient = useQueryClient()
  const key = ['header-notifications', user?.company_id, user?.id]
  const { data, isLoading, isError, isFetching, refetch } = useQuery({
    queryKey: key,
    enabled: !!user?.id,
    queryFn: async () => {
      const [announcements, conversations] = await Promise.all([
        api.get('/announcements'),
        api.get('/conversations', { params: { limit: 100, status: 'open,in_progress,pending' } }),
      ])
      return {
        announcements: (announcements.data.announcements || []) as Announcement[],
        conversations: ((conversations.data.conversations || []) as Conversation[]).filter((item) => item.unread_count > 0),
      }
    },
    refetchInterval: 30000,
  })
  const unreadAnnouncements = data?.announcements.filter((item) => !item.is_read) || []
  const conversations = data?.conversations || []
  const count = unreadAnnouncements.length + conversations.length

  useEffect(() => { setPanel(null) }, [pathname])
  useEffect(() => {
    const outside = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setPanel(null)
    }
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && panel) {
        setPanel(null)
        ;(panel === 'help' ? helpRef : bellRef).current?.focus()
      }
    }
    document.addEventListener('mousedown', outside)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('mousedown', outside)
      document.removeEventListener('keydown', escape)
    }
  }, [panel])
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined
    const refresh = () => {
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => queryClient.invalidateQueries({ queryKey: ['header-notifications'] }), 500)
    }
    wsService.on('new_message', refresh)
    wsService.on('conversation_update', refresh)
    return () => {
      if (timer) clearTimeout(timer)
      wsService.off('new_message', refresh)
      wsService.off('conversation_update', refresh)
    }
  }, [queryClient])

  const markRead = async (id: string) => {
    setReading(id)
    try {
      await api.post(`/announcements/${id}/read`)
      await queryClient.invalidateQueries({ queryKey: key })
    } catch { toast.error('Não foi possível marcar o comunicado como lido') }
    finally { setReading(null) }
  }

  return (
    <div ref={containerRef} className="relative ml-auto flex items-center gap-2 text-gray-600 dark:text-gray-300">
      <button ref={bellRef} type="button" onClick={() => setPanel(panel === 'notifications' ? null : 'notifications')} aria-label={`Notificações${count ? `: ${count} pendentes` : ''}`} aria-expanded={panel === 'notifications'} aria-controls="header-notifications" className="relative rounded-lg p-2 hover:bg-gray-100 dark:hover:bg-gray-800">
        <Bell size={18} />
        {count > 0 && <span className="absolute -right-1 -top-1 rounded-full bg-red-600 px-1 text-[10px] font-bold text-white">{count > 99 ? '99+' : count}</span>}
      </button>
      <button ref={helpRef} type="button" onClick={() => setPanel(panel === 'help' ? null : 'help')} aria-label="Ajuda" aria-expanded={panel === 'help'} aria-controls="header-help" className="rounded-lg p-2 hover:bg-gray-100 dark:hover:bg-gray-800"><HelpCircle size={18} /></button>
      {panel && <section id={`header-${panel}`} role="region" aria-label={panel === 'help' ? 'Ajuda do sistema' : 'Central de notificações'} className="absolute right-0 top-full z-50 mt-3 max-h-[75dvh] w-[min(24rem,calc(100vw-2rem))] overflow-y-auto rounded-2xl border border-gray-200 bg-white p-4 text-gray-900 shadow-xl dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100">
        <div className="mb-4 flex items-center justify-between gap-3">
          <h2 className="font-semibold">{panel === 'help' ? 'Como podemos ajudar?' : 'Notificações'}</h2>
          <button type="button" onClick={() => { setPanel(null); (panel === 'help' ? helpRef : bellRef).current?.focus() }} aria-label="Fechar painel" className="rounded-lg p-1 hover:bg-gray-100 dark:hover:bg-gray-800"><X size={18} /></button>
        </div>
        {panel === 'notifications' ? <>
          <button type="button" disabled={isFetching} onClick={() => refetch()} className="mb-3 flex items-center gap-2 text-xs text-primary-600 disabled:opacity-50"><RefreshCw size={13} className={isFetching ? 'animate-spin' : ''} />Atualizar notificações</button>
          {isLoading && <p className="py-5 text-sm text-gray-500">Carregando notificações...</p>}
          {isError && <p role="alert" className="py-3 text-sm text-red-600">Não foi possível carregar. Clique em atualizar para tentar novamente.</p>}
          {!isLoading && !isError && count === 0 && <p className="py-5 text-sm text-gray-500">Tudo em dia! Nenhuma mensagem ou comunicado pendente nas conversas recentes.</p>}
          <div className="space-y-3">
            {conversations.map((item) => <Link key={item.id} href={`/conversations?conversation=${encodeURIComponent(item.id)}`} onClick={() => setPanel(null)} className="block rounded-xl border border-gray-200 p-3 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800"><div className="flex items-center gap-2 text-sm font-semibold"><MessageSquare size={14} />{item.contact_name || 'Conversa'}<span className="ml-auto text-xs text-primary-600">{item.unread_count} não lida(s)</span></div><p className="mt-1 truncate text-xs text-gray-500">{item.last_message_preview || 'Abrir conversa'}</p><EventDate value={item.last_message_at} label="Última mensagem" /></Link>)}
            {unreadAnnouncements.map((item) => <div key={item.id} className="rounded-xl border border-gray-200 p-3 dark:border-gray-700"><p className="text-sm font-semibold">{item.title}</p><p className="mt-1 whitespace-pre-wrap text-xs text-gray-600 dark:text-gray-400">{item.content}</p><EventDate value={item.published_at} label="Publicado em" /><button type="button" disabled={reading !== null} onClick={() => markRead(item.id)} className="mt-2 text-xs font-medium text-primary-600 disabled:opacity-50">{reading === item.id ? 'Salvando...' : 'Marcar como lido'}</button></div>)}
          </div>
          <Link href="/conversations?view=all" onClick={() => setPanel(null)} className="mt-4 block text-sm font-medium text-primary-600">Ver todas as conversas</Link>
        </> : <div className="space-y-4 text-sm">
          <p className="text-gray-600 dark:text-gray-400">Atalhos e orientações para o atendimento no VGON.</p>
          <details className="rounded-xl border border-gray-200 p-3 dark:border-gray-700"><summary className="cursor-pointer font-medium">Buscar no sistema</summary><p className="mt-2 text-gray-600 dark:text-gray-400">Pressione Ctrl+K (ou Cmd+K no Mac), digite um nome, telefone, e-mail ou empresa e pressione Enter.</p></details>
          <details className="rounded-xl border border-gray-200 p-3 dark:border-gray-700"><summary className="cursor-pointer font-medium">Conversas e respostas rápidas</summary><p className="mt-2 text-gray-600 dark:text-gray-400">Abra uma conversa na lista. Digite / para selecionar uma resposta rápida. No computador, Enter envia e Shift+Enter insere uma nova linha.</p></details>
          <details className="rounded-xl border border-gray-200 p-3 dark:border-gray-700"><summary className="cursor-pointer font-medium">Transferir ou finalizar atendimento</summary><p className="mt-2 text-gray-600 dark:text-gray-400">Use as ações do cabeçalho da conversa para transferir a um atendente ou time. Clique em Resolver ao concluir; o atendimento pode ser reaberto.</p></details>
          <Link href="/settings" onClick={() => setPanel(null)} className="block font-medium text-primary-600">Configurações e preferências</Link>
          <Link href="/profile" onClick={() => setPanel(null)} className="block font-medium text-primary-600">Meu perfil</Link>
        </div>}
      </section>}
    </div>
  )
}
