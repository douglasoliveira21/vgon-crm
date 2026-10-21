export type Presence = 'online' | 'busy' | 'offline'

interface PresenceSource {
  is_online?: boolean
  availability_status?: string
}

// One rule for the whole app. "Offline" is not a setting: it means the user
// has no open connection. "Busy" is the only thing a user chooses, and only
// means something while they are connected.
export function presenceOf(user?: PresenceSource | null): Presence {
  if (!user?.is_online) return 'offline'
  return user.availability_status === 'busy' ? 'busy' : 'online'
}

export const presenceMeta: Record<Presence, { label: string; dot: string; hint: string }> = {
  online: { label: 'Online', dot: 'bg-green-500', hint: 'Disponível' },
  busy: { label: 'Ocupado', dot: 'bg-amber-500', hint: 'Conectado, mas pediu para não receber novas conversas' },
  offline: { label: 'Offline', dot: 'bg-gray-400', hint: 'Não está com o sistema aberto' },
}

// Available people first, so the best choice for a transfer is on top.
const presenceOrder: Record<Presence, number> = { online: 0, busy: 1, offline: 2 }

export function sortByPresence<T extends PresenceSource & { name: string }>(users: T[]): T[] {
  return [...users].sort(
    (a, b) => presenceOrder[presenceOf(a)] - presenceOrder[presenceOf(b)] || a.name.localeCompare(b.name)
  )
}
