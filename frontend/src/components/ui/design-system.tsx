'use client'

import type { ReactNode } from 'react'
import { clsx } from 'clsx'
import { CheckCircle2, Clock3, AlertCircle, Circle } from 'lucide-react'

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-white sm:text-[28px]">{title}</h1>
        {description && <p className="mt-1 text-sm text-slate-400">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  )
}

export function StatCard({ label, value, icon, tone = 'cyan', detail }: { label: string; value: ReactNode; icon?: ReactNode; tone?: 'cyan' | 'blue' | 'green' | 'yellow' | 'purple' | 'red'; detail?: ReactNode }) {
  return (
    <div className="v-card v-kpi rounded-2xl p-5">
      <div className="flex items-start justify-between gap-3">
        <div className={clsx('flex h-10 w-10 items-center justify-center rounded-xl', {
          'bg-cyan-400/15 text-cyan-300': tone === 'cyan', 'bg-blue-500/15 text-blue-300': tone === 'blue',
          'bg-emerald-400/15 text-emerald-300': tone === 'green', 'bg-amber-400/15 text-amber-300': tone === 'yellow',
          'bg-purple-400/15 text-purple-300': tone === 'purple', 'bg-rose-400/15 text-rose-300': tone === 'red',
        })}>{icon}</div>
        {detail && <span className="text-xs text-slate-400">{detail}</span>}
      </div>
      <div className="mt-4 text-2xl font-bold text-white">{value}</div>
      <div className="mt-1 text-sm text-slate-400">{label}</div>
    </div>
  )
}

export function StatusBadge({ status }: { status?: string | null }) {
  const normalized = (status || '').toLowerCase()
  const config = normalized.includes('resolv') || normalized === 'connected' ? { label: normalized.includes('resolv') ? 'Resolvida' : 'Conectado', className: 'bg-emerald-400/15 text-emerald-300', icon: CheckCircle2 }
    : normalized.includes('pend') || normalized.includes('away') ? { label: normalized.includes('pend') ? 'Aguardando' : 'Ausente', className: 'bg-amber-400/15 text-amber-300', icon: Clock3 }
      : normalized.includes('erro') || normalized.includes('offline') ? { label: normalized.includes('offline') ? 'Offline' : 'Erro', className: 'bg-rose-400/15 text-rose-300', icon: AlertCircle }
        : { label: status || 'Em atendimento', className: 'bg-cyan-400/15 text-cyan-300', icon: Circle }
  const Icon = config.icon
  return <span className={clsx('inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold', config.className)}><Icon size={12} />{config.label}</span>
}

export function EmptyState({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return <div className="flex min-h-40 flex-col items-center justify-center rounded-2xl border border-dashed border-slate-700 p-8 text-center"><div className="text-sm font-semibold text-slate-200">{title}</div>{description && <p className="mt-1 max-w-md text-xs text-slate-500">{description}</p>}{action && <div className="mt-4">{action}</div>}</div>
}
