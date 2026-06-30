package handlers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/evocrm/backend/internal/models"
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func GetDashboard(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		teamID := c.Query("team_id")
		channelID := c.Query("channel_id")
		period := normalizeDashboardPeriod(c.Query("period", "today"))

		stats := models.DashboardStats{}
		filterWhere, filterArgs := dashboardConversationFilter(companyID, teamID, channelID, period)
		liveWhere, liveArgs := dashboardConversationFilter(companyID, teamID, channelID, "")

		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'open'", companyID).Scan(&stats.OpenConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'pending'", companyID).Scan(&stats.PendingConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND assigned_to = $2 AND status IN ('open', 'in_progress')", companyID, userID).Scan(&stats.MyConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM deals WHERE company_id = $1 AND status = 'open'", companyID).Scan(&stats.ActiveDeals)
		svc.DB.QueryRow("SELECT COALESCE(SUM(value), 0) FROM deals WHERE company_id = $1 AND status = 'open'", companyID).Scan(&stats.TotalDealsValue)
		svc.DB.QueryRow("SELECT COUNT(*) FROM channels WHERE company_id = $1 AND status = 'connected'", companyID).Scan(&stats.ConnectedChannels)
		svc.DB.QueryRow("SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))), 0) FROM conversations WHERE company_id = $1 AND first_response_at IS NOT NULL", companyID).Scan(&stats.AvgResponseTime)

		operations := fiber.Map{}
		queryInt(svc.DB, operations, "active_conversations", "SELECT COUNT(*) FROM conversations WHERE "+liveWhere+" AND status IN ('open', 'in_progress', 'pending')", liveArgs...)
		queryInt(svc.DB, operations, "queue_size", "SELECT COUNT(*) FROM conversations WHERE "+liveWhere+" AND status IN ('open', 'pending') AND assigned_to IS NULL", liveArgs...)
		queryFloat(svc.DB, operations, "average_wait_seconds", "SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(first_response_at, NOW()) - created_at))), 0) FROM conversations WHERE "+filterWhere+" AND status IN ('open', 'pending', 'in_progress')", filterArgs...)
		queryFloat(svc.DB, operations, "first_response_seconds", "SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))), 0) FROM conversations WHERE "+filterWhere+" AND first_response_at IS NOT NULL", filterArgs...)
		queryFloat(svc.DB, operations, "average_handle_seconds", "SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at))), 0) FROM conversations WHERE "+filterWhere+" AND resolved_at IS NOT NULL", filterArgs...)
		queryFloat(svc.DB, operations, "abandonment_rate", "SELECT COALESCE(100.0 * SUM(CASE WHEN first_response_at IS NULL AND status = 'resolved' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) FROM conversations WHERE "+filterWhere, filterArgs...)
		queryFloat(svc.DB, operations, "first_contact_resolution_rate", "SELECT COALESCE(100.0 * SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) FROM conversations WHERE "+filterWhere, filterArgs...)
		queryInt(svc.DB, operations, "transfers", "SELECT COUNT(*) FROM audit_logs WHERE company_id = $1 AND action ILIKE '%transfer%' AND created_at >= "+periodSQL(period), companyID)
		operations["comparison"] = fetchDashboardComparison(svc.DB, companyID, teamID, channelID, period)

		return c.JSON(fiber.Map{
			"stats":                 stats,
			"operations":            operations,
			"queue_by_channel":      fetchQueueByChannel(svc.DB, companyID, teamID, channelID),
			"peak_hours":            fetchPeakHours(svc.DB, companyID, teamID, channelID),
			"resolution_by_channel": fetchResolutionByChannel(svc.DB, companyID, teamID, channelID, period),
			"agents":                fetchAgentDashboard(svc.DB, companyID),
			"channel_distribution":  fetchChannelDistribution(svc.DB, companyID, teamID, channelID, period),
			"sla_by_channel":        fetchSLAByChannel(svc.DB, companyID, teamID, channelID, period),
			"sla_alerts":            fetchSLAAlerts(svc.DB, companyID, teamID, channelID),
			"filters":               fetchDashboardFilters(svc.DB, companyID),
			"announcements":         fetchDashboardAnnouncements(svc.DB, companyID, userID),
			"channels":              fetchDashboardChannels(svc.DB, companyID),
		})
	}
}

func normalizeDashboardPeriod(period string) string {
	switch period {
	case "today", "7d", "30d":
		return period
	default:
		return "today"
	}
}

func periodSQL(period string) string {
	switch period {
	case "7d":
		return "NOW() - INTERVAL '7 days'"
	case "30d":
		return "NOW() - INTERVAL '30 days'"
	default:
		return "date_trunc('day', NOW())"
	}
}

func previousPeriodCondition(period string) string {
	switch period {
	case "7d":
		return "created_at >= NOW() - INTERVAL '14 days' AND created_at < NOW() - INTERVAL '7 days'"
	case "30d":
		return "created_at >= NOW() - INTERVAL '60 days' AND created_at < NOW() - INTERVAL '30 days'"
	default:
		return "created_at >= date_trunc('day', NOW()) - INTERVAL '1 day' AND created_at < date_trunc('day', NOW()) - INTERVAL '1 day' + (NOW() - date_trunc('day', NOW()))"
	}
}

func dashboardConversationFilter(companyID, teamID, channelID, period string) (string, []interface{}) {
	args := []interface{}{companyID}
	where := "company_id = $1"
	next := 2
	if teamID != "" {
		where += fmt.Sprintf(" AND team_id = $%d", next)
		args = append(args, teamID)
		next++
	}
	if channelID != "" {
		where += fmt.Sprintf(" AND channel_id = $%d", next)
		args = append(args, channelID)
		next++
	}
	if period != "" {
		where += " AND created_at >= " + periodSQL(period)
	}
	return where, args
}

func queryInt(db *sql.DB, target fiber.Map, key, query string, args ...interface{}) {
	var value int
	_ = db.QueryRow(query, args...).Scan(&value)
	target[key] = value
}

func queryFloat(db *sql.DB, target fiber.Map, key, query string, args ...interface{}) {
	var value float64
	_ = db.QueryRow(query, args...).Scan(&value)
	target[key] = value
}

func fetchDashboardComparison(db *sql.DB, companyID, teamID, channelID, period string) fiber.Map {
	baseWhere, baseArgs := dashboardConversationFilter(companyID, teamID, channelID, "")
	currentWhere, currentArgs := dashboardConversationFilter(companyID, teamID, channelID, period)
	previousWhere := baseWhere + " AND " + previousPeriodCondition(period)

	return fiber.Map{
		"active_conversations": queryDeltaPercent(db,
			"SELECT COUNT(*) FROM conversations WHERE "+currentWhere+" AND status IN ('open', 'in_progress', 'pending')",
			currentArgs,
			"SELECT COUNT(*) FROM conversations WHERE "+previousWhere+" AND status IN ('open', 'in_progress', 'pending')",
			baseArgs,
		),
		"queue_size": queryDeltaPercent(db,
			"SELECT COUNT(*) FROM conversations WHERE "+currentWhere+" AND status IN ('open', 'pending') AND assigned_to IS NULL",
			currentArgs,
			"SELECT COUNT(*) FROM conversations WHERE "+previousWhere+" AND status IN ('open', 'pending') AND assigned_to IS NULL",
			baseArgs,
		),
		"average_wait_seconds": queryDeltaPercent(db,
			"SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(first_response_at, NOW()) - created_at))), 0) FROM conversations WHERE "+currentWhere,
			currentArgs,
			"SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(first_response_at, NOW()) - created_at))), 0) FROM conversations WHERE "+previousWhere,
			baseArgs,
		),
		"first_response_seconds": queryDeltaPercent(db,
			"SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))), 0) FROM conversations WHERE "+currentWhere+" AND first_response_at IS NOT NULL",
			currentArgs,
			"SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))), 0) FROM conversations WHERE "+previousWhere+" AND first_response_at IS NOT NULL",
			baseArgs,
		),
	}
}

func queryDeltaPercent(db *sql.DB, currentQuery string, currentArgs []interface{}, previousQuery string, previousArgs []interface{}) float64 {
	var current, previous float64
	_ = db.QueryRow(currentQuery, currentArgs...).Scan(&current)
	_ = db.QueryRow(previousQuery, previousArgs...).Scan(&previous)
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return ((current - previous) / previous) * 100
}

func fetchQueueByChannel(db *sql.DB, companyID, teamID, channelID string) []fiber.Map {
	where, args := dashboardConversationFilter(companyID, teamID, channelID, "")
	rows, err := db.Query(`
		SELECT COALESCE(ch.name, 'Sem canal'), COALESCE(ch.type, 'desconhecido'),
		       COUNT(*) FILTER (WHERE c.status IN ('open', 'pending') AND c.assigned_to IS NULL) AS queue_size,
		       COUNT(*) FILTER (WHERE c.status IN ('open', 'in_progress', 'pending')) AS active_count,
		       COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(c.first_response_at, NOW()) - c.created_at))) FILTER (WHERE c.status IN ('open', 'pending', 'in_progress')), 0) AS avg_wait_seconds
		FROM conversations c
		LEFT JOIN channels ch ON ch.id = c.channel_id
		WHERE `+strings.ReplaceAll(where, "company_id", "c.company_id")+`
		GROUP BY ch.name, ch.type
		ORDER BY queue_size DESC, active_count DESC
	`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := []fiber.Map{}
	for rows.Next() {
		var name, channelType string
		var queueSize, activeCount int
		var avgWait float64
		rows.Scan(&name, &channelType, &queueSize, &activeCount, &avgWait)
		items = append(items, fiber.Map{"name": name, "type": channelType, "queue_size": queueSize, "active_count": activeCount, "avg_wait_seconds": avgWait})
	}
	return items
}

func fetchPeakHours(db *sql.DB, companyID, teamID, channelID string) []fiber.Map {
	where, args := dashboardConversationFilter(companyID, teamID, channelID, "")
	rows, err := db.Query(`
		SELECT to_char(date_trunc('hour', created_at), 'HH24:00') AS hour_label, COUNT(*)
		FROM conversations
		WHERE `+where+` AND created_at >= NOW() - INTERVAL '12 hours'
		GROUP BY date_trunc('hour', created_at)
		ORDER BY date_trunc('hour', created_at)
	`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := []fiber.Map{}
	for rows.Next() {
		var label string
		var total int
		rows.Scan(&label, &total)
		items = append(items, fiber.Map{"label": label, "total": total})
	}
	return items
}

func fetchResolutionByChannel(db *sql.DB, companyID, teamID, channelID, period string) []fiber.Map {
	where, args := dashboardConversationFilter(companyID, teamID, channelID, period)
	rows, err := db.Query(`
		SELECT COALESCE(ch.name, 'Sem canal'), COALESCE(ch.type, 'desconhecido'),
		       COALESCE(AVG(EXTRACT(EPOCH FROM (c.resolved_at - c.created_at))), 0) AS avg_resolution_seconds,
		       COUNT(*) FILTER (WHERE c.resolved_at IS NOT NULL) AS resolved_count
		FROM conversations c
		LEFT JOIN channels ch ON ch.id = c.channel_id
		WHERE `+strings.ReplaceAll(where, "company_id", "c.company_id")+`
		GROUP BY ch.name, ch.type
		ORDER BY resolved_count DESC
	`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := []fiber.Map{}
	for rows.Next() {
		var name, channelType string
		var avg float64
		var count int
		rows.Scan(&name, &channelType, &avg, &count)
		items = append(items, fiber.Map{"name": name, "type": channelType, "avg_resolution_seconds": avg, "resolved_count": count})
	}
	return items
}

func fetchAgentDashboard(db *sql.DB, companyID string) fiber.Map {
	availability := fiber.Map{"online": 0, "busy": 0, "pause": 0, "offline": 0}
	rows, err := db.Query(`
		SELECT u.id, COALESCE(u.name, 'Agente'), COALESCE(u.availability_status, CASE WHEN u.is_online THEN 'online' ELSE 'offline' END),
		       EXTRACT(EPOCH FROM (NOW() - COALESCE(u.last_seen_at, u.updated_at, u.created_at))),
		       COUNT(c.id) FILTER (WHERE c.status IN ('open', 'in_progress', 'pending')) AS active_chats
		FROM users u
		LEFT JOIN conversations c ON c.assigned_to = u.id AND c.company_id = u.company_id
		WHERE u.company_id = $1 AND u.is_active = true
		GROUP BY u.id, u.name, u.availability_status, u.is_online, u.last_seen_at, u.updated_at, u.created_at
		ORDER BY active_chats DESC, u.name
	`, companyID)
	if err != nil {
		return fiber.Map{"availability": availability, "workload": []fiber.Map{}}
	}
	defer rows.Close()
	workload := []fiber.Map{}
	for rows.Next() {
		var id, name, status string
		var idleSeconds float64
		var activeChats int
		rows.Scan(&id, &name, &status, &idleSeconds, &activeChats)
		if status == "paused" {
			status = "pause"
		}
		if _, ok := availability[status]; !ok {
			status = "offline"
		}
		availability[status] = availability[status].(int) + 1
		workload = append(workload, fiber.Map{"id": id, "name": name, "status": status, "active_chats": activeChats, "idle_seconds": idleSeconds})
	}
	return fiber.Map{"availability": availability, "workload": workload}
}

func fetchChannelDistribution(db *sql.DB, companyID, teamID, channelID, period string) []fiber.Map {
	where, args := dashboardConversationFilter(companyID, teamID, channelID, period)
	rows, err := db.Query(`
		SELECT COALESCE(ch.name, 'Sem canal'), COALESCE(ch.type, 'desconhecido'), COUNT(*)
		FROM conversations c
		LEFT JOIN channels ch ON ch.id = c.channel_id
		WHERE `+strings.ReplaceAll(where, "company_id", "c.company_id")+`
		GROUP BY ch.name, ch.type
		ORDER BY COUNT(*) DESC
	`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := []fiber.Map{}
	for rows.Next() {
		var name, channelType string
		var total int
		rows.Scan(&name, &channelType, &total)
		items = append(items, fiber.Map{"name": name, "type": channelType, "total": total})
	}
	return items
}

func fetchSLAByChannel(db *sql.DB, companyID, teamID, channelID, period string) []fiber.Map {
	where, args := dashboardConversationFilter(companyID, teamID, channelID, period)
	rows, err := db.Query(`
		SELECT COALESCE(ch.name, 'Sem canal'), COALESCE(ch.type, 'desconhecido'),
		       COALESCE(100.0 * SUM(CASE WHEN c.first_response_due_at IS NULL OR (c.first_response_at IS NOT NULL AND c.first_response_at <= c.first_response_due_at) THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) AS first_sla,
		       COALESCE(100.0 * SUM(CASE WHEN c.resolution_due_at IS NULL OR (c.resolved_at IS NOT NULL AND c.resolved_at <= c.resolution_due_at) THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) AS resolution_sla
		FROM conversations c
		LEFT JOIN channels ch ON ch.id = c.channel_id
		WHERE `+strings.ReplaceAll(where, "company_id", "c.company_id")+`
		GROUP BY ch.name, ch.type
		ORDER BY first_sla ASC
	`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := []fiber.Map{}
	for rows.Next() {
		var name, channelType string
		var first, resolution float64
		rows.Scan(&name, &channelType, &first, &resolution)
		items = append(items, fiber.Map{"name": name, "type": channelType, "first_response_sla": first, "resolution_sla": resolution})
	}
	return items
}

func fetchSLAAlerts(db *sql.DB, companyID, teamID, channelID string) []fiber.Map {
	where, args := dashboardConversationFilter(companyID, teamID, channelID, "")
	rows, err := db.Query(`
		SELECT c.id, COALESCE(co.name, co.phone, 'Contato'), COALESCE(ch.name, 'Sem canal'), c.status,
		       EXTRACT(EPOCH FROM (COALESCE(c.first_response_due_at, c.resolution_due_at) - NOW())) AS seconds_remaining
		FROM conversations c
		LEFT JOIN contacts co ON co.id = c.contact_id
		LEFT JOIN channels ch ON ch.id = c.channel_id
		WHERE `+strings.ReplaceAll(where, "company_id", "c.company_id")+`
		  AND c.status IN ('open', 'pending', 'in_progress')
		  AND (
		    (c.first_response_due_at IS NOT NULL AND c.first_response_at IS NULL AND c.first_response_due_at <= NOW() + INTERVAL '30 minutes')
		    OR (c.resolution_due_at IS NOT NULL AND c.resolved_at IS NULL AND c.resolution_due_at <= NOW() + INTERVAL '30 minutes')
		  )
		ORDER BY seconds_remaining ASC
		LIMIT 8
	`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := []fiber.Map{}
	for rows.Next() {
		var id, contact, channel, status string
		var seconds float64
		rows.Scan(&id, &contact, &channel, &status, &seconds)
		items = append(items, fiber.Map{"id": id, "contact_name": contact, "channel_name": channel, "status": status, "seconds_remaining": seconds})
	}
	return items
}

func fetchDashboardFilters(db *sql.DB, companyID string) fiber.Map {
	teams := []fiber.Map{}
	teamRows, _ := db.Query("SELECT id, COALESCE(name, 'Time') FROM teams WHERE company_id = $1 AND COALESCE(is_active, true) = true ORDER BY name", companyID)
	if teamRows != nil {
		defer teamRows.Close()
		for teamRows.Next() {
			var id, name string
			teamRows.Scan(&id, &name)
			teams = append(teams, fiber.Map{"id": id, "name": name})
		}
	}
	return fiber.Map{"teams": teams, "periods": []string{"today", "7d", "30d"}}
}

func fetchDashboardAnnouncements(db *sql.DB, companyID, userID string) []models.InternalAnnouncement {
	rows, _ := db.Query(`
		SELECT ia.id, ia.title, ia.content, ia.priority, ia.is_pinned, ia.published_at,
			   u.name as author_name,
			   EXISTS(SELECT 1 FROM announcement_reads ar WHERE ar.announcement_id = ia.id AND ar.user_id = $2) as is_read
		FROM internal_announcements ia
		JOIN users u ON ia.author_id = u.id
		WHERE ia.company_id = $1 AND (ia.expires_at IS NULL OR ia.expires_at > NOW())
		ORDER BY ia.is_pinned DESC, ia.published_at DESC
		LIMIT 10
	`, companyID, userID)
	var announcements []models.InternalAnnouncement
	if rows == nil {
		return announcements
	}
	defer rows.Close()
	for rows.Next() {
		var a models.InternalAnnouncement
		rows.Scan(&a.ID, &a.Title, &a.Content, &a.Priority, &a.IsPinned, &a.PublishedAt, &a.AuthorName, &a.IsRead)
		announcements = append(announcements, a)
	}
	return announcements
}

func fetchDashboardChannels(db *sql.DB, companyID string) []fiber.Map {
	channelRows, _ := db.Query(`
		SELECT c.id, c.name, c.type, c.status
		FROM channels c
		WHERE c.company_id = $1 AND c.is_active = true
		ORDER BY c.name
	`, companyID)
	var channels []fiber.Map
	if channelRows == nil {
		return channels
	}
	defer channelRows.Close()
	for channelRows.Next() {
		var id, name, chType, status string
		channelRows.Scan(&id, &name, &chType, &status)
		channels = append(channels, fiber.Map{"id": id, "name": name, "type": chType, "status": status})
	}
	return channels
}
