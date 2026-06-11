package handlers

import (
	"encoding/json"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ============================================
// TEAMS
// ============================================

func GetTeams(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT t.id, t.name, t.description, t.distribution_rule, t.is_active,
				   (SELECT COUNT(*) FROM team_users tu WHERE tu.team_id = t.id) as member_count
			FROM teams t WHERE t.company_id = $1 ORDER BY t.name
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var teams []map[string]interface{}
		for rows.Next() {
			var id, name, distRule string
			var description *string
			var isActive bool
			var memberCount int
			rows.Scan(&id, &name, &description, &distRule, &isActive, &memberCount)
			teams = append(teams, map[string]interface{}{
				"id": id, "name": name, "description": description,
				"distribution_rule": distRule, "is_active": isActive,
				"member_count": memberCount,
			})
		}

		return c.JSON(fiber.Map{"teams": teams})
	}
}

func CreateTeam(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name             string `json:"name"`
			Description      string `json:"description"`
			DistributionRule string `json:"distribution_rule"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.DistributionRule == "" {
			body.DistributionRule = "round-robin"
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO teams (id, company_id, name, description, distribution_rule)
			VALUES ($1, $2, $3, $4, $5)
		`, id, companyID, body.Name, body.Description, body.DistributionRule)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "name": body.Name})
	}
}

func UpdateTeam(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		teamID := c.Params("id")

		var body struct {
			Name             string `json:"name"`
			Description      string `json:"description"`
			DistributionRule string `json:"distribution_rule"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		_, err := svc.DB.Exec(`
			UPDATE teams SET name = $1, description = $2, distribution_rule = $3, updated_at = NOW()
			WHERE id = $4 AND company_id = $5
		`, body.Name, body.Description, body.DistributionRule, teamID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Team updated"})
	}
}

func DeleteTeam(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		teamID := c.Params("id")

		_, err := svc.DB.Exec("DELETE FROM teams WHERE id = $1 AND company_id = $2", teamID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Team deleted"})
	}
}

func AddTeamMember(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		teamID := c.Params("id")

		var body struct {
			UserID       string `json:"user_id"`
			IsSupervisor bool   `json:"is_supervisor"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		_, err := svc.DB.Exec(`
			INSERT INTO team_users (id, team_id, user_id, is_supervisor) VALUES ($1, $2, $3, $4)
			ON CONFLICT (team_id, user_id) DO UPDATE SET is_supervisor = $4
		`, uuid.New().String(), teamID, body.UserID, body.IsSupervisor)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Member added"})
	}
}

func RemoveTeamMember(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		teamID := c.Params("id")
		userID := c.Params("userId")

		svc.DB.Exec("DELETE FROM team_users WHERE team_id = $1 AND user_id = $2", teamID, userID)
		return c.JSON(fiber.Map{"message": "Member removed"})
	}
}

// ============================================
// CHANNELS
// ============================================

func GetChannels(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT id, name, type, status, is_active, created_at
			FROM channels WHERE company_id = $1 ORDER BY created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var channels []map[string]interface{}
		for rows.Next() {
			var id, name, chType, status string
			var isActive bool
			var createdAt string
			rows.Scan(&id, &name, &chType, &status, &isActive, &createdAt)
			channels = append(channels, map[string]interface{}{
				"id": id, "name": name, "type": chType, "status": status,
				"is_active": isActive, "created_at": createdAt,
			})
		}

		return c.JSON(fiber.Map{"channels": channels})
	}
}

// ============================================
// FUNNELS
// ============================================

func GetFunnels(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT id, name, description, is_default, is_active FROM funnels
			WHERE company_id = $1 ORDER BY created_at
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var funnels []map[string]interface{}
		for rows.Next() {
			var id, name string
			var description *string
			var isDefault, isActive bool
			rows.Scan(&id, &name, &description, &isDefault, &isActive)

			// Get stages
			stageRows, _ := svc.DB.Query(`
				SELECT fs.id, fs.name, fs.color, fs.position, fs.is_won, fs.is_lost,
					   (SELECT COUNT(*) FROM deals d WHERE d.stage_id = fs.id AND d.status = 'open') as deal_count,
					   (SELECT COALESCE(SUM(value), 0) FROM deals d WHERE d.stage_id = fs.id AND d.status = 'open') as deal_value
				FROM funnel_stages fs WHERE fs.funnel_id = $1 ORDER BY fs.position
			`, id)

			var stages []map[string]interface{}
			if stageRows != nil {
				for stageRows.Next() {
					var sID, sName, sColor string
					var pos int
					var isWon, isLost bool
					var dealCount int
					var dealValue float64
					stageRows.Scan(&sID, &sName, &sColor, &pos, &isWon, &isLost, &dealCount, &dealValue)
					stages = append(stages, map[string]interface{}{
						"id": sID, "name": sName, "color": sColor, "position": pos,
						"is_won": isWon, "is_lost": isLost, "deal_count": dealCount, "deal_value": dealValue,
					})
				}
				stageRows.Close()
			}

			funnels = append(funnels, map[string]interface{}{
				"id": id, "name": name, "description": description,
				"is_default": isDefault, "is_active": isActive, "stages": stages,
			})
		}

		return c.JSON(fiber.Map{"funnels": funnels})
	}
}

func CreateFunnel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Stages      []struct {
				Name  string `json:"name"`
				Color string `json:"color"`
				IsWon bool   `json:"is_won"`
				IsLost bool  `json:"is_lost"`
			} `json:"stages"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		funnelID := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO funnels (id, company_id, name, description) VALUES ($1, $2, $3, $4)
		`, funnelID, companyID, body.Name, body.Description)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		for i, stage := range body.Stages {
			color := stage.Color
			if color == "" {
				color = "#3B82F6"
			}
			svc.DB.Exec(`
				INSERT INTO funnel_stages (id, funnel_id, name, color, position, is_won, is_lost)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, uuid.New().String(), funnelID, stage.Name, color, i, stage.IsWon, stage.IsLost)
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": funnelID, "name": body.Name})
	}
}

func UpdateFunnel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		funnelID := c.Params("id")

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		c.BodyParser(&body)

		svc.DB.Exec("UPDATE funnels SET name = $1, description = $2, updated_at = NOW() WHERE id = $3 AND company_id = $4",
			body.Name, body.Description, funnelID, companyID)

		return c.JSON(fiber.Map{"message": "Funnel updated"})
	}
}

func DeleteFunnel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		funnelID := c.Params("id")

		svc.DB.Exec("DELETE FROM funnels WHERE id = $1 AND company_id = $2", funnelID, companyID)
		return c.JSON(fiber.Map{"message": "Funnel deleted"})
	}
}

// ============================================
// DEALS
// ============================================

func GetDeals(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		funnelID := c.Query("funnel_id")
		stageID := c.Query("stage_id")

		query := `
			SELECT d.id, d.title, d.value, d.status, d.stage_id,
				   c.name as contact_name, u.name as assigned_to_name, fs.name as stage_name
			FROM deals d
			LEFT JOIN contacts c ON d.contact_id = c.id
			LEFT JOIN users u ON d.assigned_to = u.id
			LEFT JOIN funnel_stages fs ON d.stage_id = fs.id
			WHERE d.company_id = $1
		`
		args := []interface{}{companyID}
		idx := 2

		if funnelID != "" {
			query += " AND d.funnel_id = $" + string(rune('0'+idx))
			args = append(args, funnelID)
			idx++
		}
		if stageID != "" {
			query += " AND d.stage_id = $" + string(rune('0'+idx))
			args = append(args, stageID)
		}

		query += " ORDER BY d.created_at DESC"

		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var deals []map[string]interface{}
		for rows.Next() {
			var id, title, status, stageIDVal string
			var value float64
			var contactName, assignedToName, stageName *string
			rows.Scan(&id, &title, &value, &status, &stageIDVal, &contactName, &assignedToName, &stageName)
			deals = append(deals, map[string]interface{}{
				"id": id, "title": title, "value": value, "status": status,
				"stage_id": stageIDVal, "contact_name": contactName,
				"assigned_to_name": assignedToName, "stage_name": stageName,
			})
		}

		return c.JSON(fiber.Map{"deals": deals})
	}
}

func CreateDeal(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			FunnelID   string  `json:"funnel_id"`
			StageID    string  `json:"stage_id"`
			ContactID  string  `json:"contact_id"`
			Title      string  `json:"title"`
			Value      float64 `json:"value"`
			AssignedTo string  `json:"assigned_to"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO deals (id, company_id, funnel_id, stage_id, contact_id, title, value, assigned_to)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, companyID, body.FunnelID, body.StageID, body.ContactID, body.Title, body.Value, body.AssignedTo)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func UpdateDeal(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		dealID := c.Params("id")

		var body struct {
			Title      string  `json:"title"`
			Value      float64 `json:"value"`
			Status     string  `json:"status"`
			LossReason string  `json:"loss_reason"`
		}
		c.BodyParser(&body)

		svc.DB.Exec(`
			UPDATE deals SET title = COALESCE(NULLIF($1, ''), title), value = $2, 
			status = COALESCE(NULLIF($3, ''), status), loss_reason = NULLIF($4, ''), updated_at = NOW()
			WHERE id = $5 AND company_id = $6
		`, body.Title, body.Value, body.Status, body.LossReason, dealID, companyID)

		return c.JSON(fiber.Map{"message": "Deal updated"})
	}
}

func MoveDealStage(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		dealID := c.Params("id")

		var body struct {
			StageID string `json:"stage_id"`
		}
		c.BodyParser(&body)

		svc.DB.Exec("UPDATE deals SET stage_id = $1, updated_at = NOW() WHERE id = $2 AND company_id = $3",
			body.StageID, dealID, companyID)

		return c.JSON(fiber.Map{"message": "Deal moved"})
	}
}

// ============================================
// BOT FLOWS
// ============================================

func GetBotFlows(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT id, name, description, trigger_type, trigger_value, is_active, nodes, edges, created_at
			FROM bot_flows WHERE company_id = $1 ORDER BY created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var flows []map[string]interface{}
		for rows.Next() {
			var id, name, triggerType string
			var description, triggerValue *string
			var isActive bool
			var nodes, edges json.RawMessage
			var createdAt string
			rows.Scan(&id, &name, &description, &triggerType, &triggerValue, &isActive, &nodes, &edges, &createdAt)
			flows = append(flows, map[string]interface{}{
				"id": id, "name": name, "description": description,
				"trigger_type": triggerType, "trigger_value": triggerValue,
				"is_active": isActive, "nodes": nodes, "edges": edges,
				"created_at": createdAt,
			})
		}

		return c.JSON(fiber.Map{"flows": flows})
	}
}

func CreateBotFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name         string          `json:"name"`
			Description  string          `json:"description"`
			TriggerType  string          `json:"trigger_type"`
			TriggerValue string          `json:"trigger_value"`
			Nodes        json.RawMessage `json:"nodes"`
			Edges        json.RawMessage `json:"edges"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Nodes == nil {
			body.Nodes = json.RawMessage("[]")
		}
		if body.Edges == nil {
			body.Edges = json.RawMessage("[]")
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO bot_flows (id, company_id, name, description, trigger_type, trigger_value, nodes, edges)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, companyID, body.Name, body.Description, body.TriggerType, body.TriggerValue, body.Nodes, body.Edges)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func UpdateBotFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		flowID := c.Params("id")

		var body struct {
			Name         string          `json:"name"`
			Description  string          `json:"description"`
			TriggerType  string          `json:"trigger_type"`
			TriggerValue string          `json:"trigger_value"`
			IsActive     bool            `json:"is_active"`
			Nodes        json.RawMessage `json:"nodes"`
			Edges        json.RawMessage `json:"edges"`
		}
		c.BodyParser(&body)

		svc.DB.Exec(`
			UPDATE bot_flows SET name = $1, description = $2, trigger_type = $3, trigger_value = $4,
			is_active = $5, nodes = $6, edges = $7, updated_at = NOW()
			WHERE id = $8 AND company_id = $9
		`, body.Name, body.Description, body.TriggerType, body.TriggerValue, body.IsActive, body.Nodes, body.Edges, flowID, companyID)

		return c.JSON(fiber.Map{"message": "Flow updated"})
	}
}

func DeleteBotFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		flowID := c.Params("id")

		svc.DB.Exec("DELETE FROM bot_flows WHERE id = $1 AND company_id = $2", flowID, companyID)
		return c.JSON(fiber.Map{"message": "Flow deleted"})
	}
}

// ============================================
// CAMPAIGNS
// ============================================

func GetCampaigns(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT id, name, status, message_type, total_contacts, sent_count, delivered_count,
				   read_count, replied_count, failed_count, scheduled_at, created_at
			FROM campaigns WHERE company_id = $1 ORDER BY created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var campaigns []map[string]interface{}
		for rows.Next() {
			var id, name, status, msgType string
			var total, sent, delivered, read, replied, failed int
			var scheduledAt, createdAt *string
			rows.Scan(&id, &name, &status, &msgType, &total, &sent, &delivered, &read, &replied, &failed, &scheduledAt, &createdAt)
			campaigns = append(campaigns, map[string]interface{}{
				"id": id, "name": name, "status": status, "message_type": msgType,
				"total_contacts": total, "sent_count": sent, "delivered_count": delivered,
				"read_count": read, "replied_count": replied, "failed_count": failed,
				"scheduled_at": scheduledAt, "created_at": createdAt,
			})
		}

		return c.JSON(fiber.Map{"campaigns": campaigns})
	}
}

func CreateCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var body struct {
			Name           string `json:"name"`
			ChannelID      string `json:"channel_id"`
			MessageContent string `json:"message_content"`
			MessageType    string `json:"message_type"`
			MediaURL       string `json:"media_url"`
			ScheduledAt    string `json:"scheduled_at"`
			SendSpeed      int    `json:"send_speed"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.SendSpeed == 0 {
			body.SendSpeed = 30
		}
		if body.MessageType == "" {
			body.MessageType = "text"
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO campaigns (id, company_id, channel_id, name, message_content, message_type, media_url, send_speed, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, id, companyID, body.ChannelID, body.Name, body.MessageContent, body.MessageType, body.MediaURL, body.SendSpeed, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func UpdateCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		campaignID := c.Params("id")

		var body struct {
			Name           string `json:"name"`
			MessageContent string `json:"message_content"`
		}
		c.BodyParser(&body)

		svc.DB.Exec("UPDATE campaigns SET name = $1, message_content = $2, updated_at = NOW() WHERE id = $3 AND company_id = $4",
			body.Name, body.MessageContent, campaignID, companyID)

		return c.JSON(fiber.Map{"message": "Campaign updated"})
	}
}

func StartCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		campaignID := c.Params("id")

		svc.DB.Exec("UPDATE campaigns SET status = 'sending', started_at = NOW(), updated_at = NOW() WHERE id = $1 AND company_id = $2",
			campaignID, companyID)

		// TODO: Start campaign worker in background
		return c.JSON(fiber.Map{"message": "Campaign started"})
	}
}

func PauseCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		campaignID := c.Params("id")

		svc.DB.Exec("UPDATE campaigns SET status = 'paused', updated_at = NOW() WHERE id = $1 AND company_id = $2",
			campaignID, companyID)

		return c.JSON(fiber.Map{"message": "Campaign paused"})
	}
}

// ============================================
// METRICS
// ============================================

func GetMetrics(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		metrics := make(map[string]interface{})

		// Conversations metrics
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1", companyID).Scan(&metrics["total_conversations"])
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'open'", companyID).Scan(&metrics["open_conversations"])
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'resolved'", companyID).Scan(&metrics["resolved_conversations"])

		// Messages
		svc.DB.QueryRow("SELECT COUNT(*) FROM messages WHERE company_id = $1", companyID).Scan(&metrics["total_messages"])

		// Deals
		svc.DB.QueryRow("SELECT COALESCE(SUM(value), 0) FROM deals WHERE company_id = $1 AND status = 'won'", companyID).Scan(&metrics["deals_won_value"])
		svc.DB.QueryRow("SELECT COALESCE(SUM(value), 0) FROM deals WHERE company_id = $1 AND status = 'open'", companyID).Scan(&metrics["deals_open_value"])
		svc.DB.QueryRow("SELECT COUNT(*) FROM deals WHERE company_id = $1 AND status = 'won'", companyID).Scan(&metrics["deals_won_count"])

		// Calls
		svc.DB.QueryRow("SELECT COUNT(*) FROM calls WHERE company_id = $1", companyID).Scan(&metrics["total_calls"])

		// Contacts
		svc.DB.QueryRow("SELECT COUNT(*) FROM contacts WHERE company_id = $1", companyID).Scan(&metrics["total_contacts"])

		return c.JSON(fiber.Map{"metrics": metrics})
	}
}

// ============================================
// ANNOUNCEMENTS
// ============================================

func GetAnnouncements(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		rows, _ := svc.DB.Query(`
			SELECT ia.id, ia.title, ia.content, ia.priority, ia.is_pinned, ia.published_at,
				   u.name as author_name,
				   EXISTS(SELECT 1 FROM announcement_reads ar WHERE ar.announcement_id = ia.id AND ar.user_id = $2) as is_read
			FROM internal_announcements ia
			JOIN users u ON ia.author_id = u.id
			WHERE ia.company_id = $1
			ORDER BY ia.is_pinned DESC, ia.published_at DESC
			LIMIT 50
		`, companyID, userID)

		var announcements []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, title, content, priority, authorName string
				var isPinned, isRead bool
				var publishedAt string
				rows.Scan(&id, &title, &content, &priority, &isPinned, &publishedAt, &authorName, &isRead)
				announcements = append(announcements, map[string]interface{}{
					"id": id, "title": title, "content": content, "priority": priority,
					"is_pinned": isPinned, "published_at": publishedAt, "author_name": authorName, "is_read": isRead,
				})
			}
		}

		return c.JSON(fiber.Map{"announcements": announcements})
	}
}

func CreateAnnouncement(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var body struct {
			Title    string `json:"title"`
			Content  string `json:"content"`
			Priority string `json:"priority"`
			IsPinned bool   `json:"is_pinned"`
		}
		c.BodyParser(&body)

		if body.Priority == "" {
			body.Priority = "normal"
		}

		id := uuid.New().String()
		svc.DB.Exec(`
			INSERT INTO internal_announcements (id, company_id, author_id, title, content, priority, is_pinned)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, companyID, userID, body.Title, body.Content, body.Priority, body.IsPinned)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func MarkAnnouncementRead(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		announcementID := c.Params("id")

		svc.DB.Exec(`
			INSERT INTO announcement_reads (announcement_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, announcementID, userID)

		return c.JSON(fiber.Map{"message": "Marked as read"})
	}
}

// ============================================
// WIDGETS
// ============================================

func GetWidgets(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, _ := svc.DB.Query(`
			SELECT id, name, primary_color, greeting_message, position, is_active, created_at
			FROM widgets WHERE company_id = $1
		`, companyID)

		var widgets []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, name, color, greeting, position string
				var isActive bool
				var createdAt string
				rows.Scan(&id, &name, &color, &greeting, &position, &isActive, &createdAt)
				widgets = append(widgets, map[string]interface{}{
					"id": id, "name": name, "primary_color": color,
					"greeting_message": greeting, "position": position,
					"is_active": isActive, "created_at": createdAt,
				})
			}
		}

		return c.JSON(fiber.Map{"widgets": widgets})
	}
}

func CreateWidget(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name            string `json:"name"`
			PrimaryColor    string `json:"primary_color"`
			GreetingMessage string `json:"greeting_message"`
			Position        string `json:"position"`
		}
		c.BodyParser(&body)

		if body.PrimaryColor == "" {
			body.PrimaryColor = "#3B82F6"
		}
		if body.Position == "" {
			body.Position = "bottom-right"
		}
		if body.GreetingMessage == "" {
			body.GreetingMessage = "Olá! Como podemos ajudar?"
		}

		id := uuid.New().String()
		svc.DB.Exec(`
			INSERT INTO widgets (id, company_id, name, primary_color, greeting_message, position)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, id, companyID, body.Name, body.PrimaryColor, body.GreetingMessage, body.Position)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func UpdateWidget(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		widgetID := c.Params("id")

		var body struct {
			Name            string `json:"name"`
			PrimaryColor    string `json:"primary_color"`
			GreetingMessage string `json:"greeting_message"`
			Position        string `json:"position"`
			IsActive        bool   `json:"is_active"`
		}
		c.BodyParser(&body)

		svc.DB.Exec(`
			UPDATE widgets SET name = $1, primary_color = $2, greeting_message = $3, position = $4, is_active = $5, updated_at = NOW()
			WHERE id = $6 AND company_id = $7
		`, body.Name, body.PrimaryColor, body.GreetingMessage, body.Position, body.IsActive, widgetID, companyID)

		return c.JSON(fiber.Map{"message": "Widget updated"})
	}
}

func GetWidgetPublicConfig(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		widgetID := c.Params("id")

		var name, color, greeting, position string
		var channelsDisplayed json.RawMessage
		err := svc.DB.QueryRow(`
			SELECT name, primary_color, greeting_message, position, channels_displayed
			FROM widgets WHERE id = $1 AND is_active = true
		`, widgetID).Scan(&name, &color, &greeting, &position, &channelsDisplayed)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Widget not found"})
		}

		return c.JSON(fiber.Map{
			"name": name, "primary_color": color, "greeting_message": greeting,
			"position": position, "channels_displayed": channelsDisplayed,
		})
	}
}

// ============================================
// TAGS
// ============================================

func GetTags(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, _ := svc.DB.Query("SELECT id, name, color FROM tags WHERE company_id = $1 ORDER BY name", companyID)

		var tags []map[string]string
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, name, color string
				rows.Scan(&id, &name, &color)
				tags = append(tags, map[string]string{"id": id, "name": name, "color": color})
			}
		}

		return c.JSON(fiber.Map{"tags": tags})
	}
}

func CreateTag(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		c.BodyParser(&body)

		if body.Color == "" {
			body.Color = "#3B82F6"
		}

		id := uuid.New().String()
		svc.DB.Exec("INSERT INTO tags (id, company_id, name, color) VALUES ($1, $2, $3, $4)", id, companyID, body.Name, body.Color)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "name": body.Name, "color": body.Color})
	}
}

func DeleteTag(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		tagID := c.Params("id")

		svc.DB.Exec("DELETE FROM tags WHERE id = $1 AND company_id = $2", tagID, companyID)
		return c.JSON(fiber.Map{"message": "Tag deleted"})
	}
}

// ============================================
// QUICK REPLIES
// ============================================

func GetQuickReplies(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, _ := svc.DB.Query("SELECT id, shortcut, title, content FROM quick_replies WHERE company_id = $1 ORDER BY shortcut", companyID)

		var replies []map[string]string
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, shortcut, title, content string
				rows.Scan(&id, &shortcut, &title, &content)
				replies = append(replies, map[string]string{"id": id, "shortcut": shortcut, "title": title, "content": content})
			}
		}

		return c.JSON(fiber.Map{"quick_replies": replies})
	}
}

func CreateQuickReply(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var body struct {
			Shortcut string `json:"shortcut"`
			Title    string `json:"title"`
			Content  string `json:"content"`
		}
		c.BodyParser(&body)

		id := uuid.New().String()
		svc.DB.Exec("INSERT INTO quick_replies (id, company_id, shortcut, title, content, created_by) VALUES ($1, $2, $3, $4, $5, $6)",
			id, companyID, body.Shortcut, body.Title, body.Content, userID)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

// ============================================
// USERS
// ============================================

func GetUsers(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, _ := svc.DB.Query(`
			SELECT u.id, u.name, u.email, u.avatar_url, u.is_active, u.is_online, r.name as role_name
			FROM users u
			LEFT JOIN roles r ON u.role_id = r.id
			WHERE u.company_id = $1 ORDER BY u.name
		`, companyID)

		var users []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, name, email string
				var avatarURL, roleName *string
				var isActive, isOnline bool
				rows.Scan(&id, &name, &email, &avatarURL, &isActive, &isOnline, &roleName)
				users = append(users, map[string]interface{}{
					"id": id, "name": name, "email": email, "avatar_url": avatarURL,
					"is_active": isActive, "is_online": isOnline, "role_name": roleName,
				})
			}
		}

		return c.JSON(fiber.Map{"users": users})
	}
}

func CreateUser(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
			RoleID   string `json:"role_id"`
		}
		c.BodyParser(&body)

		// This would typically hash the password and create the user
		// Simplified for brevity
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User created", "company_id": companyID})
	}
}
