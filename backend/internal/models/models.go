package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Company represents a tenant/empresa
type Company struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Document    *string         `json:"document,omitempty"`
	Email       *string         `json:"email,omitempty"`
	Phone       *string         `json:"phone,omitempty"`
	LogoURL     *string         `json:"logo_url,omitempty"`
	Plan        string          `json:"plan"`
	MaxUsers    int             `json:"max_users"`
	MaxChannels int             `json:"max_channels"`
	IsActive    bool            `json:"is_active"`
	Settings    json.RawMessage `json:"settings"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// User represents a system user
type User struct {
	ID                 string     `json:"id"`
	CompanyID          string     `json:"company_id"`
	RoleID             *string    `json:"role_id,omitempty"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	PasswordHash       string     `json:"-"`
	AvatarURL          *string    `json:"avatar_url,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	IsActive           bool       `json:"is_active"`
	IsOnline           bool       `json:"is_online"`
	AvailabilityStatus string     `json:"availability_status"`
	IsSuperAdmin       bool       `json:"is_super_admin"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
	RoleName           string     `json:"role_name,omitempty"`
	RoleSlug           string     `json:"role_slug,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Role represents a user role with permissions
type Role struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description,omitempty"`
	Permissions json.RawMessage `json:"permissions"`
	IsSystem    bool            `json:"is_system"`
	CompanyID   *string         `json:"company_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Team represents a team
type Team struct {
	ID               string          `json:"id"`
	CompanyID        string          `json:"company_id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	DistributionRule string          `json:"distribution_rule"`
	BusinessHours    json.RawMessage `json:"business_hours"`
	IsActive         bool            `json:"is_active"`
	MemberCount      int             `json:"member_count,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Channel represents a communication channel
type Channel struct {
	ID            string          `json:"id"`
	CompanyID     string          `json:"company_id"`
	TeamID        *string         `json:"team_id,omitempty"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	Settings      json.RawMessage `json:"settings"`
	BusinessHours json.RawMessage `json:"business_hours"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// WhatsAppInstance represents an Evolution API instance
type WhatsAppInstance struct {
	ID           string     `json:"id"`
	CompanyID    string     `json:"company_id"`
	ChannelID    *string    `json:"channel_id,omitempty"`
	InstanceName string     `json:"instance_name"`
	InstanceID   *string    `json:"instance_id,omitempty"`
	Token        *string    `json:"token,omitempty"`
	Status       string     `json:"status"`
	QRCode       *string    `json:"qrcode,omitempty"`
	PhoneNumber  *string    `json:"phone_number,omitempty"`
	WebhookURL   *string    `json:"webhook_url,omitempty"`
	ConnectedAt  *time.Time `json:"connected_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Contact represents a customer contact
type Contact struct {
	ID          string    `json:"id"`
	CompanyID   string    `json:"company_id"`
	Name        *string   `json:"name,omitempty"`
	Phone       *string   `json:"phone,omitempty"`
	Email       *string   `json:"email,omitempty"`
	CompanyName *string   `json:"company_name,omitempty"`
	Position    *string   `json:"position,omitempty"`
	City        *string   `json:"city,omitempty"`
	State       *string   `json:"state,omitempty"`
	Origin      *string   `json:"origin,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	AssignedTo  *string   `json:"assigned_to,omitempty"`
	IsOptedOut  bool      `json:"is_opted_out"`
	Tags        []Tag     `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Tag represents a label/tag
type Tag struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID                 string     `json:"id"`
	CompanyID          string     `json:"company_id"`
	ContactID          string     `json:"contact_id"`
	ChannelID          *string    `json:"channel_id,omitempty"`
	AssignedTo         *string    `json:"assigned_to,omitempty"`
	TeamID             *string    `json:"team_id,omitempty"`
	Status             string     `json:"status"`
	Priority           string     `json:"priority"`
	Subject            *string    `json:"subject,omitempty"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`
	LastMessagePreview *string    `json:"last_message_preview,omitempty"`
	UnreadCount        int        `json:"unread_count"`
	ContactName        *string    `json:"contact_name,omitempty"`
	ContactPhone       *string    `json:"contact_phone,omitempty"`
	ContactAvatarURL   *string    `json:"contact_avatar_url,omitempty"`
	AssignedToName     *string    `json:"assigned_to_name,omitempty"`
	ChannelName        *string    `json:"channel_name,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Message represents a chat message
type Message struct {
	ID              string          `json:"id"`
	ConversationID  string          `json:"conversation_id"`
	CompanyID       string          `json:"company_id"`
	SenderType      string          `json:"sender_type"` // contact, user, system, bot
	SenderID        *string         `json:"sender_id,omitempty"`
	Content         *string         `json:"content,omitempty"`
	MessageType     string          `json:"message_type"` // text, image, audio, video, document
	MediaURL        *string         `json:"media_url,omitempty"`
	MediaMimeType   *string         `json:"media_mime_type,omitempty"`
	MediaFilename   *string         `json:"media_filename,omitempty"`
	ExternalID      *string         `json:"external_id,omitempty"`
	Status          string          `json:"status"`
	IsPrivate       bool            `json:"is_private"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	SenderName      *string         `json:"sender_name,omitempty"`
	SenderAvatarURL *string         `json:"sender_avatar_url,omitempty"`
	ReplyToContent  *string         `json:"reply_to_content,omitempty"`
	ReplyToSender   *string         `json:"reply_to_sender,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// Funnel represents a sales funnel
type Funnel struct {
	ID          string        `json:"id"`
	CompanyID   string        `json:"company_id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	IsDefault   bool          `json:"is_default"`
	IsActive    bool          `json:"is_active"`
	Stages      []FunnelStage `json:"stages,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// FunnelStage represents a stage in a funnel
type FunnelStage struct {
	ID        string  `json:"id"`
	FunnelID  string  `json:"funnel_id"`
	Name      string  `json:"name"`
	Color     string  `json:"color"`
	Position  int     `json:"position"`
	IsWon     bool    `json:"is_won"`
	IsLost    bool    `json:"is_lost"`
	DealCount int     `json:"deal_count,omitempty"`
	DealValue float64 `json:"deal_value,omitempty"`
}

// Deal represents a sales opportunity
type Deal struct {
	ID                string        `json:"id"`
	CompanyID         string        `json:"company_id"`
	FunnelID          string        `json:"funnel_id"`
	StageID           string        `json:"stage_id"`
	ContactID         *string       `json:"contact_id,omitempty"`
	ConversationID    *string       `json:"conversation_id,omitempty"`
	AssignedTo        *string       `json:"assigned_to,omitempty"`
	Title             string        `json:"title"`
	Value             float64       `json:"value"`
	ExpectedCloseDate *sql.NullTime `json:"expected_close_date,omitempty"`
	Origin            *string       `json:"origin,omitempty"`
	LossReason        *string       `json:"loss_reason,omitempty"`
	Notes             *string       `json:"notes,omitempty"`
	Status            string        `json:"status"`
	ContactName       *string       `json:"contact_name,omitempty"`
	AssignedToName    *string       `json:"assigned_to_name,omitempty"`
	StageName         *string       `json:"stage_name,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// BotFlow represents an automation flow
type BotFlow struct {
	ID           string          `json:"id"`
	CompanyID    string          `json:"company_id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	TriggerType  string          `json:"trigger_type"`
	TriggerValue *string         `json:"trigger_value,omitempty"`
	ChannelID    *string         `json:"channel_id,omitempty"`
	IsActive     bool            `json:"is_active"`
	Nodes        json.RawMessage `json:"nodes"`
	Edges        json.RawMessage `json:"edges"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Campaign represents a mass marketing campaign
type Campaign struct {
	ID             string     `json:"id"`
	CompanyID      string     `json:"company_id"`
	ChannelID      *string    `json:"channel_id,omitempty"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	MessageContent *string    `json:"message_content,omitempty"`
	MessageType    string     `json:"message_type"`
	MediaURL       *string    `json:"media_url,omitempty"`
	Status         string     `json:"status"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	SendSpeed      int        `json:"send_speed"`
	TotalContacts  int        `json:"total_contacts"`
	SentCount      int        `json:"sent_count"`
	DeliveredCount int        `json:"delivered_count"`
	ReadCount      int        `json:"read_count"`
	RepliedCount   int        `json:"replied_count"`
	FailedCount    int        `json:"failed_count"`
	CreatedBy      *string    `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Call represents an audio call
type Call struct {
	ID             string     `json:"id"`
	CompanyID      string     `json:"company_id"`
	ContactID      *string    `json:"contact_id,omitempty"`
	UserID         *string    `json:"user_id,omitempty"`
	ConversationID *string    `json:"conversation_id,omitempty"`
	ChannelID      *string    `json:"channel_id,omitempty"`
	Direction      string     `json:"direction"`
	Status         string     `json:"status"`
	Duration       int        `json:"duration"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	RecordingURL   *string    `json:"recording_url,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	ContactName    *string    `json:"contact_name,omitempty"`
	UserName       *string    `json:"user_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Widget represents a website widget
type Widget struct {
	ID                string          `json:"id"`
	CompanyID         string          `json:"company_id"`
	Name              string          `json:"name"`
	ChannelID         *string         `json:"channel_id,omitempty"`
	TeamID            *string         `json:"team_id,omitempty"`
	PrimaryColor      string          `json:"primary_color"`
	GreetingMessage   string          `json:"greeting_message"`
	ChannelsDisplayed json.RawMessage `json:"channels_displayed"`
	Position          string          `json:"position"`
	ShowPreChatForm   bool            `json:"show_pre_chat_form"`
	PreChatFields     json.RawMessage `json:"pre_chat_fields"`
	BusinessHours     json.RawMessage `json:"business_hours"`
	IsActive          bool            `json:"is_active"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// InternalAnnouncement represents an internal announcement
type InternalAnnouncement struct {
	ID           string     `json:"id"`
	CompanyID    string     `json:"company_id"`
	AuthorID     string     `json:"author_id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Priority     string     `json:"priority"`
	TargetType   string     `json:"target_type"`
	TargetTeamID *string    `json:"target_team_id,omitempty"`
	IsPinned     bool       `json:"is_pinned"`
	PublishedAt  time.Time  `json:"published_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	AuthorName   string     `json:"author_name,omitempty"`
	IsRead       bool       `json:"is_read,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	OpenConversations    int     `json:"open_conversations"`
	PendingConversations int     `json:"pending_conversations"`
	MyConversations      int     `json:"my_conversations"`
	ActiveDeals          int     `json:"active_deals"`
	TotalDealsValue      float64 `json:"total_deals_value"`
	AvgResponseTime      float64 `json:"avg_response_time"`
	ConnectedChannels    int     `json:"connected_channels"`
}
