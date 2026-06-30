'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useAuthStore } from '@/store/auth'
import { useAppearanceStore } from '@/store/appearance'
import api from '@/lib/api'
import wsService from '@/lib/websocket'
import {
  LayoutDashboard,
  Inbox,
  MessageSquare,
  Users,
  Building2,
  UsersRound,
  Radio,
  GitBranch,
  Bot,
  Megaphone,
  BarChart3,
  Globe,
  Settings,
  LogOut,
  Zap,
  Pin,
  PinOff,
  Moon,
  Sun,
  UserCircle,
  ShieldCheck,
  ChevronUp,
  ChevronDown,
  AtSign,
  SlidersHorizontal,
} from 'lucide-react'
import { clsx } from 'clsx'

const menuItems = [
  { label: 'Painel', href: '/dashboard', icon: LayoutDashboard },
  { label: 'Caixa de Entrada', href: '/inbox', icon: Inbox },
  { label: 'Conversas', href: '/conversations', icon: MessageSquare, expandable: 'conversations' },
  { label: 'Contatos', href: '/contacts', icon: Users },
  { label: 'Empresas', href: '/companies', icon: Building2 },
  { label: 'Times', href: '/teams', icon: UsersRound, expandable: 'teams' },
  { label: 'Canais', href: '/channels', icon: Radio },
  { label: 'Funil de Vendas', href: '/funnels', icon: GitBranch },
  { label: 'Bots e Automações', href: '/automations', icon: Bot },
  { label: 'Marketing', href: '/campaigns', icon: Megaphone },
  { label: 'Métricas', href: '/metrics', icon: BarChart3 },
  { label: 'Widget', href: '/widget', icon: Globe },
  { label: 'Respostas Rápidas', href: '/quick-replies', icon: Zap },
]

const statusMeta = {
  online: { label: 'Online', dot: 'bg-green-500' },
  busy: { label: 'Ocupado', dot: 'bg-amber-500' },
  offline: { label: 'Offline', dot: 'bg-gray-400' },
}

type SidebarConversation = {
  id: string
  assigned_to?: string | null
  team_id?: string | null
  unread_count?: number
  last_message_preview?: string
}

type SidebarTeam = {
  id: string
  name: string
  is_active: boolean
  open_count?: number
  unread_count?: number
}

const Badge = ({ value }: { value?: number }) => {
  if (!value || value <= 0) return null
  return (
    <span className="ml-auto flex min-w-5 items-center justify-center rounded-full bg-primary-600 px-1.5 py-0.5 text-[10px] font-bold leading-none text-white">
      {value > 99 ? '99+' : value}
    </span>
  )
}

const resolveImage = (url?: string) => {
  if (!url) return ''
  return url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${url}` : url
}

export default function Sidebar() {
  const pathname = usePathname()
  const { user, logout, updateStatus } = useAuthStore()
  const { sidebarPinned, setSidebarPinned, setSidebarHovered, theme, toggleTheme } = useAppearanceStore()
  const [profileMenuOpen, setProfileMenuOpen] = useState(false)
  const [expandedSections, setExpandedSections] = useState<{ conversations: boolean; teams: boolean }>({ conversations: true, teams: false })
  const [conversationCounts, setConversationCounts] = useState({ mine: 0, unassigned: 0, all: 0, mentions: 0 })
  const [teams, setTeams] = useState<SidebarTeam[]>([])
  const profileMenuRef = useRef<HTMLDivElement>(null)
  const refreshTimerRef = useRef<NodeJS.Timeout | null>(null)
  const expandedClass = sidebarPinned ? 'w-64' : 'w-20 hover:w-64'
  const showTextClass = sidebarPinned ? 'opacity-100' : 'opacity-0 group-hover/sidebar:opacity-100'
  const currentStatus = user?.availability_status || (user?.is_online ? 'online' : 'offline')
  const currentStatusMeta = statusMeta[currentStatus] || statusMeta.offline
  const profileMenuItems = [
    { label: 'Perfil', href: '/profile', icon: UserCircle, show: true },
    { label: 'Configurações', href: '/settings', icon: Settings, show: true },
    { label: 'Super Admin', href: '/admin', icon: ShieldCheck, show: !!user?.is_super_admin },
  ]

  const countUnread = (items: SidebarConversation[]) => items.reduce((sum, item) => sum + (item.unread_count || 0), 0)

  const fetchSidebarCounters = async () => {
    if (!user?.id) return
    try {
      const status = 'open,in_progress,pending'
      const [mineRes, unassignedRes, allRes, teamsRes] = await Promise.all([
        api.get('/conversations', { params: { assigned_to: user.id, status, limit: 200 } }),
        api.get('/conversations', { params: { unassigned: 'true', status, limit: 200 } }),
        api.get('/conversations', { params: { status, limit: 200 } }),
        api.get('/teams'),
      ])

      const mine = (mineRes.data.conversations || []) as SidebarConversation[]
      const unassigned = (unassignedRes.data.conversations || []) as SidebarConversation[]
      const all = (allRes.data.conversations || []) as SidebarConversation[]
      const userTokens = [user.name, user.email?.split('@')[0]]
        .filter(Boolean)
        .map((value) => `@${String(value).toLowerCase()}`)
      const mentions = all.filter((conversation) => {
        const preview = (conversation.last_message_preview || '').toLowerCase()
        return userTokens.some((token) => preview.includes(token))
      })

      setConversationCounts({
        mine: countUnread(mine),
        unassigned: countUnread(unassigned),
        all: countUnread(all),
        mentions: countUnread(mentions),
      })
      setTeams((teamsRes.data.teams || []).filter((team: SidebarTeam) => team.is_active !== false))
    } catch {}
  }

  const scheduleSidebarRefresh = () => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current)
    refreshTimerRef.current = setTimeout(() => {
      fetchSidebarCounters()
    }, 500)
  }

  useEffect(() => {
    const closeMenu = (event: MouseEvent) => {
      if (!profileMenuRef.current?.contains(event.target as Node)) {
        setProfileMenuOpen(false)
      }
    }

    document.addEventListener('mousedown', closeMenu)
    return () => document.removeEventListener('mousedown', closeMenu)
  }, [])

  useEffect(() => {
    fetchSidebarCounters()
    const interval = setInterval(fetchSidebarCounters, 30000)
    const handleNewMessage = () => scheduleSidebarRefresh()
    wsService.on('new_message', handleNewMessage)
    return () => {
      clearInterval(interval)
      if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current)
      wsService.off('new_message', handleNewMessage)
    }
  }, [user?.id])

  return (
    <aside
      onMouseEnter={() => setSidebarHovered(true)}
      onMouseLeave={() => setSidebarHovered(false)}
      className={clsx(
        'group/sidebar fixed left-0 top-0 z-50 flex h-full flex-col border-r transition-all duration-300',
        expandedClass,
        'border-dark-700 bg-dark-900 dark:border-gray-800 dark:bg-gray-950'
      )}
    >
      <div className="flex h-[92px] items-center justify-center border-b border-dark-700 px-4 dark:border-gray-800">
        <img
          src="/favicon.png"
          alt="VGON Chat"
          className={clsx(
            'object-contain transition-all duration-200',
            sidebarPinned ? 'hidden' : 'h-10 w-10 rounded-xl group-hover/sidebar:hidden'
          )}
        />
        <img
          src="/logo-white.png"
          alt="VGON Chat"
          className={clsx(
            'h-14 w-auto max-w-[210px] object-contain transition-all duration-200',
            sidebarPinned ? 'block' : 'hidden group-hover/sidebar:block'
          )}
        />
      </div>

      <div className="flex items-center justify-between gap-2 px-4 py-3">
        <button
          type="button"
          onClick={() => setSidebarPinned(!sidebarPinned)}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
          title={sidebarPinned ? 'Desfixar menu' : 'Fixar menu'}
        >
          {sidebarPinned ? <PinOff size={17} /> : <Pin size={17} />}
        </button>
        <button
          type="button"
          onClick={toggleTheme}
          className={clsx(
            'h-9 items-center gap-2 rounded-lg px-3 text-sm text-gray-300 transition-colors hover:bg-white/10 hover:text-white',
            sidebarPinned ? 'flex' : 'hidden group-hover/sidebar:flex'
          )}
          title={theme === 'dark' ? 'Usar tema claro' : 'Usar tema escuro'}
        >
          {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
          <span>{theme === 'dark' ? 'Claro' : 'Escuro'}</span>
        </button>
      </div>

      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-2">
        {menuItems.map((item) => {
          const isActive = pathname === item.href || pathname?.startsWith(item.href + '/')
          const Icon = item.icon
          const sectionKey = item.expandable as 'conversations' | 'teams' | undefined
          const sectionOpen = sectionKey ? expandedSections[sectionKey] : false
          const mainBadge = sectionKey === 'conversations'
            ? conversationCounts.unassigned
            : sectionKey === 'teams'
              ? teams.reduce((sum, team) => sum + (team.unread_count || 0), 0)
              : 0

          return (
            <div key={item.href}>
              <div
                className={clsx(
                  'flex items-center gap-2 rounded-lg text-sm font-medium transition-all',
                  isActive
                    ? 'bg-primary-600/20 text-primary-300'
                    : 'text-gray-400 hover:bg-white/5 hover:text-white'
                )}
              >
                <Link href={item.href} title={item.label} className="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5">
                  <Icon size={20} className="shrink-0" />
                  <span className={clsx('min-w-0 flex-1 whitespace-nowrap transition-opacity duration-200', showTextClass)}>
                    {item.label}
                  </span>
                  <span className={clsx(showTextClass)}><Badge value={mainBadge} /></span>
                </Link>
                {sectionKey && (
                  <button
                    type="button"
                    onClick={() => setExpandedSections((prev) => ({ ...prev, [sectionKey]: !prev[sectionKey] }))}
                    className={clsx(
                      'mr-2 rounded-md p-1.5 transition-colors hover:bg-white/10',
                      sidebarPinned ? 'block' : 'hidden group-hover/sidebar:block'
                    )}
                    title={sectionOpen ? 'Recolher menu' : 'Expandir menu'}
                  >
                    <ChevronDown size={15} className={clsx('transition-transform', sectionOpen && 'rotate-180')} />
                  </button>
                )}
              </div>

              {sectionKey === 'conversations' && sectionOpen && (
                <div className={clsx('mt-1 space-y-1 pl-9 pr-2 transition-opacity duration-200', showTextClass)}>
                  <SubMenuLink href="/conversations?view=mine" label="Minhas" count={conversationCounts.mine} />
                  <SubMenuLink href="/conversations?view=unassigned" label="Nao atribuidas" count={conversationCounts.unassigned} />
                  <SubMenuLink href="/conversations?view=all" label="Todas" count={conversationCounts.all} />
                  <SubMenuLink href="/conversations?mentions=true" label="Mencoes" count={conversationCounts.mentions} icon={<AtSign size={13} />} />
                </div>
              )}

              {sectionKey === 'teams' && sectionOpen && (
                <div className={clsx('mt-1 space-y-1 pl-9 pr-2 transition-opacity duration-200', showTextClass)}>
                  <SubMenuLink href="/teams" label="Configurar times" icon={<SlidersHorizontal size={13} />} />
                  {teams.map((team) => (
                    <SubMenuLink
                      key={team.id}
                      href={`/conversations?team_id=${team.id}`}
                      label={team.name}
                      count={team.unread_count}
                    />
                  ))}
                  {teams.length === 0 && (
                    <span className="block truncate px-2 py-1.5 text-xs text-gray-500">Nenhum time ativo</span>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </nav>

      <div ref={profileMenuRef} className="relative border-t border-dark-700 p-3 dark:border-gray-800">
        {profileMenuOpen && (
          <div
            className={clsx(
              'absolute bottom-full left-3 right-3 mb-2 overflow-hidden rounded-lg border border-white/10 bg-dark-800 py-1 shadow-xl',
              sidebarPinned ? 'block' : 'hidden group-hover/sidebar:block'
            )}
          >
            {profileMenuItems.filter((item) => item.show).map((item) => {
              const Icon = item.icon

              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={() => setProfileMenuOpen(false)}
                  className="flex items-center gap-3 px-3 py-2.5 text-sm text-gray-300 transition-colors hover:bg-white/10 hover:text-white"
                >
                  <Icon size={17} className="shrink-0" />
                  <span>{item.label}</span>
                </Link>
              )
            })}
          </div>
        )}

        <div className={clsx('mb-3 transition-opacity duration-200', showTextClass)}>
          <label className="mb-1 block text-xs font-medium text-gray-500">Status do atendente</label>
          <select
            value={currentStatus}
            onChange={(e) => updateStatus(e.target.value as 'online' | 'offline' | 'busy')}
            className="w-full rounded-lg border border-white/10 bg-dark-800 px-3 py-2 text-sm text-white outline-none focus:border-primary-500"
          >
            <option value="online">Online</option>
            <option value="busy">Ocupado</option>
            <option value="offline">Offline</option>
          </select>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setProfileMenuOpen((open) => !open)}
            title="Menu do usuário"
            className="relative shrink-0"
          >
            {user?.avatar_url ? (
              <img src={resolveImage(user.avatar_url)} alt={user.name} className="h-10 w-10 rounded-full object-cover" />
            ) : (
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-600 text-sm font-medium text-white">
                {user?.name?.charAt(0)?.toUpperCase() || 'U'}
              </div>
            )}
            <span className={clsx('absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-dark-900', currentStatusMeta.dot)} />
          </button>

          <button
            type="button"
            onClick={() => setProfileMenuOpen((open) => !open)}
            className={clsx('min-w-0 flex-1 text-left transition-opacity duration-200', showTextClass)}
          >
            <p className="truncate text-sm font-medium text-white">{user?.name || 'Usuário'}</p>
            <p className="truncate text-xs text-gray-500">{currentStatusMeta.label} • {user?.role_name || 'Atendente'}</p>
          </button>

          <button
            type="button"
            onClick={() => setProfileMenuOpen((open) => !open)}
            className={clsx('text-gray-400 transition-colors hover:text-white', sidebarPinned ? 'block' : 'hidden group-hover/sidebar:block')}
            title={profileMenuOpen ? 'Fechar menu do usuário' : 'Abrir menu do usuário'}
          >
            <ChevronUp size={18} className={clsx('transition-transform', profileMenuOpen && 'rotate-180')} />
          </button>

          <button
            onClick={logout}
            className={clsx('text-gray-400 transition-colors hover:text-red-400', sidebarPinned ? 'block' : 'hidden group-hover/sidebar:block')}
            title="Sair"
          >
            <LogOut size={18} />
          </button>
        </div>
      </div>
    </aside>
  )
}

function SubMenuLink({ href, label, count, icon }: { href: string; label: string; count?: number; icon?: ReactNode }) {
  return (
    <Link
      href={href}
      title={label}
      className="flex min-w-0 items-center gap-2 rounded-md px-2 py-1.5 text-xs font-medium text-gray-500 transition-colors hover:bg-white/5 hover:text-white"
    >
      {icon}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      <Badge value={count} />
    </Link>
  )
}
