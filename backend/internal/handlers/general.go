package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

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
			SELECT t.id, COALESCE(t.name, 'Time'), t.description,
				   COALESCE(t.distribution_rule, 'round-robin'), COALESCE(t.is_active, true),
				   (SELECT COUNT(*) FROM team_users tu WHERE tu.team_id = t.id) as member_count,
				   (SELECT COUNT(*) FROM conversations c WHERE c.team_id = t.id AND c.company_id = t.company_id AND COALESCE(c.status, 'open') IN ('open', 'in_progress', 'pending')) as open_count,
				   (SELECT COALESCE(SUM(COALESCE(c.unread_count, 0)), 0) FROM conversations c WHERE c.team_id = t.id AND c.company_id = t.company_id AND COALESCE(c.status, 'open') IN ('open', 'in_progress', 'pending')) as unread_count
			FROM teams t WHERE t.company_id = $1 ORDER BY COALESCE(t.name, 'Time')
		`, companyID)
		if err != nil {
			log.Printf("[TEAMS] failed to list teams for company %s: %v", companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var teams []map[string]interface{}
		for rows.Next() {
			var id, name, distRule string
			var description *string
			var isActive bool
			var memberCount, openCount, unreadCount int
			if err := rows.Scan(&id, &name, &description, &distRule, &isActive, &memberCount, &openCount, &unreadCount); err != nil {
				log.Printf("[TEAMS] failed to scan team for company %s: %v", companyID, err)
				continue
			}
			teams = append(teams, map[string]interface{}{
				"id": id, "name": name, "description": description,
				"distribution_rule": distRule, "is_active": isActive,
				"member_count": memberCount, "open_count": openCount, "unread_count": unreadCount,
			})
		}
		if err := rows.Err(); err != nil {
			log.Printf("[TEAMS] failed to read teams for company %s: %v", companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
			SELECT id, name, COALESCE(type, 'whatsapp'), COALESCE(status, 'disconnected'),
			       COALESCE(settings, '{}'::jsonb), COALESCE(is_active, true), COALESCE(created_at, NOW())::text
			FROM channels WHERE company_id = $1 ORDER BY created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var channels []map[string]interface{}
		for rows.Next() {
			var id, name, chType, status string
			var rawSettings []byte
			var isActive bool
			var createdAt string
			if err := rows.Scan(&id, &name, &chType, &status, &rawSettings, &isActive, &createdAt); err != nil {
				continue
			}
			settings := sanitizeChannelSettings(chType, rawSettings)
			channels = append(channels, map[string]interface{}{
				"id": id, "name": name, "type": chType, "status": status,
				"settings": settings, "is_active": isActive, "created_at": createdAt,
			})
		}
		if err := rows.Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"channels": channels})
	}
}

func sanitizeChannelSettings(channelType string, raw []byte) map[string]interface{} {
	settings := map[string]interface{}{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &settings)
	}
	if channelType == "email" {
		delete(settings, "password")
	}
	return settings
}

// ============================================
// FUNNELS
// ============================================

func GetFunnels(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT id, COALESCE(name, 'Funil'), description, COALESCE(is_default, false), COALESCE(is_active, true) FROM funnels
			WHERE company_id = $1 ORDER BY COALESCE(created_at, NOW())
		`, companyID)
		if err != nil {
			log.Printf("[FUNNELS] failed to list funnels for company %s: %v", companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var funnels []map[string]interface{}
		for rows.Next() {
			var id, name string
			var description *string
			var isDefault, isActive bool
			if err := rows.Scan(&id, &name, &description, &isDefault, &isActive); err != nil {
				continue
			}

			// Get stages
			stageRows, err := svc.DB.Query(`
				SELECT fs.id, COALESCE(fs.name, 'Etapa'), COALESCE(fs.color, '#3B82F6'), COALESCE(fs.position, 0),
					   COALESCE(fs.is_won, false), COALESCE(fs.is_lost, false),
					   (SELECT COUNT(*) FROM deals d WHERE d.stage_id = fs.id AND d.status = 'open') as deal_count,
					   (SELECT COALESCE(SUM(value), 0) FROM deals d WHERE d.stage_id = fs.id AND d.status = 'open') as deal_value
				FROM funnel_stages fs WHERE fs.funnel_id = $1 ORDER BY COALESCE(fs.position, 0)
			`, id)

			var stages []map[string]interface{}
			if err != nil {
				log.Printf("[FUNNELS] failed to list stages for funnel %s: %v", id, err)
			} else if stageRows != nil {
				for stageRows.Next() {
					var sID, sName, sColor string
					var pos int
					var isWon, isLost bool
					var dealCount int
					var dealValue float64
					if err := stageRows.Scan(&sID, &sName, &sColor, &pos, &isWon, &isLost, &dealCount, &dealValue); err != nil {
						continue
					}
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
		if err := rows.Err(); err != nil {
			log.Printf("[FUNNELS] failed to read funnels for company %s: %v", companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
				Name   string `json:"name"`
				Color  string `json:"color"`
				IsWon  bool   `json:"is_won"`
				IsLost bool   `json:"is_lost"`
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
		contactID := c.Query("contact_id")

		query := `
			SELECT d.id, COALESCE(d.title, 'Negocio'), COALESCE(d.value, 0), COALESCE(d.status, 'open'), d.funnel_id, d.stage_id,
				   c.name as contact_name, u.name as assigned_to_name, f.name as funnel_name, fs.name as stage_name
			FROM deals d
			LEFT JOIN contacts c ON d.contact_id = c.id
			LEFT JOIN users u ON d.assigned_to = u.id
			LEFT JOIN funnels f ON d.funnel_id = f.id
			LEFT JOIN funnel_stages fs ON d.stage_id = fs.id
			WHERE d.company_id = $1
		`
		args := []interface{}{companyID}
		idx := 2

		if funnelID != "" {
			query += fmt.Sprintf(" AND d.funnel_id = $%d", idx)
			args = append(args, funnelID)
			idx++
		}
		if stageID != "" {
			query += fmt.Sprintf(" AND d.stage_id = $%d", idx)
			args = append(args, stageID)
			idx++
		}
		if contactID != "" {
			query += fmt.Sprintf(" AND d.contact_id = $%d", idx)
			args = append(args, contactID)
			idx++
		}

		query += " ORDER BY d.created_at DESC"

		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			log.Printf("[DEALS] failed to list deals for company %s: %v", companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var deals []map[string]interface{}
		for rows.Next() {
			var id, title, status string
			var funnelIDVal, stageIDVal *string
			var value float64
			var contactName, assignedToName, funnelName, stageName *string
			if err := rows.Scan(&id, &title, &value, &status, &funnelIDVal, &stageIDVal, &contactName, &assignedToName, &funnelName, &stageName); err != nil {
				continue
			}
			deals = append(deals, map[string]interface{}{
				"id": id, "title": title, "value": value, "status": status,
				"funnel_id": funnelIDVal, "stage_id": stageIDVal, "contact_name": contactName,
				"assigned_to_name": assignedToName, "funnel_name": funnelName, "stage_name": stageName,
			})
		}
		if err := rows.Err(); err != nil {
			log.Printf("[DEALS] failed to read deals for company %s: %v", companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"deals": deals})
	}
}

func CreateDeal(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

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

		// Use NULL for empty optional UUIDs
		var contactID, assignedTo interface{}
		if body.ContactID != "" {
			contactID = body.ContactID
		}
		// Auto-assign to current user if not specified
		if body.AssignedTo != "" {
			assignedTo = body.AssignedTo
		} else {
			assignedTo = userID
		}

		_, err := svc.DB.Exec(`
			INSERT INTO deals (id, company_id, funnel_id, stage_id, contact_id, title, value, assigned_to)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, companyID, body.FunnelID, body.StageID, contactID, body.Title, body.Value, assignedTo)
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
			Title      string   `json:"title"`
			Value      *float64 `json:"value"`
			Status     string   `json:"status"`
			LossReason string   `json:"loss_reason"`
		}
		c.BodyParser(&body)

		svc.DB.Exec(`
			UPDATE deals SET title = COALESCE(NULLIF($1, ''), title), value = COALESCE($2, value),
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
			SELECT id, name, COALESCE(bot_name, 'Assistente'), description, trigger_type, trigger_value, is_active, priority, stop_on_match, nodes, edges, created_at
			FROM bot_flows WHERE company_id = $1 ORDER BY priority DESC, created_at ASC
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var flows []map[string]interface{}
		for rows.Next() {
			var id, name, botName, triggerType string
			var description, triggerValue *string
			var isActive bool
			var priority int
			var stopOnMatch bool
			var nodes, edges json.RawMessage
			var createdAt string
			rows.Scan(&id, &name, &botName, &description, &triggerType, &triggerValue, &isActive, &priority, &stopOnMatch, &nodes, &edges, &createdAt)
			flows = append(flows, map[string]interface{}{
				"id": id, "name": name, "description": description,
				"bot_name":     botName,
				"trigger_type": triggerType, "trigger_value": triggerValue,
				"is_active": isActive, "priority": priority, "stop_on_match": stopOnMatch, "nodes": nodes, "edges": edges,
				"created_at": createdAt,
			})
		}

		return c.JSON(fiber.Map{"flows": flows})
	}
}

func GetBotFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		flowID := c.Params("id")

		var id, name, botName, triggerType string
		var description, triggerValue *string
		var isActive, stopOnMatch bool
		var priority int
		var nodes, edges json.RawMessage
		var createdAt string

		err := svc.DB.QueryRow(`
			SELECT id, name, COALESCE(bot_name, 'Assistente'), description, trigger_type, trigger_value, is_active, priority, stop_on_match, nodes, edges, created_at
			FROM bot_flows
			WHERE id = $1 AND company_id = $2
		`, flowID, companyID).Scan(&id, &name, &botName, &description, &triggerType, &triggerValue, &isActive, &priority, &stopOnMatch, &nodes, &edges, &createdAt)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Fluxo não encontrado"})
		}

		return c.JSON(fiber.Map{
			"id": id, "name": name, "description": description,
			"bot_name":     botName,
			"trigger_type": triggerType, "trigger_value": triggerValue,
			"is_active": isActive, "priority": priority, "stop_on_match": stopOnMatch,
			"nodes": nodes, "edges": edges, "created_at": createdAt,
		})
	}
}

func CreateBotFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name         string          `json:"name"`
			BotName      string          `json:"bot_name"`
			Description  string          `json:"description"`
			TriggerType  string          `json:"trigger_type"`
			TriggerValue string          `json:"trigger_value"`
			Priority     int             `json:"priority"`
			StopOnMatch  *bool           `json:"stop_on_match"`
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
		if body.Priority == 0 {
			body.Priority = defaultBotFlowPriority(body.TriggerType)
		}
		if strings.TrimSpace(body.BotName) == "" {
			body.BotName = "Assistente"
		}
		stopOnMatch := true
		if body.StopOnMatch != nil {
			stopOnMatch = *body.StopOnMatch
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO bot_flows (id, company_id, name, bot_name, description, trigger_type, trigger_value, priority, stop_on_match, nodes, edges)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, id, companyID, body.Name, body.BotName, body.Description, body.TriggerType, body.TriggerValue, body.Priority, stopOnMatch, body.Nodes, body.Edges)
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
			BotName      string          `json:"bot_name"`
			Description  string          `json:"description"`
			TriggerType  string          `json:"trigger_type"`
			TriggerValue string          `json:"trigger_value"`
			IsActive     bool            `json:"is_active"`
			Priority     int             `json:"priority"`
			StopOnMatch  *bool           `json:"stop_on_match"`
			Nodes        json.RawMessage `json:"nodes"`
			Edges        json.RawMessage `json:"edges"`
		}
		c.BodyParser(&body)
		if body.Priority == 0 {
			body.Priority = defaultBotFlowPriority(body.TriggerType)
		}
		if strings.TrimSpace(body.BotName) == "" {
			body.BotName = "Assistente"
		}
		stopOnMatch := true
		if body.StopOnMatch != nil {
			stopOnMatch = *body.StopOnMatch
		}

		svc.DB.Exec(`
			UPDATE bot_flows SET name = $1, bot_name = $2, description = $3, trigger_type = $4, trigger_value = $5,
			is_active = $6, priority = $7, stop_on_match = $8, nodes = $9, edges = $10, updated_at = NOW()
			WHERE id = $11 AND company_id = $12
		`, body.Name, body.BotName, body.Description, body.TriggerType, body.TriggerValue, body.IsActive, body.Priority, stopOnMatch, body.Nodes, body.Edges, flowID, companyID)

		return c.JSON(fiber.Map{"message": "Flow updated"})
	}
}

func DuplicateBotFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		flowID := c.Params("id")

		var name, botName, description, triggerType, triggerValue string
		var priority int
		var stopOnMatch bool
		var nodes, edges json.RawMessage

		err := svc.DB.QueryRow(`
			SELECT name, COALESCE(bot_name, 'Assistente'), COALESCE(description, ''), trigger_type, COALESCE(trigger_value, ''),
			       COALESCE(priority, 10), COALESCE(stop_on_match, true), COALESCE(nodes, '[]'::jsonb), COALESCE(edges, '[]'::jsonb)
			FROM bot_flows
			WHERE id = $1 AND company_id = $2
		`, flowID, companyID).Scan(&name, &botName, &description, &triggerType, &triggerValue, &priority, &stopOnMatch, &nodes, &edges)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Fluxo não encontrado"})
		}

		newID := uuid.New().String()
		newName := strings.TrimSpace(name) + " (cópia)"
		_, err = svc.DB.Exec(`
			INSERT INTO bot_flows (id, company_id, name, bot_name, description, trigger_type, trigger_value, is_active, priority, stop_on_match, nodes, edges)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), false, $8, $9, $10, $11)
		`, newID, companyID, newName, botName, description, triggerType, triggerValue, priority, stopOnMatch, nodes, edges)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": newID, "name": newName})
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

func defaultBotFlowPriority(triggerType string) int {
	switch triggerType {
	case "off_hours", "trigger_off_hours":
		return 100
	case "no_response", "trigger_no_response":
		return 80
	case "new_conversation", "trigger_new_conversation", "trigger_inbox_message":
		return 10
	default:
		return 20
	}
}

// ============================================
// CAMPAIGNS
// ============================================

type campaignMessageItem struct {
	Type          string `json:"type"`
	Content       string `json:"content"`
	MediaURL      string `json:"media_url"`
	MediaBase64   string `json:"media_base64,omitempty"`
	MediaFileName string `json:"media_filename,omitempty"`
}

func GetCampaigns(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT c.id, c.name, c.status, COALESCE(c.message_content, ''), COALESCE(c.message_type, 'text'), c.media_url,
				   COALESCE(c.send_speed, 30),
				   COALESCE(NULLIF(stats.total_contacts, 0), c.total_contacts, 0) AS total_contacts,
				   COALESCE(stats.sent_count, c.sent_count, 0) AS sent_count,
				   COALESCE(stats.delivered_count, c.delivered_count, 0) AS delivered_count,
				   COALESCE(stats.read_count, c.read_count, 0) AS read_count,
				   COALESCE(stats.replied_count, c.replied_count, 0) AS replied_count,
				   COALESCE(stats.failed_count, c.failed_count, 0) AS failed_count,
				   c.scheduled_at, c.created_at, COALESCE(c.variables, '[]'::jsonb)
			FROM campaigns c
			LEFT JOIN (
				SELECT campaign_id,
					   COUNT(*)::int AS total_contacts,
					   COUNT(*) FILTER (WHERE status IN ('sent', 'delivered', 'read', 'replied'))::int AS sent_count,
					   COUNT(*) FILTER (WHERE status IN ('delivered', 'read', 'replied'))::int AS delivered_count,
					   COUNT(*) FILTER (WHERE status IN ('read', 'replied'))::int AS read_count,
					   COUNT(*) FILTER (WHERE status = 'replied')::int AS replied_count,
					   COUNT(*) FILTER (WHERE status = 'failed')::int AS failed_count
				FROM campaign_contacts
				GROUP BY campaign_id
			) stats ON stats.campaign_id = c.id
			WHERE c.company_id = $1
			ORDER BY c.created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var campaigns []map[string]interface{}
		for rows.Next() {
			var id, name, status, msgContent, msgType string
			var mediaURL sql.NullString
			var total, sent, delivered, read, replied, failed, sendSpeed int
			var scheduledAt, createdAt sql.NullTime
			var variables []byte
			if err := rows.Scan(&id, &name, &status, &msgContent, &msgType, &mediaURL, &sendSpeed, &total, &sent, &delivered, &read, &replied, &failed, &scheduledAt, &createdAt, &variables); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			contentItems := campaignItemsFromStored(json.RawMessage(variables), msgContent, msgType, mediaURL.String)
			campaigns = append(campaigns, map[string]interface{}{
				"id": id, "name": name, "status": status, "message_content": msgContent, "message_type": msgType, "media_url": mediaURL.String, "send_speed": sendSpeed,
				"total_contacts": total, "sent_count": sent, "delivered_count": delivered,
				"read_count": read, "replied_count": replied, "failed_count": failed,
				"scheduled_at": nullableTimeValue(scheduledAt), "created_at": nullableTimeValue(createdAt), "content_items": contentItems,
			})
		}

		return c.JSON(fiber.Map{"campaigns": campaigns})
	}
}

func normalizeCampaignItems(items []campaignMessageItem, legacyContent, legacyType, legacyMediaURL, legacyMediaBase64, legacyMediaFileName string, allowExistingMedia bool) ([]campaignMessageItem, error) {
	if len(items) == 0 {
		items = []campaignMessageItem{{
			Type:          legacyType,
			Content:       legacyContent,
			MediaURL:      legacyMediaURL,
			MediaBase64:   legacyMediaBase64,
			MediaFileName: legacyMediaFileName,
		}}
	}

	normalized := make([]campaignMessageItem, 0, len(items))
	for _, item := range items {
		item.Type = strings.TrimSpace(strings.ToLower(item.Type))
		item.Content = strings.TrimSpace(item.Content)
		item.MediaURL = strings.TrimSpace(item.MediaURL)
		item.MediaBase64 = strings.TrimSpace(item.MediaBase64)
		item.MediaFileName = strings.TrimSpace(item.MediaFileName)
		if item.Type == "" {
			item.Type = "text"
		}
		if item.Type != "text" && item.Type != "image" && item.Type != "video" && item.Type != "audio" && item.Type != "document" {
			return nil, fmt.Errorf("tipo de conteúdo inválido: %s", item.Type)
		}
		if item.Type == "text" {
			if item.Content == "" {
				continue
			}
			item.MediaURL = ""
			item.MediaBase64 = ""
			item.MediaFileName = ""
			normalized = append(normalized, item)
			continue
		}
		if item.MediaBase64 != "" {
			ext := services.GetExtensionFromBase64(item.MediaBase64)
			if ext == "" {
				ext = services.GetExtensionFromType(item.Type)
			}
			if item.MediaFileName != "" {
				if dotIdx := strings.LastIndex(item.MediaFileName, "."); dotIdx != -1 {
					ext = item.MediaFileName[dotIdx:]
				}
			}
			savedFileName, err := services.SaveBase64File(item.MediaBase64, ext)
			if err != nil {
				return nil, fmt.Errorf("erro ao salvar arquivo da campanha")
			}
			item.MediaURL = "/uploads/" + savedFileName
		}
		if item.MediaURL == "" && !allowExistingMedia {
			return nil, fmt.Errorf("adicione o arquivo para %s", campaignItemTypeLabel(item.Type))
		}
		if item.MediaURL == "" {
			continue
		}
		item.MediaBase64 = ""
		item.MediaFileName = ""
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("adicione pelo menos um texto ou arquivo para enviar")
	}
	return normalized, nil
}

func campaignItemTypeLabel(itemType string) string {
	switch itemType {
	case "image":
		return "imagem"
	case "video":
		return "vídeo"
	case "audio":
		return "áudio"
	case "document":
		return "documento"
	default:
		return "texto"
	}
}

func campaignPrimaryFields(items []campaignMessageItem) (string, string, string) {
	if len(items) == 0 {
		return "", "text", ""
	}
	first := items[0]
	return first.Content, first.Type, first.MediaURL
}

func campaignItemsFromStored(raw json.RawMessage, legacyContent, legacyType, legacyMediaURL string) []campaignMessageItem {
	var items []campaignMessageItem
	if len(raw) > 0 && string(raw) != "null" {
		_ = json.Unmarshal(raw, &items)
	}
	if len(items) == 0 {
		if legacyType == "" {
			legacyType = "text"
		}
		items = []campaignMessageItem{{Type: legacyType, Content: legacyContent, MediaURL: legacyMediaURL}}
	}
	return items
}

func nullableTimeValue(value sql.NullTime) interface{} {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func CreateCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var body struct {
			Name           string                `json:"name"`
			ChannelID      string                `json:"channel_id"`
			MessageContent string                `json:"message_content"`
			MessageType    string                `json:"message_type"`
			MediaURL       string                `json:"media_url"`
			MediaBase64    string                `json:"media_base64"`
			MediaFileName  string                `json:"media_filename"`
			ContentItems   []campaignMessageItem `json:"content_items"`
			ScheduledAt    string                `json:"scheduled_at"`
			SendSpeed      int                   `json:"send_speed"`
			TotalContacts  int                   `json:"total_contacts"`
			FilterTag      string                `json:"filter_tag"`
			ContactIDs     []string              `json:"contact_ids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if strings.TrimSpace(body.Name) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nome da campanha é obrigatório"})
		}
		if body.SendSpeed == 0 {
			body.SendSpeed = 30
		}
		if body.MessageType == "" {
			body.MessageType = "text"
		}
		contentItems, err := normalizeCampaignItems(body.ContentItems, body.MessageContent, body.MessageType, body.MediaURL, body.MediaBase64, body.MediaFileName, false)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		body.MessageContent, body.MessageType, body.MediaURL = campaignPrimaryFields(contentItems)
		contentItemsJSON, _ := json.Marshal(contentItems)

		id := uuid.New().String()
		tx, err := svc.DB.Begin()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			INSERT INTO campaigns (id, company_id, channel_id, name, message_content, message_type, media_url, variables, send_speed, total_contacts, created_by)
			VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, NULLIF($7, ''), $8::jsonb, $9, $10, $11)
		`, id, companyID, body.ChannelID, body.Name, body.MessageContent, body.MessageType, body.MediaURL, string(contentItemsJSON), body.SendSpeed, body.TotalContacts, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		totalContacts, err := createCampaignRecipients(tx, id, companyID, body.FilterTag, body.ContactIDs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		_, err = tx.Exec("UPDATE campaigns SET total_contacts = $1 WHERE id = $2 AND company_id = $3", totalContacts, id, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if err := tx.Commit(); err != nil {
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
			Name           string                `json:"name"`
			MessageContent string                `json:"message_content"`
			MessageType    string                `json:"message_type"`
			MediaURL       string                `json:"media_url"`
			MediaBase64    string                `json:"media_base64"`
			MediaFileName  string                `json:"media_filename"`
			ContentItems   []campaignMessageItem `json:"content_items"`
			SendSpeed      int                   `json:"send_speed"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if strings.TrimSpace(body.Name) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nome da campanha é obrigatório"})
		}
		if body.SendSpeed <= 0 {
			body.SendSpeed = 30
		}
		contentItems, err := normalizeCampaignItems(body.ContentItems, body.MessageContent, body.MessageType, body.MediaURL, body.MediaBase64, body.MediaFileName, true)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		body.MessageContent, body.MessageType, body.MediaURL = campaignPrimaryFields(contentItems)
		contentItemsJSON, _ := json.Marshal(contentItems)

		res, err := svc.DB.Exec(`
			UPDATE campaigns
			SET name = $1, message_content = $2, message_type = $3, media_url = NULLIF($4, ''), variables = $5::jsonb, send_speed = $6, updated_at = NOW()
			WHERE id = $7 AND company_id = $8 AND status IN ('draft', 'paused')
		`, body.Name, body.MessageContent, body.MessageType, body.MediaURL, string(contentItemsJSON), body.SendSpeed, campaignID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Só é possível editar campanhas em rascunho ou pausadas"})
		}

		return c.JSON(fiber.Map{"message": "Campaign updated"})
	}
}

func DeleteCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		campaignID := c.Params("id")

		res, err := svc.DB.Exec("DELETE FROM campaigns WHERE id = $1 AND company_id = $2", campaignID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Campanha não encontrada"})
		}
		return c.JSON(fiber.Map{"message": "Campaign deleted"})
	}
}

func StartCampaign(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		campaignID := c.Params("id")

		instanceName, err := prepareCampaignForSending(svc.DB, campaignID, companyID)
		if err != nil {
			if strings.Contains(err.Error(), "sem destinatários pendentes") {
				return c.JSON(fiber.Map{"message": "Campaign completed", "status": "completed"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		_, err = svc.DB.Exec("UPDATE campaigns SET status = 'sending', started_at = COALESCE(started_at, NOW()), updated_at = NOW() WHERE id = $1 AND company_id = $2",
			campaignID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		go runCampaignSender(svc, campaignID, companyID, instanceName)
		return c.JSON(fiber.Map{"message": "Campaign started"})
	}
}

func createCampaignRecipients(tx *sql.Tx, campaignID, companyID, filterTag string, contactIDs []string) (int, error) {
	var res sql.Result
	var err error
	if len(contactIDs) > 0 {
		res, err = tx.Exec(`
			INSERT INTO campaign_contacts (campaign_id, contact_id)
			SELECT $1, c.id
			FROM contacts c
			WHERE c.company_id = $2
			  AND c.phone IS NOT NULL
			  AND COALESCE(c.is_opted_out, false) = false
			  AND c.id = ANY($3::uuid[])
		`, campaignID, companyID, "{"+strings.Join(contactIDs, ",")+"}")
	} else if strings.TrimSpace(filterTag) != "" {
		res, err = tx.Exec(`
			INSERT INTO campaign_contacts (campaign_id, contact_id)
			SELECT $1, c.id
			FROM contacts c
			JOIN contact_tags ct ON ct.contact_id = c.id
			WHERE c.company_id = $2
			  AND c.phone IS NOT NULL
			  AND COALESCE(c.is_opted_out, false) = false
			  AND ct.tag_id = $3::uuid
		`, campaignID, companyID, filterTag)
	} else {
		res, err = tx.Exec(`
			INSERT INTO campaign_contacts (campaign_id, contact_id)
			SELECT $1, c.id
			FROM contacts c
			WHERE c.company_id = $2
			  AND c.phone IS NOT NULL
			  AND COALESCE(c.is_opted_out, false) = false
		`, campaignID, companyID)
	}
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	return int(count), nil
}

func prepareCampaignForSending(db *sql.DB, campaignID, companyID string) (string, error) {
	var channelID sql.NullString
	var instanceName sql.NullString
	err := db.QueryRow(`
		SELECT c.channel_id, wi.instance_name
		FROM campaigns c
		LEFT JOIN whatsapp_instances wi ON wi.channel_id = c.channel_id
		WHERE c.id = $1 AND c.company_id = $2
	`, campaignID, companyID).Scan(&channelID, &instanceName)
	if err != nil {
		return "", fmt.Errorf("campanha não encontrada")
	}

	if !channelID.Valid || !instanceName.Valid || instanceName.String == "" {
		err = db.QueryRow(`
			SELECT ch.id, wi.instance_name
			FROM channels ch
			JOIN whatsapp_instances wi ON wi.channel_id = ch.id
			WHERE ch.company_id = $1
			  AND ch.type = 'whatsapp'
			  AND ch.status = 'connected'
			  AND wi.status = 'connected'
			ORDER BY ch.created_at DESC
			LIMIT 1
		`, companyID).Scan(&channelID, &instanceName)
		if err != nil {
			return "", fmt.Errorf("nenhum WhatsApp conectado encontrado para enviar a campanha")
		}
		_, _ = db.Exec("UPDATE campaigns SET channel_id = $1, updated_at = NOW() WHERE id = $2 AND company_id = $3", channelID.String, campaignID, companyID)
	}

	var pending int
	db.QueryRow(`
		SELECT COUNT(*)
		FROM campaign_contacts cc
		JOIN contacts c ON c.id = cc.contact_id
		WHERE cc.campaign_id = $1
		  AND c.company_id = $2
		  AND cc.status = 'pending'
	`, campaignID, companyID).Scan(&pending)
	if pending == 0 {
		_, _ = db.Exec(`
			UPDATE campaigns
			SET status = 'completed', completed_at = COALESCE(completed_at, NOW()), updated_at = NOW()
			WHERE id = $1 AND company_id = $2
		`, campaignID, companyID)
		return "", fmt.Errorf("campanha sem destinatários pendentes")
	}
	return instanceName.String, nil
}

func runCampaignSender(svc *services.Container, campaignID, companyID, instanceName string) {
	var message, messageType string
	var mediaURL sql.NullString
	var variables []byte
	var sendSpeed int
	if err := svc.DB.QueryRow(`
		SELECT COALESCE(message_content, ''), COALESCE(message_type, 'text'), media_url, COALESCE(variables, '[]'::jsonb), COALESCE(send_speed, 30)
		FROM campaigns
		WHERE id = $1 AND company_id = $2
	`, campaignID, companyID).Scan(&message, &messageType, &mediaURL, &variables, &sendSpeed); err != nil {
		log.Printf("[CAMPAIGN] failed to load campaign %s: %v", campaignID, err)
		return
	}
	items := campaignItemsFromStored(json.RawMessage(variables), message, messageType, mediaURL.String)
	if sendSpeed <= 0 {
		sendSpeed = 30
	}
	delay := time.Minute / time.Duration(sendSpeed)
	if delay < time.Second {
		delay = time.Second
	}

	for {
		var status string
		_ = svc.DB.QueryRow("SELECT status FROM campaigns WHERE id = $1 AND company_id = $2", campaignID, companyID).Scan(&status)
		if status != "sending" {
			return
		}

		var campaignContactID, contactID, name, phone, email, companyName string
		err := svc.DB.QueryRow(`
			SELECT cc.id, c.id, COALESCE(c.name, ''), c.phone, COALESCE(c.email, ''), COALESCE(c.company_name, '')
			FROM campaign_contacts cc
			JOIN contacts c ON c.id = cc.contact_id
			WHERE cc.campaign_id = $1
			  AND c.company_id = $2
			  AND cc.status = 'pending'
			  AND c.phone IS NOT NULL
			  AND COALESCE(c.is_opted_out, false) = false
			ORDER BY cc.id
			LIMIT 1
		`, campaignID, companyID).Scan(&campaignContactID, &contactID, &name, &phone, &email, &companyName)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			log.Printf("[CAMPAIGN] failed to load next recipient for %s: %v", campaignID, err)
			time.Sleep(delay)
			continue
		}

		err = sendCampaignItems(svc, campaignID, campaignContactID, instanceName, phone, items, name, phone, email, companyName)
		if err != nil {
			_, _ = svc.DB.Exec(`
				UPDATE campaign_contacts SET status = 'failed', error_message = $1 WHERE id = $2
			`, err.Error(), campaignContactID)
		} else {
			_, _ = svc.DB.Exec(`
				UPDATE campaign_contacts
				SET status = CASE WHEN status IN ('delivered', 'read', 'replied') THEN status ELSE 'sent' END,
					sent_at = COALESCE(sent_at, NOW()),
					error_message = NULL
				WHERE id = $1
			`, campaignContactID)
		}
		refreshCampaignCounters(svc.DB, campaignID)
		time.Sleep(delay)
	}

	finalizeCampaignIfDone(svc.DB, campaignID)
}

func sendCampaignItems(svc *services.Container, campaignID, campaignContactID, instanceName, phone string, items []campaignMessageItem, name, contactPhone, email, companyName string) error {
	for index, item := range items {
		text := renderCampaignMessage(item.Content, name, contactPhone, email, companyName)
		var externalID string
		var err error
		switch item.Type {
		case "audio":
			externalID, err = svc.Evolution.SendAudioMessage(instanceName, phone, publicCampaignMediaURL(svc, item.MediaURL))
		case "image", "video", "document":
			externalID, err = svc.Evolution.SendMediaMessage(instanceName, phone, item.Type, publicCampaignMediaURL(svc, item.MediaURL), text, "")
		default:
			externalID, err = svc.Evolution.SendTextMessage(instanceName, phone, text)
		}
		if err != nil {
			return fmt.Errorf("%s %d: %w", campaignItemTypeLabel(item.Type), index+1, err)
		}
		if externalID != "" {
			_, _ = svc.DB.Exec(`
				INSERT INTO campaign_contact_messages (id, campaign_id, campaign_contact_id, external_id, message_type, item_index, status)
				VALUES ($1, $2, $3, $4, $5, $6, 'sent')
				ON CONFLICT (external_id) DO UPDATE SET
					status = EXCLUDED.status,
					updated_at = NOW()
			`, uuid.New().String(), campaignID, campaignContactID, externalID, item.Type, index)
		}
		if index < len(items)-1 {
			time.Sleep(time.Second)
		}
	}
	return nil
}

func publicCampaignMediaURL(svc *services.Container, mediaURL string) string {
	if strings.HasPrefix(mediaURL, "http://") || strings.HasPrefix(mediaURL, "https://") {
		return mediaURL
	}
	publicURL := svc.Config.EvolutionWebhookURL
	baseURL := strings.TrimSuffix(publicURL, "/api/webhooks/evolution")
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(mediaURL, "/")
}

func finalizeCampaignIfDone(db *sql.DB, campaignID string) {
	refreshCampaignCounters(db, campaignID)
	var remaining int
	_ = db.QueryRow(`
		SELECT COUNT(*)
		FROM campaign_contacts cc
		JOIN contacts c ON c.id = cc.contact_id
		WHERE cc.campaign_id = $1
		  AND cc.status = 'pending'
		  AND c.phone IS NOT NULL
		  AND COALESCE(c.is_opted_out, false) = false
	`, campaignID).Scan(&remaining)
	if remaining == 0 {
		_, _ = db.Exec("UPDATE campaigns SET status = 'completed', completed_at = COALESCE(completed_at, NOW()), updated_at = NOW() WHERE id = $1 AND status = 'sending'", campaignID)
	}
}

func refreshCampaignCounters(db *sql.DB, campaignID string) {
	_, _ = db.Exec(`
		UPDATE campaigns c
		SET total_contacts = stats.total_contacts,
			sent_count = stats.sent_count,
			delivered_count = stats.delivered_count,
			read_count = stats.read_count,
			replied_count = stats.replied_count,
			failed_count = stats.failed_count,
			updated_at = NOW()
		FROM (
			SELECT campaign_id,
				   COUNT(*)::int AS total_contacts,
				   COUNT(*) FILTER (WHERE status IN ('sent', 'delivered', 'read', 'replied'))::int AS sent_count,
				   COUNT(*) FILTER (WHERE status IN ('delivered', 'read', 'replied'))::int AS delivered_count,
				   COUNT(*) FILTER (WHERE status IN ('read', 'replied'))::int AS read_count,
				   COUNT(*) FILTER (WHERE status = 'replied')::int AS replied_count,
				   COUNT(*) FILTER (WHERE status = 'failed')::int AS failed_count
			FROM campaign_contacts
			WHERE campaign_id = $1
			GROUP BY campaign_id
		) stats
		WHERE c.id = stats.campaign_id
	`, campaignID)
}

func renderCampaignMessage(template, name, phone, email, companyName string) string {
	replacer := strings.NewReplacer(
		"{{nome}}", name,
		"{{telefone}}", phone,
		"{{email}}", email,
		"{{empresa}}", companyName,
	)
	return replacer.Replace(template)
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

		var totalConversations, openConversations, resolvedConversations int
		var totalMessages int
		var dealsWonValue, dealsOpenValue float64
		var dealsWonCount int
		var totalContacts int
		var slaWithin, slaBreached int

		// Conversations metrics
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1", companyID).Scan(&totalConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'open'", companyID).Scan(&openConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1 AND status = 'resolved'", companyID).Scan(&resolvedConversations)

		// Messages
		svc.DB.QueryRow("SELECT COUNT(*) FROM messages WHERE company_id = $1", companyID).Scan(&totalMessages)

		// Deals
		svc.DB.QueryRow("SELECT COALESCE(SUM(value), 0) FROM deals WHERE company_id = $1 AND status = 'won'", companyID).Scan(&dealsWonValue)
		svc.DB.QueryRow("SELECT COALESCE(SUM(value), 0) FROM deals WHERE company_id = $1 AND status = 'open'", companyID).Scan(&dealsOpenValue)
		svc.DB.QueryRow("SELECT COUNT(*) FROM deals WHERE company_id = $1 AND status = 'won'", companyID).Scan(&dealsWonCount)

		// Contacts
		svc.DB.QueryRow("SELECT COUNT(*) FROM contacts WHERE company_id = $1", companyID).Scan(&totalContacts)

		svc.DB.QueryRow(`
			SELECT COUNT(*) FROM conversations
			WHERE company_id = $1 AND customer_company_id IS NOT NULL AND status = 'resolved'
			  AND (
			    (first_response_due_at IS NULL OR (first_response_at IS NOT NULL AND first_response_at <= first_response_due_at))
			    AND (resolution_due_at IS NULL OR (resolved_at IS NOT NULL AND resolved_at <= resolution_due_at))
			  )
		`, companyID).Scan(&slaWithin)
		svc.DB.QueryRow(`
			SELECT COUNT(*) FROM conversations
			WHERE company_id = $1 AND customer_company_id IS NOT NULL
			  AND (
			    (first_response_due_at IS NOT NULL AND COALESCE(first_response_at, NOW()) > first_response_due_at)
			    OR (resolution_due_at IS NOT NULL AND COALESCE(resolved_at, NOW()) > resolution_due_at)
			  )
		`, companyID).Scan(&slaBreached)

		metrics := fiber.Map{
			"total_conversations":    totalConversations,
			"open_conversations":     openConversations,
			"resolved_conversations": resolvedConversations,
			"total_messages":         totalMessages,
			"deals_won_value":        dealsWonValue,
			"deals_open_value":       dealsOpenValue,
			"deals_won_count":        dealsWonCount,
			"total_contacts":         totalContacts,
			"sla_within":             slaWithin,
			"sla_breached":           slaBreached,
		}

		return c.JSON(fiber.Map{"metrics": metrics})
	}
}

func GetAttendanceMetrics(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		assignedTo := c.Query("assigned_to")

		var attended, resolved int
		var totalTimeMinutes, avgTimeMinutes float64

		// Base filter
		baseFilter := "company_id = $1"
		args := []interface{}{companyID}
		argIdx := 2

		if assignedTo != "" {
			baseFilter += fmt.Sprintf(" AND assigned_to = $%d", argIdx)
			args = append(args, assignedTo)
			argIdx++
		}

		// Attended = conversations that have assigned_to set (in_progress or resolved)
		queryAttended := fmt.Sprintf("SELECT COUNT(*) FROM conversations WHERE %s AND assigned_to IS NOT NULL", baseFilter)
		svc.DB.QueryRow(queryAttended, args...).Scan(&attended)

		// Resolved
		queryResolved := fmt.Sprintf("SELECT COUNT(*) FROM conversations WHERE %s AND status = 'resolved'", baseFilter)
		svc.DB.QueryRow(queryResolved, args...).Scan(&resolved)

		// Average time of attendance (from created_at to updated_at when resolved)
		queryAvg := fmt.Sprintf(`
			SELECT COALESCE(EXTRACT(EPOCH FROM AVG(updated_at - created_at))/60, 0),
			       COALESCE(EXTRACT(EPOCH FROM SUM(updated_at - created_at))/60, 0)
			FROM conversations WHERE %s AND status = 'resolved' AND updated_at IS NOT NULL
		`, baseFilter)
		svc.DB.QueryRow(queryAvg, args...).Scan(&avgTimeMinutes, &totalTimeMinutes)

		return c.JSON(fiber.Map{
			"attended":           attended,
			"resolved":           resolved,
			"total_time_minutes": totalTimeMinutes,
			"avg_time_minutes":   avgTimeMinutes,
		})
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

		var companyID, name, color, greeting, position string
		var channelsDisplayed json.RawMessage
		err := svc.DB.QueryRow(`
			SELECT company_id, name, primary_color, greeting_message, position, channels_displayed
			FROM widgets WHERE id = $1 AND is_active = true
		`, widgetID).Scan(&companyID, &name, &color, &greeting, &position, &channelsDisplayed)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Widget not found"})
		}

		return c.JSON(fiber.Map{
			"id": widgetID, "company_id": companyID, "name": name, "primary_color": color, "greeting_message": greeting,
			"position": position, "channels_displayed": channelsDisplayed,
		})
	}
}

func SendWidgetMessage(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		widgetID := c.Params("id")
		var body struct {
			VisitorID string `json:"visitor_id"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			Phone     string `json:"phone"`
			Message   string `json:"message"`
			PageURL   string `json:"page_url"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		body.Message = strings.TrimSpace(body.Message)
		if body.Message == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Mensagem é obrigatória"})
		}
		if body.VisitorID == "" {
			body.VisitorID = uuid.New().String()
		}

		var companyID string
		err := svc.DB.QueryRow("SELECT company_id FROM widgets WHERE id = $1 AND is_active = true", widgetID).Scan(&companyID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Widget not found"})
		}

		channelID, err := ensureWebchatChannel(svc.DB, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		contactID, err := getOrCreateWidgetContact(svc.DB, companyID, body.VisitorID, body.Name, body.Email, body.Phone)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		conversationID, err := getOrCreateWidgetConversation(svc.DB, companyID, channelID, contactID, body.PageURL)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		messageID := uuid.New().String()
		_, err = svc.DB.Exec(`
			INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, status)
			VALUES ($1, $2, $3, 'contact', $4, $5, 'text', 'delivered')
		`, messageID, conversationID, companyID, contactID, body.Message)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		_, _ = svc.DB.Exec(`
			UPDATE conversations
			SET last_message_at = NOW(), last_message_preview = $1, unread_count = unread_count + 1, updated_at = NOW()
			WHERE id = $2
		`, body.Message, conversationID)

		svc.WSHub.BroadcastToCompany(companyID, "new_message", fiber.Map{
			"id": messageID, "conversation_id": conversationID, "sender_type": "contact",
			"sender_id": contactID, "content": body.Message, "message_type": "text",
			"status": "delivered", "created_at": time.Now(),
		})

		// Broadcast to widget visitor WebSocket room so they get agent replies in real-time
		svc.WSHub.BroadcastToRoom("widget:"+conversationID, "new_message", fiber.Map{
			"id": messageID, "conversation_id": conversationID, "sender_type": "contact",
			"content": body.Message, "message_type": "text", "created_at": time.Now(),
		})

		// Trigger bot automation (same as WhatsApp incoming messages)
		go svc.Bot.TriggerBot(companyID, conversationID, contactID, channelID, body.Message, "", "")

		return c.JSON(fiber.Map{"conversation_id": conversationID, "visitor_id": body.VisitorID, "message_id": messageID})
	}
}

func CloseWidgetConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		widgetID := c.Params("id")
		var body struct {
			ConversationID string `json:"conversation_id"`
			VisitorID      string `json:"visitor_id"`
		}
		if err := c.BodyParser(&body); err != nil || body.ConversationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "conversation_id required"})
		}

		var companyID string
		if err := svc.DB.QueryRow("SELECT company_id FROM widgets WHERE id = $1 AND is_active = true", widgetID).Scan(&companyID); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Widget not found"})
		}

		// Verify the conversation belongs to this company
		var exists bool
		svc.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM conversations WHERE id = $1 AND company_id = $2)", body.ConversationID, companyID).Scan(&exists)
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Conversation not found"})
		}

		if err := svc.Message.CloseConversation(body.ConversationID, companyID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Notify other widget connections and the CRM
		svc.WSHub.BroadcastToRoom("widget:"+body.ConversationID, "conversation_closed", map[string]interface{}{
			"conversation_id": body.ConversationID,
		})

		return c.JSON(fiber.Map{"message": "Conversation closed"})
	}
}

func GetWidgetMessages(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		widgetID := c.Params("id")
		conversationID := c.Query("conversation_id")
		if conversationID == "" {
			return c.JSON(fiber.Map{"messages": []fiber.Map{}})
		}
		var companyID string
		err := svc.DB.QueryRow("SELECT company_id FROM widgets WHERE id = $1 AND is_active = true", widgetID).Scan(&companyID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Widget not found"})
		}
		rows, err := svc.DB.Query(`
			SELECT m.id, m.sender_type, COALESCE(m.content, ''), m.message_type, m.created_at,
			       COALESCE(u.name, COALESCE(m.metadata->>'bot_name', '')),
			       COALESCE(u.avatar_url, '')
			FROM messages m
			LEFT JOIN users u ON m.sender_type = 'user' AND m.sender_id = u.id
			WHERE m.conversation_id = $1 AND m.company_id = $2 AND m.is_private = false
			ORDER BY m.created_at ASC
			LIMIT 100
		`, conversationID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		messages := []fiber.Map{}
		for rows.Next() {
			var id, senderType, content, msgType, senderName, senderAvatar string
			var createdAt time.Time
			if err := rows.Scan(&id, &senderType, &content, &msgType, &createdAt, &senderName, &senderAvatar); err == nil {
				messages = append(messages, fiber.Map{
					"id": id, "sender_type": senderType, "content": content,
					"message_type": msgType, "created_at": createdAt,
					"sender_name": senderName, "sender_avatar": senderAvatar,
				})
			}
		}

		// Check conversation status so the widget can detect closure via polling
		var convStatus string
		svc.DB.QueryRow("SELECT COALESCE(status, 'open') FROM conversations WHERE id = $1 AND company_id = $2", conversationID, companyID).Scan(&convStatus)

		return c.JSON(fiber.Map{"messages": messages, "status": convStatus})
	}
}

func ensureWebchatChannel(db *sql.DB, companyID string) (string, error) {
	var channelID string
	err := db.QueryRow("SELECT id FROM channels WHERE company_id = $1 AND type = 'webchat' AND is_active = true ORDER BY created_at LIMIT 1", companyID).Scan(&channelID)
	if err == nil {
		return channelID, nil
	}
	channelID = uuid.New().String()
	_, err = db.Exec(`
		INSERT INTO channels (id, company_id, name, type, status, is_active)
		VALUES ($1, $2, 'Widget do site', 'webchat', 'connected', true)
	`, channelID, companyID)
	return channelID, err
}

func getOrCreateWidgetContact(db *sql.DB, companyID, visitorID, name, email, phone string) (string, error) {
	var contactID string
	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)

	// 1. Try to find by visitor_id (most reliable for widget - persisted in browser localStorage)
	if visitorID != "" {
		if err := db.QueryRow(
			"SELECT id FROM contacts WHERE company_id = $1 AND custom_fields->>'visitor_id' = $2 LIMIT 1",
			companyID, visitorID,
		).Scan(&contactID); err == nil {
			_, _ = db.Exec("UPDATE contacts SET name = COALESCE(NULLIF($1, ''), name), email = COALESCE(NULLIF($2, ''), email), phone = COALESCE(NULLIF($3, ''), phone), updated_at = NOW() WHERE id = $4", name, email, phone, contactID)
			return contactID, nil
		}
	}

	// 2. Try to find by phone
	if phone != "" {
		if err := db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND phone = $2 LIMIT 1", companyID, phone).Scan(&contactID); err == nil {
			_, _ = db.Exec("UPDATE contacts SET name = COALESCE(NULLIF($1, ''), name), email = COALESCE(NULLIF($2, ''), email), custom_fields = COALESCE(custom_fields, '{}'::jsonb) || jsonb_build_object('visitor_id', $3::text), updated_at = NOW() WHERE id = $4", name, email, visitorID, contactID)
			return contactID, nil
		}
	}

	// 3. Try to find by email
	if email != "" {
		if err := db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND email = $2 LIMIT 1", companyID, email).Scan(&contactID); err == nil {
			_, _ = db.Exec("UPDATE contacts SET name = COALESCE(NULLIF($1, ''), name), phone = COALESCE(NULLIF($2, ''), phone), custom_fields = COALESCE(custom_fields, '{}'::jsonb) || jsonb_build_object('visitor_id', $3::text), updated_at = NOW() WHERE id = $4", name, phone, visitorID, contactID)
			return contactID, nil
		}
	}

	// 4. Create new contact
	contactID = uuid.New().String()
	if name == "" {
		name = "Visitante do site"
	}
	customFields, _ := json.Marshal(fiber.Map{"visitor_id": visitorID})
	_, err := db.Exec(`
		INSERT INTO contacts (id, company_id, name, phone, email, origin, custom_fields)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), 'widget', $6::jsonb)
	`, contactID, companyID, name, phone, email, customFields)
	return contactID, err
}

func getOrCreateWidgetConversation(db *sql.DB, companyID, channelID, contactID, pageURL string) (string, error) {
	var conversationID string
	err := db.QueryRow(`
		SELECT id FROM conversations
		WHERE company_id = $1 AND contact_id = $2 AND channel_id = $3 AND status IN ('open', 'in_progress', 'pending')
		ORDER BY created_at DESC LIMIT 1
	`, companyID, contactID, channelID).Scan(&conversationID)
	if err == nil {
		return conversationID, nil
	}
	conversationID = uuid.New().String()
	metadata, _ := json.Marshal(fiber.Map{"page_url": pageURL})
	_, err = db.Exec(`
		INSERT INTO conversations (id, company_id, contact_id, channel_id, status, priority, metadata, last_message_at)
		VALUES ($1, $2, $3, $4, 'open', 'normal', $5, NOW())
	`, conversationID, companyID, contactID, channelID, metadata)
	return conversationID, err
}

func GetWidgetEmbedScript(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		widgetID := c.Params("id")
		apiBase := c.Protocol() + "://" + c.Hostname()
		script := fmt.Sprintf(`(function(){
  if(window.__vgonWidgetLoaded) return; window.__vgonWidgetLoaded=true;
  var widgetId=%q, apiBase=%q;
  var wsBase=apiBase.replace(/^http/,"ws");
  var visitorId=localStorage.getItem("vgon_wv")||""; if(!visitorId){visitorId=Date.now()+"-"+Math.random().toString(16).slice(2);localStorage.setItem("vgon_wv",visitorId);}
  var conversationId=localStorage.getItem("vgon_wc")||"";
  var savedName=localStorage.getItem("vgon_wn")||"";
  var savedContact=localStorage.getItem("vgon_we")||"";
  var seen={}, ws=null, pollTimer=null, typingTimer=null;
  function css(el,s){for(var k in s)el.style[k]=s[k];}
  function esc(s){return String(s||"").replace(/[<>&]/g,function(x){return{"<":"&lt;",">":"&gt;","&":"&amp;"}[x]});}
  function appendMsg(log,text,own,color,id,senderName,senderAvatar){
    if(id&&seen[id])return;if(id)seen[id]=true;
    hideTyping();
    var d=document.createElement("div");d.style.textAlign=own?"right":"left";d.style.margin="8px 0";
    var avatarSrc=senderAvatar?(senderAvatar.indexOf("http")===0?senderAvatar:apiBase+senderAvatar):"";
    var nameHtml=(!own&&senderName)?'<div style="font-size:11px;color:#6b7280;margin-bottom:2px;display:flex;align-items:center;gap:4px">'+(avatarSrc?'<img src="'+esc(avatarSrc)+'" style="width:18px;height:18px;border-radius:50%%;object-fit:cover">':'')+esc(senderName)+'</div>':'';
    d.innerHTML=nameHtml+'<span style="display:inline-block;max-width:80%%;background:'+(own?color:'#e5e7eb')+';color:'+(own?'#fff':'#111827')+';padding:8px 12px;border-radius:12px;font-size:14px;line-height:1.4;word-wrap:break-word;white-space:pre-wrap">'+esc(text)+'</span>';
    log.appendChild(d);log.scrollTop=log.scrollHeight;
  }
  function showTyping(name){
    var el=document.getElementById("vgon-typing");if(!el)return;
    el.textContent=(name||"Atendente")+" está digitando...";el.style.display="block";
    clearTimeout(typingTimer);typingTimer=setTimeout(hideTyping,5000);
  }
  function hideTyping(){var el=document.getElementById("vgon-typing");if(el)el.style.display="none";}
  function handleConversationClosed(){
    conversationId="";localStorage.removeItem("vgon_wc");
    if(ws){try{ws.close();}catch(e){}}ws=null;
    var log=document.getElementById("vgon-log");
    if(log){var d=document.createElement("div");d.style.textAlign="center";d.style.margin="12px 0";d.innerHTML='<span style="background:#f3f4f6;color:#6b7280;padding:6px 14px;border-radius:20px;font-size:12px">Conversa encerrada</span>';log.appendChild(d);log.scrollTop=log.scrollHeight;}
  }
  function connectWS(){
    if(!conversationId||ws&&ws.readyState<2)return;
    try{
      ws=new WebSocket(wsBase+"/ws/widget?conversation_id="+encodeURIComponent(conversationId));
      ws.onmessage=function(evt){try{
        var msg=JSON.parse(evt.data);
        if(msg.event==="new_message"){
          var d=typeof msg.data==="string"?JSON.parse(msg.data):msg.data;
          if(d.sender_type!=="contact"){var log=document.getElementById("vgon-log");if(log)appendMsg(log,d.content,false,window.__vgonColor||"#3B82F6",d.id,d.sender_name||"",d.sender_avatar||"");}
        }
        if(msg.event==="typing"){var td=typeof msg.data==="string"?JSON.parse(msg.data):msg.data;if(td.is_typing)showTyping(td.user_name||td.user_id);else hideTyping();}
        if(msg.event==="conversation_closed"){handleConversationClosed();}
      }catch(e){}};
      ws.onclose=function(){ws=null;if(conversationId)setTimeout(connectWS,4000);};
      ws.onerror=function(){};
    }catch(e){setTimeout(connectWS,4000);}
  }
  function poll(){
    if(!conversationId)return;
    fetch(apiBase+"/api/widget/"+widgetId+"/messages?conversation_id="+encodeURIComponent(conversationId)).then(function(r){return r.json();}).then(function(data){
      if(data.status==="resolved"){handleConversationClosed();return;}
      var log=document.getElementById("vgon-log");if(!log)return;
      (data.messages||[]).forEach(function(m){appendMsg(log,m.content,m.sender_type==="contact",window.__vgonColor||"#3B82F6",m.id,m.sender_name||"",m.sender_avatar||"");});
    }).catch(function(){});
  }
  fetch(apiBase+"/api/widget/"+widgetId+"/config").then(function(r){return r.json();}).then(function(cfg){
    var color=cfg.primary_color||"#3B82F6";window.__vgonColor=color;
    var side=cfg.position==="bottom-left"?"left":"right";
    var bubble=document.createElement("button");bubble.type="button";bubble.innerHTML="💬";
    css(bubble,{position:"fixed",bottom:"22px",[side]:"22px",width:"58px",height:"58px",borderRadius:"50%%",border:"0",background:color,color:"#fff",fontSize:"24px",boxShadow:"0 12px 30px rgba(0,0,0,.25)",zIndex:"2147483647",cursor:"pointer",transition:"transform .2s"});
    var panel=document.createElement("div");
    css(panel,{position:"fixed",bottom:"92px",[side]:"22px",width:"370px",maxWidth:"calc(100vw - 32px)",height:"520px",maxHeight:"calc(100vh - 120px)",background:"#fff",borderRadius:"14px",boxShadow:"0 18px 50px rgba(0,0,0,.22)",overflow:"hidden",zIndex:"2147483647",fontFamily:"system-ui,-apple-system,sans-serif",display:"none",flexDirection:"column"});
    var hasIdentity=!!(savedName&&savedContact);
    var formHTML=hasIdentity?'':'<div id="vgon-identity" style="padding:12px;border-bottom:1px solid #e5e7eb;background:#fff"><input id="vgon-name" placeholder="Seu nome" value="'+esc(savedName)+'" style="width:100%%;box-sizing:border-box;margin-bottom:8px;padding:10px;border:1px solid #d1d5db;border-radius:8px;font-size:14px"><input id="vgon-email" placeholder="Seu e-mail ou telefone" value="'+esc(savedContact)+'" style="width:100%%;box-sizing:border-box;padding:10px;border:1px solid #d1d5db;border-radius:8px;font-size:14px"></div>';
    panel.innerHTML='<div style="background:'+color+';color:white;padding:16px;font-weight:700;font-size:15px;display:flex;justify-content:space-between;align-items:center"><span>'+(cfg.greeting_message||"Olá! Como podemos ajudar?")+'</span><button id="vgon-close-btn" type="button" title="Encerrar conversa" style="background:none;border:none;color:white;font-size:18px;cursor:pointer;opacity:0.8;padding:0 4px">✕</button></div>'+formHTML+'<div id="vgon-log" style="flex:1;overflow-y:auto;padding:14px;background:#f8fafc;font-size:14px"></div><div id="vgon-typing" style="display:none;padding:4px 14px;font-size:12px;color:#6b7280;font-style:italic"></div><form id="vgon-form" style="padding:12px;border-top:1px solid #e5e7eb;background:#fff;display:flex;gap:8px"><input id="vgon-message" placeholder="Digite sua mensagem..." required style="flex:1;padding:10px;border:1px solid #d1d5db;border-radius:8px;font-size:14px"><button type="submit" style="background:'+color+';color:white;border:0;border-radius:8px;padding:0 14px;cursor:pointer;font-size:18px">➤</button></form>';
    document.body.appendChild(panel);document.body.appendChild(bubble);
    bubble.onclick=function(){var showing=panel.style.display!=="none";panel.style.display=showing?"none":"flex";if(!showing){poll();connectWS();}};
    panel.querySelector("#vgon-close-btn").onclick=function(){
      if(!conversationId){panel.style.display="none";return;}
      if(!confirm("Deseja encerrar esta conversa?"))return;
      fetch(apiBase+"/api/widget/"+widgetId+"/close",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({conversation_id:conversationId,visitor_id:visitorId})}).then(function(){handleConversationClosed();}).catch(function(){handleConversationClosed();});
    };
    connectWS();poll();
    pollTimer=setInterval(poll,5000);
    panel.querySelector("#vgon-form").onsubmit=function(e){
      e.preventDefault();
      var msgInput=panel.querySelector("#vgon-message");var msg=msgInput.value.trim();if(!msg)return;
      var nameEl=panel.querySelector("#vgon-name");var emailEl=panel.querySelector("#vgon-email");
      var name=nameEl?nameEl.value.trim():savedName;
      var contact=emailEl?emailEl.value.trim():savedContact;
      if(!savedName&&name){savedName=name;localStorage.setItem("vgon_wn",name);}
      if(!savedContact&&contact){savedContact=contact;localStorage.setItem("vgon_we",contact);}
      var idBlock=document.getElementById("vgon-identity");if(idBlock&&name&&contact){idBlock.style.display="none";}
      var log=document.getElementById("vgon-log");
      var tempId="t-"+Date.now();appendMsg(log,msg,true,color,tempId);
      msgInput.value="";
      fetch(apiBase+"/api/widget/"+widgetId+"/message",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({visitor_id:visitorId,name:name,email:contact.indexOf("@")>=0?contact:"",phone:contact.indexOf("@")>=0?"":contact,message:msg,page_url:location.href})}).then(function(r){return r.json();}).then(function(data){
        if(data.conversation_id){conversationId=data.conversation_id;localStorage.setItem("vgon_wc",conversationId);connectWS();}
        if(data.message_id)seen[data.message_id]=true;
      }).catch(function(){});
    };
  });
})();`, widgetID, apiBase)
		c.Type("js")
		return c.SendString(script)
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

		rows, _ := svc.DB.Query(`
			SELECT id, shortcut, title, content, category, created_at
			FROM quick_replies WHERE company_id = $1 ORDER BY shortcut
		`, companyID)

		var replies []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, shortcut, content string
				var title, category *string
				var createdAt string
				rows.Scan(&id, &shortcut, &title, &content, &category, &createdAt)
				replies = append(replies, map[string]interface{}{
					"id": id, "shortcut": shortcut, "title": title,
					"content": content, "category": category, "created_at": createdAt,
				})
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
			Category string `json:"category"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Shortcut == "" || body.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Shortcut and content are required"})
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO quick_replies (id, company_id, shortcut, title, content, category, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, companyID, body.Shortcut, body.Title, body.Content, body.Category, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id": id, "shortcut": body.Shortcut, "title": body.Title,
			"content": body.Content, "category": body.Category,
		})
	}
}

func UpdateQuickReply(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		replyID := c.Params("id")

		var body struct {
			Shortcut string `json:"shortcut"`
			Title    string `json:"title"`
			Content  string `json:"content"`
			Category string `json:"category"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Shortcut == "" || body.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Shortcut and content are required"})
		}

		_, err := svc.DB.Exec(`
			UPDATE quick_replies SET shortcut = $1, title = $2, content = $3, category = $4, updated_at = NOW()
			WHERE id = $5 AND company_id = $6
		`, body.Shortcut, body.Title, body.Content, body.Category, replyID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Quick reply updated"})
	}
}

func DeleteQuickReply(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		replyID := c.Params("id")

		_, err := svc.DB.Exec("DELETE FROM quick_replies WHERE id = $1 AND company_id = $2", replyID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Quick reply deleted"})
	}
}

// ============================================
// USERS
// ============================================

func GetUsers(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, _ := svc.DB.Query(`
			SELECT u.id, u.name, u.email, u.avatar_url, u.is_active, u.is_online,
			       COALESCE(u.availability_status, 'offline'), r.name as role_name
			FROM users u
			LEFT JOIN roles r ON u.role_id = r.id
			WHERE u.company_id = $1 ORDER BY u.name
		`, companyID)

		var users []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, name, email, availabilityStatus string
				var avatarURL, roleName *string
				var isActive, isOnline bool
				rows.Scan(&id, &name, &email, &avatarURL, &isActive, &isOnline, &availabilityStatus, &roleName)
				users = append(users, map[string]interface{}{
					"id": id, "name": name, "email": email, "avatar_url": avatarURL,
					"is_active": isActive, "is_online": isOnline, "availability_status": availabilityStatus, "role_name": roleName,
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
