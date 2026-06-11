package handlers

import (
	"github.com/evocrm/backend/internal/models"
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func GetDashboard(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		stats := models.DashboardStats{}

		// Open conversations
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'open'", companyID).Scan(&stats.OpenConversations)

		// Pending conversations
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'pending'", companyID).Scan(&stats.PendingConversations)

		// My conversations
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND assigned_to = $2 AND status IN ('open', 'in_progress')", companyID, userID).Scan(&stats.MyConversations)

		// Active deals
		svc.DB.QueryRow("SELECT COUNT(*) FROM deals WHERE company_id = $1 AND status = 'open'", companyID).Scan(&stats.ActiveDeals)

		// Total deals value
		svc.DB.QueryRow("SELECT COALESCE(SUM(value), 0) FROM deals WHERE company_id = $1 AND status = 'open'", companyID).Scan(&stats.TotalDealsValue)

		// Calls today
		svc.DB.QueryRow("SELECT COUNT(*) FROM calls WHERE company_id = $1 AND created_at >= CURRENT_DATE", companyID).Scan(&stats.CallsToday)

		// Connected channels
		svc.DB.QueryRow("SELECT COUNT(*) FROM channels WHERE company_id = $1 AND status = 'connected'", companyID).Scan(&stats.ConnectedChannels)

		// Get announcements
		rows, _ := svc.DB.Query(`
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
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var a models.InternalAnnouncement
				rows.Scan(&a.ID, &a.Title, &a.Content, &a.Priority, &a.IsPinned, &a.PublishedAt, &a.AuthorName, &a.IsRead)
				announcements = append(announcements, a)
			}
		}

		// Get channels status
		channelRows, _ := svc.DB.Query(`
			SELECT c.id, c.name, c.type, c.status
			FROM channels c
			WHERE c.company_id = $1 AND c.is_active = true
		`, companyID)

		var channels []map[string]string
		if channelRows != nil {
			defer channelRows.Close()
			for channelRows.Next() {
				var id, name, chType, status string
				channelRows.Scan(&id, &name, &chType, &status)
				channels = append(channels, map[string]string{
					"id": id, "name": name, "type": chType, "status": status,
				})
			}
		}

		return c.JSON(fiber.Map{
			"stats":         stats,
			"announcements": announcements,
			"channels":      channels,
		})
	}
}
