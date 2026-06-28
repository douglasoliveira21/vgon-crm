// EvoCRM Pro - TypeScript Types

export interface User {
  id: string
  company_id: string
  name: string
  email: string
  role_slug: string
  role_name: string
  avatar_url?: string
  phone?: string
  is_active: boolean
  is_online: boolean
  last_seen_at?: string
}

export interface Company {
  id: string
  name: string
  slug: string
  plan: string
  is_active: boolean
}

export interface Channel {
  id: string
  name: string
  type: 'whatsapp' | 'instagram' | 'facebook' | 'telegram' | 'webchat' | 'email'
  status: 'connected' | 'disconnected' | 'connecting' | 'error'
  settings?: Record<string, any>
  is_active: boolean
}

export interface WhatsAppInstance {
  id: string
  company_id: string
  channel_id?: string
  instance_name: string
  instance_id?: string
  token?: string
  status: 'disconnected' | 'qr_code' | 'connecting' | 'connected' | 'error'
  qrcode?: string
  phone_number?: string
  webhook_url?: string
  connected_at?: string
  created_at: string
}

export interface Contact {
  id: string
  company_id: string
  name?: string
  phone?: string
  email?: string
  company_name?: string
  position?: string
  city?: string
  state?: string
  origin?: string
  avatar_url?: string
  notes?: string
  assigned_to?: string
  is_opted_out: boolean
  tags?: Tag[]
  created_at: string
  updated_at: string
}

export interface Tag {
  id: string
  name: string
  color: string
}

export interface Conversation {
  id: string
  company_id: string
  contact_id: string
  channel_id?: string
  assigned_to?: string
  team_id?: string
  status: 'open' | 'pending' | 'in_progress' | 'resolved' | 'waiting_client'
  priority: 'low' | 'normal' | 'high' | 'urgent'
  subject?: string
  last_message_at?: string
  last_message_preview?: string
  unread_count: number
  contact_name?: string
  contact_phone?: string
  contact_avatar_url?: string
  assigned_to_name?: string
  channel_name?: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  conversation_id: string
  company_id: string
  sender_type: 'contact' | 'user' | 'system' | 'bot'
  sender_id?: string
  content?: string
  message_type: 'text' | 'image' | 'audio' | 'video' | 'document' | 'sticker' | 'location'
  media_url?: string
  media_mime_type?: string
  media_filename?: string
  external_id?: string
  status: 'sent' | 'delivered' | 'read' | 'failed'
  is_private: boolean
  sender_name?: string
  created_at: string
}

export interface Team {
  id: string
  name: string
  description?: string
  distribution_rule: 'manual' | 'round-robin' | 'least-busy' | 'first-available' | 'keyword' | 'schedule'
  is_active: boolean
  member_count: number
}

export interface Funnel {
  id: string
  name: string
  description?: string
  is_default: boolean
  is_active: boolean
  stages: FunnelStage[]
}

export interface FunnelStage {
  id: string
  funnel_id: string
  name: string
  color: string
  position: number
  is_won: boolean
  is_lost: boolean
  deal_count: number
  deal_value: number
}

export interface Deal {
  id: string
  funnel_id: string
  stage_id: string
  contact_id?: string
  conversation_id?: string
  assigned_to?: string
  title: string
  value: number
  expected_close_date?: string
  origin?: string
  loss_reason?: string
  status: 'open' | 'won' | 'lost'
  contact_name?: string
  assigned_to_name?: string
  stage_name?: string
}

export interface BotFlow {
  id: string
  name: string
  description?: string
  trigger_type: 'new_conversation' | 'keyword' | 'off_hours' | 'no_response' | 'transfer' | 'funnel_stage' | 'tag_added'
  trigger_value?: string
  channel_id?: string
  is_active: boolean
  nodes: BotNode[]
  edges: BotEdge[]
}

export interface BotNode {
  id: string
  type: 'send_message' | 'send_media' | 'ask_question' | 'transfer_team' | 'transfer_agent' | 'add_tag' | 'change_stage' | 'call_webhook' | 'condition' | 'delay'
  position: { x: number; y: number }
  data: Record<string, any>
}

export interface BotEdge {
  id: string
  source: string
  target: string
  label?: string
}

export interface Campaign {
  id: string
  name: string
  description?: string
  message_content?: string
  message_type: 'text' | 'image' | 'document' | 'audio'
  media_url?: string
  status: 'draft' | 'scheduled' | 'sending' | 'completed' | 'paused' | 'cancelled'
  scheduled_at?: string
  send_speed: number
  total_contacts: number
  sent_count: number
  delivered_count: number
  read_count: number
  replied_count: number
  failed_count: number
}

export interface Call {
  id: string
  contact_id?: string
  user_id?: string
  direction: 'inbound' | 'outbound'
  status: 'initiated' | 'ringing' | 'answered' | 'completed' | 'missed' | 'declined' | 'no_answer'
  duration: number
  started_at?: string
  ended_at?: string
  contact_name?: string
  user_name?: string
}

export interface Widget {
  id: string
  name: string
  primary_color: string
  greeting_message: string
  position: 'bottom-right' | 'bottom-left'
  is_active: boolean
}

export interface Announcement {
  id: string
  title: string
  content: string
  priority: 'low' | 'normal' | 'high' | 'urgent'
  is_pinned: boolean
  author_name: string
  published_at: string
  is_read: boolean
}

export interface DashboardStats {
  open_conversations: number
  pending_conversations: number
  my_conversations: number
  active_deals: number
  total_deals_value: number
  avg_response_time: number
  connected_channels: number
}

// API Response types
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}

export interface ApiError {
  error: string
  message?: string
}
