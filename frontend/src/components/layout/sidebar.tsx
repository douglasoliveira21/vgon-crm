'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import {
  LayoutDashboard,
  Inbox,
  MessageSquare,
  Users,
  UsersRound,
  Radio,
  GitBranch,
  Bot,
  Megaphone,
  Phone,
  BarChart3,
  Globe,
  Settings,
  LogOut,
  MessageCircle,
  Zap,
} from 'lucide-react'
import { clsx } from 'clsx'

const menuItems = [
  { label: 'Painel', href: '/dashboard', icon: LayoutDashboard },
  { label: 'Caixa de Entrada', href: '/inbox', icon: Inbox },
  { label: 'Conversas', href: '/conversations', icon: MessageSquare },
  { label: 'Contatos', href: '/contacts', icon: Users },
  { label: 'Times', href: '/teams', icon: UsersRound },
  { label: 'Canais', href: '/channels', icon: Radio },
  { label: 'Funil de Vendas', href: '/funnels', icon: GitBranch },
  { label: 'Bots e Automações', href: '/automations', icon: Bot },
  { label: 'Marketing', href: '/campaigns', icon: Megaphone },
  { label: 'Chamadas', href: '/calls', icon: Phone },
  { label: 'Métricas', href: '/metrics', icon: BarChart3 },
  { label: 'Widget', href: '/widget', icon: Globe },
  { label: 'Respostas Rápidas', href: '/quick-replies', icon: Zap },
  { label: 'Configurações', href: '/settings', icon: Settings },
]

export default function Sidebar() {
  const pathname = usePathname()
  const { user, logout } = useAuthStore()

  return (
    <aside className="fixed left-0 top-0 h-full w-64 bg-dark-900 border-r border-dark-700 flex flex-col z-50">
      {/* Logo */}
      <div className="p-5 flex items-center gap-3 border-b border-dark-700">
        <div className="w-10 h-10 bg-primary-600 rounded-xl flex items-center justify-center">
          <MessageCircle className="w-5 h-5 text-white" />
        </div>
        <div>
          <h1 className="text-white font-bold text-lg">EvoCRM</h1>
          <p className="text-gray-500 text-xs">Pro</p>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-1">
        {menuItems.map((item) => {
          const isActive = pathname === item.href || pathname?.startsWith(item.href + '/')
          const Icon = item.icon

          return (
            <Link
              key={item.href}
              href={item.href}
              className={clsx(
                'sidebar-item',
                isActive ? 'sidebar-item-active' : 'sidebar-item-inactive'
              )}
            >
              <Icon size={20} />
              <span>{item.label}</span>
            </Link>
          )
        })}
      </nav>

      {/* User */}
      <div className="p-4 border-t border-dark-700">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 bg-primary-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
            {user?.name?.charAt(0)?.toUpperCase() || 'U'}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-white text-sm font-medium truncate">{user?.name || 'Usuário'}</p>
            <p className="text-gray-500 text-xs truncate">{user?.role_name || 'Administrador'}</p>
          </div>
          <button
            onClick={logout}
            className="text-gray-400 hover:text-red-400 transition-colors"
            title="Sair"
          >
            <LogOut size={18} />
          </button>
        </div>
      </div>
    </aside>
  )
}
