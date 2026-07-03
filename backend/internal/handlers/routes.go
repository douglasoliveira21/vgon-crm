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
		return c.JSON(fiber.Map{"status": "ok", "service": "crmvgon"})
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
	auth.Post("/forgot-password", AuthForgotPassword(svc))
	auth.Post("/reset-password", AuthResetPassword(svc))

	// ============================================
	// Webhook Routes (public - from Evolution API) - MUST be before protected routes
	// ============================================
	api.Post("/webhooks/evolution/:instanceName", HandleEvolutionWebhook(svc))

	// ============================================
	// Widget Public Routes (CORS open - widget can be embedded on any site)
	// ============================================
	widgetCORS := func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}
		return c.Next()
	}
	api.Options("/widget/:id/config", widgetCORS)
	api.Options("/widget/:id/message", widgetCORS)
	api.Options("/widget/:id/messages", widgetCORS)
	api.Options("/widget/:id/close", widgetCORS)
	api.Get("/widget/:id/config", widgetCORS, GetWidgetPublicConfig(svc))
	api.Post("/widget/:id/message", widgetCORS, SendWidgetMessage(svc))
	api.Get("/widget/:id/messages", widgetCORS, GetWidgetMessages(svc))
	api.Post("/widget/:id/close", widgetCORS, CloseWidgetConversation(svc))
	app.Get("/widget/:id/embed.js", GetWidgetEmbedScript(svc))
	api.Get("/oauth/email/:provider/callback", EmailOAuthCallback(svc))

	// ============================================
	// Protected Routes
	// ============================================
	protected := api.Group("", middleware.AuthMiddleware(svc.Config))

	// Dashboard
	protected.Get("/dashboard", GetDashboard(svc))

	// User profile
	protected.Get("/me", GetCurrentUser(svc))
	protected.Put("/me", UpdateCurrentUser(svc))
	protected.Put("/me/status", UpdateCurrentUserStatus(svc))
	protected.Put("/me/password", UpdateCurrentUserPassword(svc))
	protected.Post("/me/avatar", UploadCurrentUserAvatar(svc))

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
	conversations.Post("/:id/unassign", UnassignConversation(svc))
	conversations.Post("/:id/transfer", TransferConversation(svc))
	conversations.Put("/:id/customer-company", LinkConversationCustomerCompany(svc))
	conversations.Post("/:id/close", CloseConversation(svc))
	conversations.Post("/:id/reopen", ReopenConversation(svc))
	conversations.Post("/:id/read", MarkConversationRead(svc))
	conversations.Post("/:id/unread", MarkConversationUnread(svc))
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

	// Customer companies
	customerCompanies := protected.Group("/customer-companies")
	customerCompanies.Get("/", ListCustomerCompanies(svc))
	customerCompanies.Get("/lookup/:cnpj", LookupCNPJ(svc))
	customerCompanies.Get("/sla-report.csv", ExportCustomerCompanySLAReport(svc))
	customerCompanies.Post("/", CreateCustomerCompany(svc))
	customerCompanies.Put("/:id", UpdateCustomerCompany(svc))
	customerCompanies.Delete("/:id", DeleteCustomerCompany(svc))

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
	channels.Get("/email/oauth/:provider/start", StartEmailOAuth(svc))
	channels.Post("/email", CreateEmailChannel(svc))
	channels.Put("/email/:id", UpdateEmailChannel(svc))
	channels.Delete("/email/:id", DeleteEmailChannel(svc))
	channels.Post("/email/:id/sync", SyncEmailChannel(svc))

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
	bots.Get("/:id", GetBotFlow(svc))
	bots.Post("/", CreateBotFlow(svc))
	bots.Put("/:id", UpdateBotFlow(svc))
	bots.Delete("/:id", DeleteBotFlow(svc))

	// Campaigns
	campaigns := protected.Group("/campaigns")
	campaigns.Get("/", GetCampaigns(svc))
	campaigns.Post("/", CreateCampaign(svc))
	campaigns.Put("/:id", UpdateCampaign(svc))
	campaigns.Delete("/:id", DeleteCampaign(svc))
	campaigns.Post("/:id/start", StartCampaign(svc))
	campaigns.Post("/:id/pause", PauseCampaign(svc))

	// Metrics
	protected.Get("/metrics", GetMetrics(svc))
	protected.Get("/metrics/attendance", GetAttendanceMetrics(svc))

	// GLPI
	glpi := protected.Group("/glpi")
	glpi.Post("/tickets", GLPICreateTicket(svc))
	glpi.Get("/tickets/:id", GLPIGetTicket(svc))
	glpi.Get("/entities", GLPIGetEntities(svc))
	glpi.Get("/entities/:id", GLPIGetEntity(svc))
	glpi.Post("/flow/start", GLPIStartFlow(svc))

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
	quickReplies.Put("/:id", UpdateQuickReply(svc))
	quickReplies.Delete("/:id", DeleteQuickReply(svc))

	// Users
	users := protected.Group("/users")
	users.Get("/", GetUsers(svc))
	users.Post("/", CreateUser(svc))

	// ============================================
	// Super Admin Routes
	// ============================================
	admin := api.Group("/admin", middleware.AuthMiddleware(svc.Config), middleware.SuperAdminMiddleware(svc.DB))
	admin.Get("/tenants", GetTenants(svc))
	admin.Post("/tenants", CreateTenant(svc))
	admin.Get("/tenants/:id", GetTenant(svc))
	admin.Put("/tenants/:id", UpdateTenant(svc))
	admin.Delete("/tenants/:id", DeleteTenant(svc))
	admin.Get("/stats", GetAdminStats(svc))

	// Admin user management
	admin.Get("/tenants/:id/users", AdminGetTenantUsers(svc))
	admin.Post("/tenants/:id/users", AdminCreateTenantUser(svc))
	admin.Put("/users/:userId/password", AdminResetUserPassword(svc))
	admin.Put("/users/:userId", AdminUpdateUser(svc))
	admin.Delete("/users/:userId", AdminDeleteUser(svc))

	// ============================================
	// WebSocket
	// ============================================
	app.Get("/ws", WebSocketHandler(wsHub, svc.Config, svc))
	app.Get("/ws/widget", WidgetWebSocketHandler(wsHub))
}
