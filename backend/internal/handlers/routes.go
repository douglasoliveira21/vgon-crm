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
	protected := api.Group("", middleware.AuthMiddleware(svc.Config), middleware.ActiveTenantMiddleware(svc.DB))

	// Dashboard
	protected.Get("/dashboard", GetDashboard(svc))
	protected.Get("/search", GlobalSearch(svc))

	// User profile
	protected.Get("/me", GetCurrentUser(svc))
	protected.Put("/me", UpdateCurrentUser(svc))
	protected.Put("/me/status", UpdateCurrentUserStatus(svc))
	protected.Put("/me/password", UpdateCurrentUserPassword(svc))
	protected.Post("/me/avatar", UploadCurrentUserAvatar(svc))

	// WhatsApp Instances
	whatsapp := protected.Group("/whatsapp", middleware.DenyRoles("agent", "supervisor"))
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
	conversationAccess := middleware.ConversationAccess(svc.DB)
	conversations.Get("/:id", conversationAccess, GetConversation(svc))
	conversations.Post("/:id/assign", conversationAccess, AssignConversation(svc))
	conversations.Post("/:id/unassign", conversationAccess, UnassignConversation(svc))
	conversations.Post("/:id/transfer", conversationAccess, TransferConversation(svc))
	conversations.Put("/:id/customer-company", conversationAccess, LinkConversationCustomerCompany(svc))
	conversations.Post("/:id/close", conversationAccess, CloseConversation(svc))
	conversations.Post("/:id/reopen", conversationAccess, ReopenConversation(svc))
	conversations.Post("/:id/read", conversationAccess, MarkConversationRead(svc))
	conversations.Post("/:id/unread", conversationAccess, MarkConversationUnread(svc))
	conversations.Get("/:id/messages", conversationAccess, GetConversationMessages(svc))
	conversations.Post("/:id/messages/text", conversationAccess, SendTextMessage(svc))
	conversations.Post("/:id/messages/media", conversationAccess, SendMediaMessage(svc))
	conversations.Post("/:id/messages/audio", conversationAccess, SendAudioMessage(svc))
	conversations.Delete("/:id/messages/:msgId", conversationAccess, DeleteMessage(svc))

	// Contacts
	contacts := protected.Group("/contacts")
	contacts.Get("/", GetContacts(svc))
	contacts.Get("/:id/export", ExportContactData(svc))
	contacts.Get("/:id/audit", GetContactAudit(svc))
	contacts.Post("/:id/consent", UpdateContactConsent(svc))
	contacts.Get("/:id", GetContact(svc))
	contacts.Post("/", CreateContact(svc))
	contacts.Put("/:id", UpdateContact(svc))
	contacts.Delete("/:id", DeleteContact(svc))
	contacts.Post("/:id/block", BlockContact(svc))
	contacts.Post("/:id/unblock", UnblockContact(svc))
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
	teams.Post("/", middleware.DenyRoles("agent", "supervisor"), CreateTeam(svc))
	teams.Put("/:id", middleware.DenyRoles("agent", "supervisor"), UpdateTeam(svc))
	teams.Delete("/:id", middleware.DenyRoles("agent", "supervisor"), DeleteTeam(svc))
	teams.Post("/:id/members", middleware.DenyRoles("agent"), middleware.SupervisorTeamAccess(svc.DB), AddTeamMember(svc))
	teams.Delete("/:id/members/:userId", middleware.DenyRoles("agent", "supervisor"), RemoveTeamMember(svc))

	// Channels
	channels := protected.Group("/channels")
	channels.Get("/", GetChannels(svc))
	channels.Get("/email/oauth/:provider/start", middleware.DenyRoles("agent", "supervisor"), StartEmailOAuth(svc))
	channels.Post("/email", middleware.DenyRoles("agent", "supervisor"), CreateEmailChannel(svc))
	channels.Put("/email/:id", middleware.DenyRoles("agent", "supervisor"), UpdateEmailChannel(svc))
	channels.Delete("/email/:id", middleware.DenyRoles("agent", "supervisor"), DeleteEmailChannel(svc))
	channels.Post("/email/:id/sync", middleware.DenyRoles("agent", "supervisor"), SyncEmailChannel(svc))

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
	bots := protected.Group("/bot-flows", middleware.DenyRoles("agent", "supervisor"))
	bots.Get("/", GetBotFlows(svc))
	bots.Get("/:id", GetBotFlow(svc))
	bots.Post("/", CreateBotFlow(svc))
	bots.Post("/:id/duplicate", DuplicateBotFlow(svc))
	bots.Put("/:id", UpdateBotFlow(svc))
	bots.Delete("/:id", DeleteBotFlow(svc))

	// Campaigns
	campaigns := protected.Group("/campaigns", middleware.DenyRoles("agent", "supervisor"))
	campaigns.Get("/", GetCampaigns(svc))
	campaigns.Post("/", CreateCampaign(svc))
	campaigns.Put("/:id", UpdateCampaign(svc))
	campaigns.Delete("/:id", DeleteCampaign(svc))
	campaigns.Post("/email/send", SendEmailCampaign(svc))
	campaigns.Post("/:id/start", StartCampaign(svc))
	campaigns.Post("/:id/pause", PauseCampaign(svc))

	// Metrics
	protected.Get("/metrics", middleware.DenyRoles("agent"), GetMetrics(svc))
	protected.Get("/metrics/attendance", middleware.DenyRoles("agent"), GetAttendanceMetrics(svc))
	protected.Get("/audit-logs", middleware.DenyRoles("agent", "supervisor"), GetAuditLogs(svc))

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
	widgets := protected.Group("/widgets", middleware.DenyRoles("agent", "supervisor"))
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
	quickReplies.Post("/", middleware.DenyRoles("agent"), CreateQuickReply(svc))
	quickReplies.Put("/:id", middleware.DenyRoles("agent"), UpdateQuickReply(svc))
	quickReplies.Delete("/:id", middleware.DenyRoles("agent"), DeleteQuickReply(svc))

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
	admin.Get("/health", GetPlatformHealth(svc))
	admin.Get("/audit-logs", GetGlobalAuditLogs(svc))
	admin.Get("/incidents", GetPlatformIncidents(svc))
	admin.Post("/incidents/:id/retry", RetryPlatformIncident(svc))
	admin.Get("/sessions", GetAdminSessions(svc))
	admin.Delete("/sessions/:id", RevokeAdminSession(svc))
	admin.Get("/security", GetAdminSecuritySummary(svc))
	admin.Post("/tenants/:id/impersonate", ImpersonateTenant(svc))
	admin.Get("/tenants/:id/export", ExportTenantData(svc))
	admin.Post("/tenants/:id/schedule-deletion", ScheduleTenantDeletion(svc))
	admin.Post("/tenants/:id/restore", RestoreTenant(svc))

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
