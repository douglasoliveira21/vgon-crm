'use client'

import { useMemo, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuthStore } from '@/store/auth'
import api from '@/lib/api'
import {
  AlertTriangle,
  BarChart3,
  Clock,
  Gauge,
  MessageSquare,
  RefreshCw,
  ShieldCheck,
  Timer,
  TrendingUp,
  UserCheck,
  Users,
  Wifi,
} from 'lucide-react'
import { clsx } from 'clsx'

type DashboardData = {
	personal?: boolean
	supervisor?: boolean
  stats: {
    open_conversations: number
    pending_conversations: number
    my_conversations: number
    active_deals: number
    total_deals_value: number
    connected_channels: number
    avg_response_time: number
  }
  operations: {
    active_conversations: number
    queue_size: number
    average_wait_seconds: number
    first_response_seconds: number
    average_handle_seconds: number
    abandonment_rate: number
    first_contact_resolution_rate: number
    transfers: number
    comparison?: {
      active_conversations: number
      queue_size: number
      average_wait_seconds: number
      first_response_seconds: number
    }
  }
  queue_by_channel: Array<{ name: string; type: string; queue_size: number; active_count: number; avg_wait_seconds: number }>
  peak_hours: Array<{ label: string; total: number }>
  resolution_by_channel: Array<{ name: string; type: string; avg_resolution_seconds: number; resolved_count: number }>
  agents: {
    availability: { online: number; busy: number; pause: number; offline: number }
    workload: Array<{ id: string; name: string; status: string; active_chats: number; idle_seconds: number }>
  }
  channel_distribution: Array<{ name: string; type: string; total: number }>
  sla_by_channel: Array<{ name: string; type: string; first_response_sla: number; resolution_sla: number }>
  sla_alerts: Array<{ id: string; contact_name: string; channel_name: string; status: string; seconds_remaining: number }>
  filters: { teams: Array<{ id: string; name: string }> }
  channels: Array<{ id: string; name: string; type: string; status: string }>
}

const AUTO_REFRESH_MS = 60000

export default function DashboardPage() {
  const { user } = useAuthStore()
  const [period, setPeriod] = useState('today')
  const [teamId, setTeamId] = useState('')
  const [channelId, setChannelId] = useState('')

  const {
    data,
    isLoading: loading,
    isError: error,
    isFetching: refreshing,
    dataUpdatedAt,
    refetch,
  } = useQuery({
    queryKey: ['dashboard', period, teamId, channelId],
    queryFn: async () => {
      const response = await api.get('/dashboard', {
        params: {
          period,
          team_id: teamId || undefined,
          channel_id: channelId || undefined,
        },
      })
      return response.data as DashboardData
    },
    refetchInterval: AUTO_REFRESH_MS,
  })
  const lastUpdated = dataUpdatedAt ? new Date(dataUpdatedAt) : null
  const fetchDashboard = () => refetch()

  const maxPeak = useMemo(() => Math.max(...(data?.peak_hours || []).map((item) => item.total), 1), [data])
  const channelTotal = useMemo(() => (data?.channel_distribution || []).reduce((sum, item) => sum + item.total, 0), [data])

  if (loading && !data) {
    return (
    <div className="animate-pulse p-4 sm:p-6">
        <div className="h-8 bg-gray-200 dark:bg-gray-800 rounded w-64 mb-6" />
      <div className="mb-6 grid grid-cols-1 gap-3 min-[420px]:grid-cols-2 lg:grid-cols-4 lg:gap-4">
          {[1, 2, 3, 4].map((i) => <div key={i} className="h-28 bg-gray-200 dark:bg-gray-800 rounded-xl" />)}
        </div>
      </div>
    )
  }

  if (error && !data) {
    return (
      <div className="p-6">
        <div className="card p-12 text-center">
          <p className="text-red-500">Erro ao carregar dashboard</p>
          <button onClick={() => fetchDashboard()} className="btn-primary mt-4">Tentar novamente</button>
        </div>
      </div>
    )
  }

  const operations = data?.operations
  const comparison = operations?.comparison
  const availability = data?.agents?.availability || { online: 0, busy: 0, pause: 0, offline: 0 }
	const personal = user?.role_slug === 'agent' || data?.personal
	const supervisor = user?.role_slug === 'supervisor' || data?.supervisor

  return (
    <div className="app-surface mx-auto min-h-screen max-w-[1700px] space-y-6 p-4 sm:p-6 lg:p-7">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
		  <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-[28px]">{personal ? 'Meu desempenho' : supervisor ? 'Desempenho do meu time' : 'Dashboard de Atendimento'}</h1>
          <p className="mt-1 text-sm text-slate-400">
	            Visão geral, desempenho da equipe, canais e conversas em tempo real.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <select value={period} onChange={(e) => setPeriod(e.target.value)} className="input w-auto border-slate-700 bg-slate-50 dark:bg-slate-900/80 text-slate-200">
            <option value="today">Hoje</option>
            <option value="7d">Ultimos 7 dias</option>
            <option value="30d">Ultimos 30 dias</option>
          </select>
		  {!personal && <select value={teamId} onChange={(e) => setTeamId(e.target.value)} className="input w-auto border-slate-700 bg-slate-50 dark:bg-slate-900/80 text-slate-200">
			<option value="">{supervisor ? 'Time supervisionado' : 'Todas as equipes'}</option>
            {data?.filters?.teams?.map((team) => <option key={team.id} value={team.id}>{team.name}</option>)}
		  </select>}
          <select value={channelId} onChange={(e) => setChannelId(e.target.value)} className="input w-auto border-slate-700 bg-slate-50 dark:bg-slate-900/80 text-slate-200">
            <option value="">Todos os canais</option>
            {data?.channels?.map((channel) => <option key={channel.id} value={channel.id}>{channel.name}</option>)}
          </select>
          <button onClick={() => fetchDashboard()} className="btn-secondary border border-cyan-400/40 bg-cyan-400/10 text-cyan-200 hover:bg-cyan-400/20">
            <RefreshCw size={16} className={clsx(refreshing && 'animate-spin')} />
            Atualizar
          </button>
        </div>
      </div>

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Tempo real</h2>
          <span className="text-xs text-gray-400">Ultima atualizacao: {lastUpdated ? lastUpdated.toLocaleTimeString('pt-BR') : '-'}</span>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
		  <MetricCard title="Conversas hoje" value={data?.stats?.open_conversations || operations?.active_conversations || 0} icon={<MessageSquare size={20} />} tone="blue" delta={personal ? undefined : comparisonLabel(comparison?.active_conversations)} />
		  <MetricCard title="Em atendimento" value={operations?.active_conversations || 0} icon={<UserCheck size={20} />} tone="cyan" />
		  <MetricCard title="Aguardando" value={operations?.queue_size || 0} icon={<Clock size={20} />} tone={(operations?.queue_size || 0) > 5 ? 'red' : 'yellow'} delta={personal ? undefined : comparisonLabel(comparison?.queue_size)} />
		  <MetricCard title="FCR" value={`${formatPercent(operations?.first_contact_resolution_rate || 0)}%`} icon={<ShieldCheck size={20} />} tone="green" />
		  <MetricCard title="TMA médio" value={formatDuration(operations?.average_handle_seconds || 0)} icon={<Timer size={20} />} tone="purple" />
		  <MetricCard title="SLA de atendimento" value={`${formatPercent(slaAverage(data?.sla_by_channel || []))}%`} icon={<Gauge size={20} />} tone="cyan" />
        </div>
      </section>

      <section className="grid grid-cols-1 gap-5 xl:grid-cols-5">
        <RealtimeGauge active={operations?.active_conversations || 0} waiting={operations?.queue_size || 0} paused={availability.pause} abandoned={operations?.abandonment_rate || 0} className="xl:col-span-2" />
        <VolumeChart points={data?.peak_hours || []} className="xl:col-span-3" />
      </section>

      <section className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <AvailabilityCard availability={availability} total={(availability.online || 0) + (availability.busy || 0) + (availability.pause || 0) + (availability.offline || 0)} />
        <ChannelBars channels={data?.channel_distribution || []} className="xl:col-span-2" />
      </section>

      <section className="grid grid-cols-1 gap-6 xl:grid-cols-3">
		<Panel title={supervisor ? 'Volume e fila do time por canal' : 'Volume e fila por canal'} className="xl:col-span-2">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            {(data?.queue_by_channel || []).map((channel) => (
              <div key={channel.name} className="rounded-lg border border-gray-100 p-4 dark:border-gray-800">
                <div className="mb-3 flex items-center justify-between">
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">{channel.name}</p>
                    <p className="text-xs uppercase text-gray-400">{channel.type}</p>
                  </div>
                  <Badge tone={channel.queue_size > 0 ? 'yellow' : 'green'}>{channel.queue_size} na fila</Badge>
                </div>
                <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                  <SmallStat label="Ativas" value={channel.active_count} />
                  <SmallStat label="Espera media" value={formatDuration(channel.avg_wait_seconds)} />
                </div>
              </div>
            ))}
            {data?.queue_by_channel?.length === 0 && <Empty text="Nenhuma conversa ativa por canal." />}
          </div>
        </Panel>

		<Panel title={personal ? 'Minha disponibilidade' : supervisor ? 'Disponibilidade do time' : 'Disponibilidade dos agentes'}>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SmallStat label="Online" value={availability.online} tone="green" />
            <SmallStat label="Ocupados" value={availability.busy} tone="yellow" />
            <SmallStat label="Em pausa" value={availability.pause} tone="blue" />
            <SmallStat label="Offline" value={availability.offline} tone="gray" />
          </div>
        </Panel>
      </section>

      <section className="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <Panel title="Pico de atendimento - ultimas horas" className="xl:col-span-2">
          <div className="flex h-56 items-end gap-2">
            {(data?.peak_hours || []).map((item) => (
              <div key={item.label} className="flex flex-1 flex-col items-center gap-2">
                <div className="w-full rounded-t bg-primary-500/80" style={{ height: `${Math.max(8, (item.total / maxPeak) * 190)}px` }} />
                <span className="text-[10px] text-gray-400">{item.label}</span>
              </div>
            ))}
            {data?.peak_hours?.length === 0 && <Empty text="Sem entradas nas ultimas 12 horas." />}
          </div>
        </Panel>

        <Panel title="Qualidade e abandono">
          <div className="space-y-3">
            <SmallStat label="AHT medio" value={formatDuration(operations?.average_handle_seconds || 0)} tone="blue" />
            <SmallStat label="Taxa de abandono" value={`${formatPercent(operations?.abandonment_rate || 0)}%`} tone={(operations?.abandonment_rate || 0) > 10 ? 'red' : 'green'} />
            <SmallStat label="FCR" value={`${formatPercent(operations?.first_contact_resolution_rate || 0)}%`} tone="green" />
            <SmallStat label="Transferencias" value={operations?.transfers || 0} tone="yellow" />
          </div>
        </Panel>
      </section>

      <section className="grid grid-cols-1 gap-6 xl:grid-cols-2">
		<Panel title={personal ? 'Minha carga de trabalho' : supervisor ? 'Carga de trabalho do time' : 'Carga de trabalho por agente'}>
          <div className="space-y-3">
            {(data?.agents?.workload || []).map((agent) => (
              <div key={agent.id} className="flex items-center gap-3">
                <div className="flex-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="font-medium text-gray-800 dark:text-gray-100">{agent.name}</span>
                    <span className="text-gray-500">{agent.active_chats} chats</span>
                  </div>
                  <div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
                    <div className="h-full rounded-full bg-primary-500" style={{ width: `${Math.min(100, agent.active_chats * 12)}%` }} />
                  </div>
                  <p className="mt-1 text-xs text-gray-400">Inatividade: {formatDuration(agent.idle_seconds)}</p>
                </div>
              </div>
            ))}
          </div>
        </Panel>

        <Panel title="Distribuicao por canal">
          <div className="space-y-3">
            {(data?.channel_distribution || []).map((channel) => {
              const percent = channelTotal ? (channel.total / channelTotal) * 100 : 0
              return (
                <div key={channel.name}>
                  <div className="mb-1 flex justify-between text-sm">
                    <span className="font-medium text-gray-800 dark:text-gray-100">{channel.name}</span>
                    <span className="text-gray-500">{formatPercent(percent)}%</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
                    <div className="h-full rounded-full bg-emerald-500" style={{ width: `${percent}%` }} />
                  </div>
                </div>
              )
            })}
          </div>
        </Panel>
      </section>

      <section className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <Panel title="SLA por canal">
          <div className="space-y-3">
            {(data?.sla_by_channel || []).map((channel) => (
              <div key={channel.name} className="rounded-lg border border-gray-100 p-3 dark:border-gray-800">
                <p className="mb-2 font-medium text-gray-900 dark:text-white">{channel.name}</p>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <SmallStat label="Resposta" value={`${formatPercent(channel.first_response_sla)}%`} tone={channel.first_response_sla < 80 ? 'red' : 'green'} />
                  <SmallStat label="Resolucao" value={`${formatPercent(channel.resolution_sla)}%`} tone={channel.resolution_sla < 80 ? 'red' : 'green'} />
                </div>
              </div>
            ))}
          </div>
        </Panel>

        <Panel title="Alertas de SLA">
          <div className="space-y-3">
            {(data?.sla_alerts || []).map((alert) => (
              <a key={alert.id} href={`/conversations?conversation=${alert.id}`} className="block rounded-lg border border-orange-200 bg-orange-50 p-3 transition hover:bg-orange-100 dark:border-orange-900/60 dark:bg-orange-950/30">
                <div className="flex items-center gap-2">
                  <AlertTriangle size={16} className={alert.seconds_remaining < 0 ? 'text-red-500' : 'text-orange-500'} />
                  <p className="font-medium text-gray-900 dark:text-white">{alert.contact_name}</p>
                  <Badge tone={alert.seconds_remaining < 0 ? 'red' : 'yellow'}>
                    {alert.seconds_remaining < 0 ? 'Estourado' : formatDuration(alert.seconds_remaining)}
                  </Badge>
                </div>
                <p className="mt-1 text-xs text-gray-500">{alert.channel_name} - {alert.status}</p>
              </a>
            ))}
            {data?.sla_alerts?.length === 0 && <Empty text="Nenhum SLA em risco no momento." />}
          </div>
        </Panel>
      </section>
    </div>
  )
}

function RealtimeGauge({ active, waiting, paused, abandoned, className }: { active: number; waiting: number; paused: number; abandoned: number; className?: string }) {
  const total = active + waiting + paused
  const circumference = 2 * Math.PI * 54
  const progress = Math.min(1, total / Math.max(total + 8, 1))
  return <div className={clsx('v-card rounded-2xl p-5', className)}><div className="mb-4 flex items-center justify-between"><div><h2 className="font-semibold text-gray-900 dark:text-white">Atendimentos em tempo real</h2><p className="text-xs text-slate-400">Distribuição dos atendimentos ativos</p></div><Wifi size={18} className="text-cyan-300" /></div><div className="flex items-center gap-5"><div className="relative h-36 w-36 shrink-0"><svg viewBox="0 0 140 140" className="h-full w-full -rotate-90"><circle cx="70" cy="70" r="54" fill="none" stroke="#1b3047" strokeWidth="13" /><circle cx="70" cy="70" r="54" fill="none" stroke="#00dce8" strokeWidth="13" strokeLinecap="round" strokeDasharray={circumference} strokeDashoffset={circumference * (1 - progress)} /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><strong className="text-3xl text-gray-900 dark:text-white">{total}</strong><span className="text-[10px] text-slate-400">ativos</span></div></div><div className="min-w-0 flex-1 space-y-3"><GaugeRow label="Em atendimento" value={active} color="bg-cyan-400" /><GaugeRow label="Aguardando" value={waiting} color="bg-amber-400" /><GaugeRow label="Pausados" value={paused} color="bg-blue-400" /><GaugeRow label="Abandonados" value={Math.round(abandoned)} color="bg-rose-400" /></div></div></div>
}

function GaugeRow({ label, value, color }: { label: string; value: number; color: string }) { return <div className="flex items-center justify-between text-sm"><span className="flex items-center gap-2 text-slate-300"><span className={clsx('h-2 w-2 rounded-full', color)} />{label}</span><strong className="text-gray-900 dark:text-white">{value}</strong></div> }

function VolumeChart({ points, className }: { points: Array<{ label: string; total: number }>; className?: string }) {
  const max = Math.max(...points.map((point) => point.total), 1)
  const bars = points.slice(-12)
  return <div className={clsx('v-card rounded-2xl p-5', className)}><div className="mb-4 flex items-center justify-between"><div><h2 className="font-semibold text-gray-900 dark:text-white">Volume de atendimentos</h2><p className="text-xs text-slate-400">Evolução temporal do período selecionado</p></div><span className="rounded-lg bg-cyan-400/10 px-2.5 py-1 text-xs font-semibold text-cyan-300">Tempo real</span></div><div className="flex h-44 items-end gap-2 border-b border-slate-700/70 pb-5">{bars.length ? bars.map((point) => <div key={point.label} className="group flex min-w-0 flex-1 flex-col items-center gap-2"><div className="relative w-full rounded-t-md bg-gradient-to-t from-cyan-500/70 to-cyan-300" style={{ height: `${Math.max(8, (point.total / max) * 125)}px` }}><span className="absolute -top-6 left-1/2 hidden -translate-x-1/2 rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-white group-hover:block">{point.total}</span></div><span className="truncate text-[10px] text-slate-500">{point.label}</span></div>) : <p className="m-auto text-sm text-slate-500">Sem dados para o período</p>}</div></div>
}

function AvailabilityCard({ availability, total }: { availability: { online: number; busy: number; pause: number; offline: number }; total: number }) {
  return <div className="v-card rounded-2xl p-5"><div className="mb-4 flex items-center justify-between"><h2 className="font-semibold text-gray-900 dark:text-white">Agentes e disponibilidade</h2><Users size={18} className="text-cyan-300" /></div><div className="flex items-center gap-5"><div className="relative h-28 w-28 shrink-0 rounded-full" style={{ background: `conic-gradient(#16d77a 0 ${(availability.online / Math.max(total, 1)) * 100}%, #f5a623 0 ${((availability.online + availability.busy) / Math.max(total, 1)) * 100}%, #168bff 0 ${((availability.online + availability.busy + availability.pause) / Math.max(total, 1)) * 100}%, #334155 0 100%)` }}><div className="absolute inset-3 flex flex-col items-center justify-center rounded-full bg-[#0c1929]"><strong className="text-2xl text-gray-900 dark:text-white">{total}</strong><span className="text-[10px] text-slate-400">agentes</span></div></div><div className="flex-1 space-y-2 text-sm"><GaugeRow label="Online" value={availability.online} color="bg-emerald-400" /><GaugeRow label="Ocupados" value={availability.busy} color="bg-amber-400" /><GaugeRow label="Em pausa" value={availability.pause} color="bg-blue-400" /><GaugeRow label="Offline" value={availability.offline} color="bg-slate-600" /></div></div></div>
}

function ChannelBars({ channels, className }: { channels: Array<{ name: string; type: string; total: number }>; className?: string }) { const total = channels.reduce((sum, channel) => sum + channel.total, 0); return <div className={clsx('v-card rounded-2xl p-5', className)}><div className="mb-4 flex items-center justify-between"><div><h2 className="font-semibold text-gray-900 dark:text-white">Atendimentos por canal</h2><p className="text-xs text-slate-400">Distribuição dos canais configurados</p></div><BarChart3 size={18} className="text-cyan-300" /></div><div className="space-y-4">{channels.length ? channels.map((channel, index) => { const percent = total ? (channel.total / total) * 100 : 0; return <div key={`${channel.name}-${index}`}><div className="mb-1.5 flex justify-between text-sm"><span className="text-slate-200">{channel.name}</span><span className="text-slate-400">{channel.total} <span className="text-xs">({formatPercent(percent)}%)</span></span></div><div className="h-2 overflow-hidden rounded-full bg-slate-800"><div className="h-full rounded-full bg-gradient-to-r from-cyan-500 to-blue-500" style={{ width: `${percent}%` }} /></div></div> }) : <p className="py-8 text-center text-sm text-slate-500">Nenhum canal com dados no período</p>}</div></div> }

function MetricCard({ title, value, icon, tone, delta, href }: { title: string; value: ReactNode; icon: ReactNode; tone: Tone; delta?: string; href?: string }) {
  return (
    <div className="v-card v-kpi rounded-2xl p-5">
      <div className="mb-4 flex items-center justify-between">
        <div className={clsx('flex h-10 w-10 items-center justify-center rounded-lg', toneClass(tone))}>{icon}</div>
        {delta && <span className="rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">{delta}</span>}
      </div>
      <p className="text-2xl font-bold text-gray-900 dark:text-white">{value}</p>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{title}</p>
    </div>
  )
}

function Panel({ title, children, className }: { title: string; children: ReactNode; className?: string }) {
  return (
    <div className={clsx('v-card rounded-2xl p-5', className)}>
      <div className="mb-4 flex items-center gap-2">
        <BarChart3 size={17} className="text-primary-500" />
        <h2 className="font-semibold text-gray-900 dark:text-white">{title}</h2>
      </div>
      {children}
    </div>
  )
}

type Tone = 'blue' | 'cyan' | 'green' | 'yellow' | 'purple' | 'red' | 'gray'

function SmallStat({ label, value, tone = 'gray' }: { label: string; value: ReactNode; tone?: Tone }) {
  return (
    <div className={clsx('rounded-lg p-3', toneSoftClass(tone))}>
      <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
      <p className="mt-1 text-lg font-bold text-gray-900 dark:text-white">{value}</p>
    </div>
  )
}

function Badge({ children, tone = 'gray' }: { children: ReactNode; tone?: Tone }) {
  return <span className={clsx('ml-auto rounded-full px-2 py-1 text-xs font-semibold', toneSoftClass(tone))}>{children}</span>
}

function Empty({ text }: { text: string }) {
  return <p className="py-6 text-center text-sm text-gray-400">{text}</p>
}

function toneClass(tone: Tone) {
  const map = {
    blue: 'bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-300',
    cyan: 'bg-cyan-400/15 text-cyan-300',
    green: 'bg-green-100 text-green-600 dark:bg-green-950 dark:text-green-300',
    yellow: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300',
    purple: 'bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-300',
    red: 'bg-red-100 text-red-600 dark:bg-red-950 dark:text-red-300',
    gray: 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300',
  }
  return map[tone]
}

function toneSoftClass(tone: Tone) {
  const map = {
    blue: 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300',
    cyan: 'bg-cyan-400/10 text-cyan-200',
    green: 'bg-green-50 text-green-700 dark:bg-green-950/40 dark:text-green-300',
    yellow: 'bg-yellow-50 text-yellow-800 dark:bg-yellow-950/40 dark:text-yellow-300',
    purple: 'bg-purple-50 text-purple-700 dark:bg-purple-950/40 dark:text-purple-300',
    red: 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300',
    gray: 'bg-gray-50 text-gray-700 dark:bg-gray-800/70 dark:text-gray-300',
  }
  return map[tone]
}

function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds) || seconds <= 0) return '0s'
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

function formatPercent(value: number) {
  if (!Number.isFinite(value)) return '0'
  return value.toFixed(value >= 10 ? 0 : 1)
}

function comparisonLabel(value?: number, lowerIsBetter = false) {
  if (value == null || !Number.isFinite(value)) return 'vs anterior: 0%'
  const direction = value > 0 ? '↑' : value < 0 ? '↓' : '→'
  const normalized = Math.abs(value)
  const good = lowerIsBetter ? value <= 0 : value >= 0
  return `${direction} ${formatPercent(normalized)}% ${good ? 'ok' : 'alerta'}`
}

function slaAverage(items: Array<{ first_response_sla: number; resolution_sla: number }>) {
  if (!items.length) return 0
  return items.reduce((total, item) => total + ((item.first_response_sla || 0) + (item.resolution_sla || 0)) / 2, 0) / items.length
}
