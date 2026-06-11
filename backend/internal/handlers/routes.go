package handlers

import (
	"github.com/evocrm/backend/internal/middleware"
	"github.com/evocrm/backend/internal/services"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, svc *services.Container, wsHub *websocket.Hub) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "EvoCRM Pro"})
	})

	// Static files - uploads
	app.Static("/uploads", "/app/uploads", fiber.Static{
		Browse: false,
	})

	// Media proxy (public - serves cached media)
	app.Get("/media/:messageId", MediaProxy(svc))

	api := app.Group("/api")

	// ============================================
	// AUTH Routes (public)
	// ============================================
	auth := api.Group("/auth")
	auth.Post("/login", AuthLogin(svc))
	auth.Post("/register", AuthRegister(svc))
	auth.Post("/refresh", AuthRefresh(svc))

	// ============================================
	// Webhook Routes (public - from Evolution API) - MUST be before protected routes
	// ============================================
	api.Post("/webhooks/evolution/:instanceName", HandleEvolutionWebhook(svc))

	// ============================================
	// Widget Public Routes
	// ============================================
	api.Get("/widget/:id/config", GetWidgetPublicConfig(svc))

	// ============================================
	// Protected Routes
	// ============================================
	protected := api.Group("", middleware.AuthMiddleware(svc.Config))

	// Dashboard
	protected.Get("/dashboard", GetDashboard(svc))

	// User profile
	protected.Get("/me", GetCurrentUser(svc))

	// WhatsApp Instances
	whatsapp := protected.Group("/whatsapp")
	whatsapp.Get("/instances", GetWhatsAppInstances(svc))
	whatsapp.Post("/instances", CreateWhatsAppInstance(svc))
	whatsapp.Get("/instances/:id/qrcode", GetWhatsAppQRCode(svc))
	whatsapp.Get("/instances/:id/status", GetWhatsAppStatus(svc))
	whatsapp.Post("/instances/:id/disconnect", DisconnectWhatsAppInstance(svc))
	whatsapp.Delete("/instances/:id", DeleteWhatsAppInstance(svc))
	whatsapp.Post("/instances/:id/sync-contacts", SyncWhatsAppContacts(svc))
	whatsapp.Post("/instances/:id/sync-photos", SyncWhatsAppPhotos(svc))

	// Conversations
	conversations := protected.Group("/conversations")
	conversations.Get("/", GetConversations(svc))
	conversations.Get("/assigned-to-me", GetMyConversations(svc))
	conversations.Post("/start", StartConversation(svc))
	conversations.Get("/:id", GetConversation(svc))
	conversations.Post("/:id/assign", AssignConversation(svc))
	conversations.Post("/:id/transfer", TransferConversation(svc))
	conversations.Post("/:id/close", CloseConversation(svc))
	conversations.Post("/:id/reopen", ReopenConversation(svc))
	conversations.Get("/:id/messages", GetConversationMessages(svc))
	conversations.Post("/:id/messages/text", SendTextMessage(svc))
	conversations.Post("/:id/messages/media", SendMediaMessage(svc))
	conversations.Post("/:id/messages/audio", SendAudioMessage(svc))
	conversations.Delete("/:id/messages/:msgId", DeleteMessage(svc))

	// Contacts
	contacts := protected.Group("/contacts")
	contacts.Get("/", GetContacts(svc))
	contacts.Get("/:id", GetContact(svc))
	contacts.Post("/", CreateContact(svc))
	contacts.Put("/:id", UpdateContact(svc))
	contacts.Delete("/:id", DeleteContact(svc))
	contacts.Post("/:id/tags", AddContactTag(svc))
	contacts.Delete("/:id/tags/:tagId", RemoveContactTag(svc))

	// Teams
	teams := protected.Group("/teams")
	teams.Get("/", GetTeams(svc))
	teams.Post("/", CreateTeam(svc))
	teams.Put("/:id", UpdateTeam(svc))
	teams.Delete("/:id", DeleteTeam(svc))
	teams.Post("/:id/members", AddTeamMember(svc))
	teams.Delete("/:id/members/:userId", RemoveTeamMember(svc))

	// Channels
	channels := protected.Group("/channels")
	channels.Get("/", GetChannels(svc))

	// Funnels
	funnels := protected.Group("/funnels")
	funnels.Get("/", GetFunnels(svc))
	funnels.Post("/", CreateFunnel(svc))
	funnels.Put("/:id", UpdateFunnel(svc))
	funnels.Delete("/:id", DeleteFunnel(svc))

	// Deals
	deals := protected.Group("/deals")
	deals.Get("/", GetDeals(svc))
	deals.Post("/", CreateDeal(svc))
	deals.Put("/:id", UpdateDeal(svc))
	deals.Put("/:id/stage", MoveDealStage(svc))

	// Bot Flows
	bots := protected.Group("/bot-flows")
	bots.Get("/", GetBotFlows(svc))
	bots.Post("/", CreateBotFlow(svc))
	bots.Put("/:id", UpdateBotFlow(svc))
	bots.Delete("/:id", DeleteBotFlow(svc))

	// Campaigns
	campaigns := protected.Group("/campaigns")
	campaigns.Get("/", GetCampaigns(svc))
	campaigns.Post("/", CreateCampaign(svc))
	campaigns.Put("/:id", UpdateCampaign(svc))
	campaigns.Post("/:id/start", StartCampaign(svc))
	campaigns.Post("/:id/pause", PauseCampaign(svc))

	// Metrics
	protected.Get("/metrics", GetMetrics(svc))

	// Announcements
	announcements := protected.Group("/announcements")
	announcements.Get("/", GetAnnouncements(svc))
	announcements.Post("/", CreateAnnouncement(svc))
	announcements.Post("/:id/read", MarkAnnouncementRead(svc))

	// Widgets
	widgets := protected.Group("/widgets")
	widgets.Get("/", GetWidgets(svc))
	widgets.Post("/", CreateWidget(svc))
	widgets.Put("/:id", UpdateWidget(svc))

	// Tags
	tags := protected.Group("/tags")
	tags.Get("/", GetTags(svc))
	tags.Post("/", CreateTag(svc))
	tags.Delete("/:id", DeleteTag(svc))

	// Quick Replies
	quickReplies := protected.Group("/quick-replies")
	quickReplies.Get("/", GetQuickReplies(svc))
	quickReplies.Post("/", CreateQuickReply(svc))

	// Users
	users := protected.Group("/users")
	users.Get("/", GetUsers(svc))
	users.Post("/", CreateUser(svc))

	// Telephony
	telephony := protected.Group("/telephony")
	telephony.Post("/provider", SaveTelephonyProvider(svc))
	telephony.Get("/provider", GetTelephonyProvider(svc))

	// ============================================
	// WebSocket
	// ============================================
	app.Get("/ws", WebSocketHandler(wsHub, svc.Config))
}
