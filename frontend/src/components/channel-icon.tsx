import {
  Bot,
  Globe2,
  Inbox,
  Mail,
  Megaphone,
  MessageCircle,
  Phone,
  ShieldCheck,
  type LucideIcon,
} from 'lucide-react'
import { clsx } from 'clsx'

export type ChannelIconType =
  | 'whatsapp'
  | 'email'
  | 'webchat'
  | 'website'
  | 'phone'
  | 'campaign'
  | 'automation'
  | 'security'
  | 'inbox'

const iconMap: Record<Exclude<ChannelIconType, 'whatsapp'>, LucideIcon> = {
  email: Mail,
  webchat: MessageCircle,
  website: Globe2,
  phone: Phone,
  campaign: Megaphone,
  automation: Bot,
  security: ShieldCheck,
  inbox: Inbox,
}

const colorMap: Record<ChannelIconType, string> = {
  whatsapp: 'text-[#25D366]',
  email: 'text-blue-600',
  webchat: 'text-violet-600',
  website: 'text-violet-600',
  phone: 'text-sky-600',
  campaign: 'text-amber-600',
  automation: 'text-indigo-600',
  security: 'text-emerald-600',
  inbox: 'text-gray-600',
}

export function ChannelIcon({
  type,
  size = 20,
  className,
  title,
}: {
  type: ChannelIconType | string
  size?: number
  className?: string
  title?: string
}) {
  const normalized = type === 'site' || type === 'widget' ? 'webchat' : type
  const safeType = normalized in colorMap ? normalized as ChannelIconType : 'inbox'

  if (safeType === 'whatsapp') {
    return (
      <svg
        viewBox="0 0 24 24"
        width={size}
        height={size}
        role={title ? 'img' : undefined}
        aria-hidden={title ? undefined : true}
        aria-label={title}
        className={clsx('shrink-0 fill-current', colorMap.whatsapp, className)}
      >
        <path d="M12.04 2a9.84 9.84 0 0 0-8.5 14.78L2 22l5.36-1.5A9.97 9.97 0 0 0 12.04 22 10 10 0 0 0 12.04 2Zm0 18.16a8.12 8.12 0 0 1-4.14-1.13l-.3-.18-3.18.89.85-3.1-.2-.32a8.14 8.14 0 1 1 6.97 3.84Zm4.46-6.1c-.24-.12-1.44-.71-1.66-.79-.22-.08-.38-.12-.54.12-.16.24-.62.79-.76.95-.14.16-.28.18-.52.06-.24-.12-1.03-.38-1.96-1.21a7.26 7.26 0 0 1-1.36-1.69c-.14-.24-.01-.37.11-.49.11-.11.24-.28.36-.42.12-.14.16-.24.24-.4.08-.16.04-.3-.02-.42-.06-.12-.54-1.3-.74-1.78-.2-.47-.4-.41-.54-.42h-.46c-.16 0-.42.06-.64.3-.22.24-.84.82-.84 2s.86 2.32.98 2.48c.12.16 1.69 2.58 4.1 3.62.57.25 1.02.39 1.37.5.58.18 1.1.16 1.51.1.46-.07 1.44-.59 1.64-1.16.2-.57.2-1.06.14-1.16-.06-.1-.22-.16-.46-.28Z" />
      </svg>
    )
  }

  const Icon = iconMap[safeType]
  return <Icon size={size} aria-label={title} className={clsx('shrink-0', colorMap[safeType], className)} />
}

export function channelSurface(type: string) {
  switch (type) {
    case 'whatsapp': return 'bg-green-50 dark:bg-green-950/30'
    case 'email': return 'bg-blue-50 dark:bg-blue-950/30'
    case 'webchat':
    case 'website':
    case 'site':
    case 'widget': return 'bg-violet-50 dark:bg-violet-950/30'
    default: return 'bg-gray-100 dark:bg-gray-800'
  }
}
