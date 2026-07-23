package handlers

import (
	"github.com/evocrm/backend/internal/middleware"
	"github.com/evocrm/backend/internal/services"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, svc *services.Container, wsHub *websocket.Hub) {
	// Health check
	app.Get("/health", Liveness())
	app.Get("/health/live", Liveness())
	app.Get("/health/ready", Readiness(svc))

	app.Get("/uploads/*", ServeUpload(svc))

	api := app.Group("/api")
	api.Get("/health/live", Liveness())
	api.Get("/health/ready", Readiness(svc))
	api.Get("/media/:messageId", MediaProxy(svc))

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
	api.Post("/webhooks/email/events", HandleEmailDeliveryEvent(svc))
	api.Get("/marketing/unsubscribe", HandleMarketingUnsubscribe(svc))

	// ============================================
	// Widget Public Routes (CORS open - widget can be embedded on any site)
	// ============================================
	widgetCORS := func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, X-Widget-Session")
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
	protected := api.Group("", middleware.AuthMiddleware(svc.DB, svc.Config), middleware.ActiveTenantMiddleware(svc.DB))

	// Dashboard
	protected.Get("/dashboard", middleware.RBACMiddleware("dashboard.view"), GetDashboard(svc))
	protected.Get("/search", middleware.RBACMiddleware("search.view"), GlobalSearch(svc))

	// User profile
	protected.Get("/me", GetCurrentUser(svc))
	protected.Put("/me", UpdateCurrentUser(svc))
	protected.Put("/me/status", UpdateCurrentUserStatus(svc))
	protected.Put("/me/password", UpdateCurrentUserPassword(svc))
	protected.Post("/me/avatar", UploadCurrentUserAvatar(svc))
	protected.Post("/me/2fa/setup", BeginTwoFactorSetup(svc))
	protected.Post("/me/2fa/confirm", ConfirmTwoFactor(svc))
	protected.Delete("/me/2fa", DisableTwoFactor(svc))
	protected.Get("/me/sessions", GetCurrentSessions(svc))
	protected.Delete("/me/sessions/:id", RevokeCurrentSession(svc))
	protected.Post("/auth/logout", AuthLogout(svc))
	protected.Post("/auth/impersonation/end", EndImpersonation(svc))

	// WhatsApp Instances
	whatsapp := protected.Group("/whatsapp", middleware.RBACMiddleware("channels.manage"))
	whatsapp.Get("/instances", GetWhatsAppInstances(svc))
	whatsapp.Post("/instances", CreateWhatsAppInstance(svc))
	whatsapp.Get("/instances/:id/qrcode", GetWhatsAppQRCode(svc))
	whatsapp.Get("/instances/:id/status", GetWhatsAppStatus(svc))
	whatsapp.Post("/instances/:id/disconnect", DisconnectWhatsAppInstance(svc))
	whatsapp.Delete("/instances/:id", DeleteWhatsAppInstance(svc))
	whatsapp.Post("/instances/:id/sync-contacts", SyncWhatsAppContacts(svc))
	whatsapp.Post("/instances/:id/sync-photos", SyncWhatsAppPhotos(svc))

	// Conversations
	conversations := protected.Group("/conversations", middleware.RBACMiddleware("conversations.own"))
	conversations.Get("/", GetConversations(svc))
	conversations.Get("/assigned-to-me", GetMyConversations(svc))
	conversations.Post("/start", middleware.RBACMiddleware("conversations.create"), StartConversation(svc))
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
	contacts := protected.Group("/contacts", middleware.RBACMiddleware("contacts.view"))
	contacts.Get("/", GetContacts(svc))
	contacts.Get("/duplicates", middleware.RBACMiddleware("contacts.edit"), GetDuplicateContacts(svc))
	contacts.Get("/:id/export", ExportContactData(svc))
	contacts.Get("/:id/audit", GetContactAudit(svc))
	contacts.Post("/:id/consent", middleware.RBACMiddleware("contacts.edit"), UpdateContactConsent(svc))
	contacts.Get("/:id/channel-consents", GetContactChannelConsents(svc))
	contacts.Put("/:id/channel-consents", middleware.RBACMiddleware("contacts.edit"), UpsertContactChannelConsent(svc))
	contacts.Get("/:id", GetContact(svc))
	contacts.Post("/", middleware.RBACMiddleware("contacts.edit"), CreateContact(svc))
	contacts.Put("/:id", middleware.RBACMiddleware("contacts.edit"), UpdateContact(svc))
	contacts.Post("/:id/merge", middleware.RBACMiddleware("contacts.edit"), MergeContacts(svc))
	contacts.Delete("/:id", middleware.RBACMiddleware("contacts.delete"), DeleteContact(svc))
	contacts.Post("/:id/block", middleware.RBACMiddleware("contacts.edit"), BlockContact(svc))
	contacts.Post("/:id/unblock", middleware.RBACMiddleware("contacts.edit"), UnblockContact(svc))
	contacts.Post("/:id/tags", middleware.RBACMiddleware("contacts.edit"), AddContactTag(svc))
	contacts.Delete("/:id/tags/:tagId", middleware.RBACMiddleware("contacts.edit"), RemoveContactTag(svc))

	// Customer companies
	customerCompanies := protected.Group("/customer-companies", middleware.RBACMiddleware("companies.view"))
	customerCompanies.Get("/", ListCustomerCompanies(svc))
	customerCompanies.Get("/lookup/:cnpj", LookupCNPJ(svc))
	customerCompanies.Get("/sla-report.csv", ExportCustomerCompanySLAReport(svc))
	customerCompanies.Post("/", middleware.RBACMiddleware("companies.edit"), CreateCustomerCompany(svc))
	customerCompanies.Put("/:id", middleware.RBACMiddleware("companies.edit"), UpdateCustomerCompany(svc))
	customerCompanies.Delete("/:id", middleware.RBACMiddleware("companies.delete"), DeleteCustomerCompany(svc))

	// Teams
	teams := protected.Group("/teams", middleware.RBACMiddleware("teams.view"))
	teams.Get("/", GetTeams(svc))
	teams.Post("/", middleware.DenyRoles("agent", "supervisor"), CreateTeam(svc))
	teams.Put("/:id", middleware.DenyRoles("agent", "supervisor"), UpdateTeam(svc))
	teams.Delete("/:id", middleware.DenyRoles("agent", "supervisor"), DeleteTeam(svc))
	teams.Post("/:id/members", middleware.DenyRoles("agent"), middleware.SupervisorTeamAccess(svc.DB), AddTeamMember(svc))
	teams.Delete("/:id/members/:userId", middleware.DenyRoles("agent", "supervisor"), RemoveTeamMember(svc))

	// Channels
	channels := protected.Group("/channels", middleware.RBACMiddleware("channels.manage"))
	channels.Get("/", GetChannels(svc))
	channels.Get("/email/oauth/:provider/start", middleware.DenyRoles("agent", "supervisor"), StartEmailOAuth(svc))
	channels.Post("/email", middleware.DenyRoles("agent", "supervisor"), CreateEmailChannel(svc))
	channels.Put("/email/:id", middleware.DenyRoles("agent", "supervisor"), UpdateEmailChannel(svc))
	channels.Delete("/email/:id", middleware.DenyRoles("agent", "supervisor"), DeleteEmailChannel(svc))
	channels.Post("/email/:id/sync", middleware.DenyRoles("agent", "supervisor"), SyncEmailChannel(svc))

	// Funnels
	funnels := protected.Group("/funnels", middleware.RBACMiddleware("funnels.view"))
	funnels.Get("/", GetFunnels(svc))
	funnels.Post("/", middleware.RBACMiddleware("funnels.edit"), CreateFunnel(svc))
	funnels.Put("/:id", middleware.RBACMiddleware("funnels.edit"), UpdateFunnel(svc))
	funnels.Delete("/:id", middleware.RBACMiddleware("funnels.delete"), DeleteFunnel(svc))

	// Deals
	deals := protected.Group("/deals", middleware.RBACMiddleware("deals.view"))
	deals.Get("/", GetDeals(svc))
	deals.Post("/", middleware.RBACMiddleware("deals.edit"), CreateDeal(svc))
	deals.Put("/:id", middleware.RBACMiddleware("deals.edit"), UpdateDeal(svc))
	deals.Put("/:id/stage", middleware.RBACMiddleware("deals.edit"), MoveDealStage(svc))

	// Bot Flows
	bots := protected.Group("/bot-flows", middleware.RBACMiddleware("bots.view"))
	bots.Get("/", GetBotFlows(svc))
	bots.Get("/:id", GetBotFlow(svc))
	bots.Post("/", CreateBotFlow(svc))
	bots.Post("/:id/duplicate", DuplicateBotFlow(svc))
	bots.Post("/:id/publish", PublishBotFlow(svc))
	bots.Post("/:id/unpublish", UnpublishBotFlow(svc))
	bots.Get("/:id/versions", GetBotFlowVersions(svc))
	bots.Post("/:id/versions/:version/rollback", RollbackBotFlow(svc))
	bots.Post("/:id/simulate", SimulateBotFlow(svc))
	bots.Get("/:id/executions", GetBotExecutionHistory(svc))
	bots.Put("/:id", UpdateBotFlow(svc))
	bots.Delete("/:id", DeleteBotFlow(svc))

	// Campaigns
	campaigns := protected.Group("/campaigns", middleware.RBACMiddleware("campaigns.view"))
	campaigns.Get("/", GetCampaigns(svc))
	campaigns.Post("/", CreateCampaign(svc))
	campaigns.Put("/:id", UpdateCampaign(svc))
	campaigns.Delete("/:id", DeleteCampaign(svc))
	campaigns.Post("/email/send", SendEmailCampaign(svc))
	campaigns.Post("/:id/start", StartCampaign(svc))
	campaigns.Post("/:id/approve", middleware.DenyRoles("marketing", "agent", "supervisor"), ApproveCampaign(svc))
	campaigns.Post("/:id/schedule", ScheduleCampaign(svc))
	campaigns.Post("/:id/pause", PauseCampaign(svc))
	campaigns.Get("/suppressions", ListCampaignSuppressions(svc))
	campaigns.Post("/suppressions", CreateCampaignSuppression(svc))
	campaigns.Delete("/suppressions/:id", DeleteCampaignSuppression(svc))

	privacy := protected.Group("/privacy", middleware.RBACMiddleware("audit.view"))
	privacy.Get("/requests", ListDataSubjectRequests(svc))
	privacy.Post("/requests", CreateDataSubjectRequest(svc))
	privacy.Put("/requests/:id", UpdateDataSubjectRequest(svc))
	privacy.Get("/retention", GetRetentionPolicies(svc))
	privacy.Put("/retention", UpsertRetentionPolicy(svc))

	// Metrics
	protected.Get("/metrics", middleware.RBACMiddleware("metrics.view"), GetMetrics(svc))
	protected.Get("/metrics/attendance", middleware.RBACMiddleware("metrics.view"), GetAttendanceMetrics(svc))
	protected.Get("/audit-logs", middleware.RBACMiddleware("audit.view"), GetAuditLogs(svc))
	protected.Get("/audit-logs/verify", middleware.RBACMiddleware("audit.view"), VerifyAuditChain(svc))

	// GLPI
	glpi := protected.Group("/glpi", middleware.RBACMiddleware("glpi.view"))
	glpi.Post("/tickets", GLPICreateTicket(svc))
	glpi.Get("/tickets/:id", GLPIGetTicket(svc))
	glpi.Get("/entities", GLPIGetEntities(svc))
	glpi.Get("/entities/:id", GLPIGetEntity(svc))
	glpi.Post("/flow/start", GLPIStartFlow(svc))

	// Announcements
	announcements := protected.Group("/announcements")
	announcements.Get("/", GetAnnouncements(svc))
	announcements.Post("/", middleware.RBACMiddleware("announcements.manage"), CreateAnnouncement(svc))
	announcements.Post("/:id/read", MarkAnnouncementRead(svc))

	// Widgets
	widgets := protected.Group("/widgets", middleware.RBACMiddleware("widget.view"))
	widgets.Get("/", GetWidgets(svc))
	widgets.Post("/", CreateWidget(svc))
	widgets.Put("/:id", UpdateWidget(svc))

	// Tags
	tags := protected.Group("/tags")
	tags.Get("/", GetTags(svc))
	tags.Post("/", middleware.RBACMiddleware("contacts.edit"), CreateTag(svc))
	tags.Delete("/:id", middleware.RBACMiddleware("contacts.edit"), DeleteTag(svc))

	// Quick Replies
	quickReplies := protected.Group("/quick-replies")
	quickReplies.Get("/", GetQuickReplies(svc))
	quickReplies.Post("/", middleware.DenyRoles("agent"), CreateQuickReply(svc))
	quickReplies.Put("/:id", middleware.DenyRoles("agent"), UpdateQuickReply(svc))
	quickReplies.Delete("/:id", middleware.DenyRoles("agent"), DeleteQuickReply(svc))

	// Users
	users := protected.Group("/users", middleware.RBACMiddleware("users.view"))
	users.Get("/", GetUsers(svc))
	users.Post("/", middleware.RBACMiddleware("users.manage"), CreateUser(svc))
	protected.Get("/jobs/dead-letter", middleware.RBACMiddleware("audit.view"), ListDeadLetterJobs(svc, false))
	protected.Post("/jobs/dead-letter/:id/retry", middleware.RBACMiddleware("audit.view"), RetryDeadLetterJob(svc, false))

	// ============================================
	// Super Admin Routes
	// ============================================
	admin := api.Group("/admin", middleware.AuthMiddleware(svc.DB, svc.Config), middleware.SuperAdminMiddleware(svc.DB))
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
	admin.Get("/jobs/dead-letter", ListDeadLetterJobs(svc, true))
	admin.Post("/jobs/dead-letter/:id/retry", RetryDeadLetterJob(svc, true))
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
	app.Get("/ws/widget", WidgetWebSocketHandler(wsHub, svc))
}
