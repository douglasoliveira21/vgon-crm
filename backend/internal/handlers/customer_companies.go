package handlers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type customerCompanyPayload struct {
	Name                      string `json:"name"`
	CNPJ                      string `json:"cnpj"`
	TradeName                 string `json:"trade_name"`
	LegalName                 string `json:"legal_name"`
	Email                     string `json:"email"`
	Phone                     string `json:"phone"`
	City                      string `json:"city"`
	State                     string `json:"state"`
	Address                   string `json:"address"`
	InitialResponseSLAMinutes int    `json:"initial_response_sla_minutes"`
	ResolutionSLAMinutes      int    `json:"resolution_sla_minutes"`
	IsActive                  *bool  `json:"is_active"`
}

var onlyDigits = regexp.MustCompile(`\D`)

func cleanCNPJ(cnpj string) string {
	return onlyDigits.ReplaceAllString(cnpj, "")
}

func formatCNPJ(cnpj string) string {
	d := cleanCNPJ(cnpj)
	if len(d) != 14 {
		return cnpj
	}
	return fmt.Sprintf("%s.%s.%s/%s-%s", d[0:2], d[2:5], d[5:8], d[8:12], d[12:14])
}

func ListCustomerCompanies(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		search := strings.TrimSpace(c.Query("search"))
		args := []interface{}{companyID}
		query := `
			SELECT id, name, cnpj, trade_name, legal_name, email, phone, city, state, address,
			       initial_response_sla_minutes, resolution_sla_minutes, is_active, created_at, updated_at
			FROM customer_companies
			WHERE company_id = $1
		`
		if search != "" {
			query += " AND (name ILIKE $2 OR cnpj ILIKE $2 OR trade_name ILIKE $2 OR legal_name ILIKE $2)"
			args = append(args, "%"+search+"%")
		}
		query += " ORDER BY name"

		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		items := []fiber.Map{}
		for rows.Next() {
			item, err := scanCustomerCompany(rows)
			if err == nil {
				items = append(items, item)
			}
		}
		return c.JSON(fiber.Map{"companies": items})
	}
}

func CreateCustomerCompany(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		var body customerCompanyPayload
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if strings.TrimSpace(body.Name) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
		}
		if body.InitialResponseSLAMinutes <= 0 {
			body.InitialResponseSLAMinutes = 30
		}
		if body.ResolutionSLAMinutes <= 0 {
			body.ResolutionSLAMinutes = 240
		}
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO customer_companies
			(id, company_id, name, cnpj, trade_name, legal_name, email, phone, city, state, address,
			 initial_response_sla_minutes, resolution_sla_minutes, is_active)
			VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		`, id, companyID, strings.TrimSpace(body.Name), formatCNPJ(body.CNPJ), body.TradeName, body.LegalName,
			body.Email, body.Phone, body.City, strings.ToUpper(body.State), body.Address,
			body.InitialResponseSLAMinutes, body.ResolutionSLAMinutes, isActive)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func UpdateCustomerCompany(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		var body customerCompanyPayload
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if strings.TrimSpace(body.Name) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
		}
		if body.InitialResponseSLAMinutes <= 0 {
			body.InitialResponseSLAMinutes = 30
		}
		if body.ResolutionSLAMinutes <= 0 {
			body.ResolutionSLAMinutes = 240
		}
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}

		_, err := svc.DB.Exec(`
			UPDATE customer_companies SET
				name=$1, cnpj=NULLIF($2,''), trade_name=$3, legal_name=$4, email=$5, phone=$6,
				city=$7, state=$8, address=$9, initial_response_sla_minutes=$10,
				resolution_sla_minutes=$11, is_active=$12, updated_at=NOW()
			WHERE id=$13 AND company_id=$14
		`, strings.TrimSpace(body.Name), formatCNPJ(body.CNPJ), body.TradeName, body.LegalName, body.Email, body.Phone,
			body.City, strings.ToUpper(body.State), body.Address, body.InitialResponseSLAMinutes,
			body.ResolutionSLAMinutes, isActive, id, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		refreshOpenConversationSLA(svc.DB, companyID, id)
		return c.JSON(fiber.Map{"message": "Company updated"})
	}
}

func DeleteCustomerCompany(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		_, err := svc.DB.Exec("DELETE FROM customer_companies WHERE id = $1 AND company_id = $2", id, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Company deleted"})
	}
}

func LookupCNPJ(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cnpj := cleanCNPJ(c.Params("cnpj"))
		if len(cnpj) != 14 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "CNPJ inválido"})
		}
		client := http.Client{Timeout: 12 * time.Second}
		data, source, err := lookupCNPJData(client, cnpj)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"cnpj":       formatCNPJ(cnpj),
			"name":       coalesceString(data["nome_fantasia"], data["fantasia"], data["razao_social"], data["nome"]),
			"trade_name": coalesceString(data["nome_fantasia"], data["fantasia"]),
			"legal_name": coalesceString(data["razao_social"], data["nome"]),
			"email":      coalesceString(data["email"]),
			"phone":      coalesceString(data["ddd_telefone_1"], data["telefone"]),
			"city":       coalesceString(data["municipio"]),
			"state":      coalesceString(data["uf"]),
			"address":    strings.TrimSpace(fmt.Sprintf("%s, %s - %s", coalesceString(data["logradouro"]), coalesceString(data["numero"]), coalesceString(data["bairro"]))),
			"source":     source,
		})
	}
}

func lookupCNPJData(client http.Client, cnpj string) (map[string]interface{}, string, error) {
	if data, statusCode, err := fetchCNPJJSON(client, "https://brasilapi.com.br/api/cnpj/v1/"+cnpj); err == nil && statusCode == http.StatusOK {
		return data, "brasilapi", nil
	}

	data, statusCode, err := fetchCNPJJSON(client, "https://www.receitaws.com.br/v1/cnpj/"+cnpj)
	if err != nil {
		return nil, "", fmt.Errorf("Falha ao consultar CNPJ")
	}
	if statusCode == http.StatusOK && strings.ToUpper(coalesceString(data["status"])) != "ERROR" {
		return data, "receitaws", nil
	}
	if statusCode == http.StatusNotFound || strings.ToUpper(coalesceString(data["status"])) == "ERROR" {
		return nil, "", fmt.Errorf("CNPJ não encontrado")
	}
	if statusCode == http.StatusTooManyRequests {
		return nil, "", fmt.Errorf("Limite de consultas de CNPJ atingido. Tente novamente em alguns minutos")
	}
	return nil, "", fmt.Errorf("Serviço de CNPJ indisponível no momento")
}

func fetchCNPJJSON(client http.Client, url string) (map[string]interface{}, int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func ExportCustomerCompanySLAReport(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		customerCompanyID := c.Query("company_id")
		filter := ""
		args := []interface{}{companyID}
		if customerCompanyID != "" {
			filter = " AND cc.id = $2"
			args = append(args, customerCompanyID)
		}

		rows, err := svc.DB.Query(`
			SELECT cc.name, COALESCE(cc.cnpj,''), COALESCE(ct.name,''), COALESCE(ct.phone,''), conv.id,
			       conv.created_at, conv.first_response_at, conv.resolved_at,
			       ROUND(COALESCE(EXTRACT(EPOCH FROM (conv.first_response_at - conv.created_at))/60, 0))::int AS first_minutes,
			       ROUND(COALESCE(EXTRACT(EPOCH FROM (conv.resolved_at - conv.created_at))/60, 0))::int AS resolution_minutes,
			       CASE WHEN conv.first_response_at IS NOT NULL AND conv.first_response_due_at IS NOT NULL AND conv.first_response_at <= conv.first_response_due_at THEN 'sim' ELSE 'nao' END AS first_sla_ok,
			       CASE WHEN conv.resolved_at IS NOT NULL AND conv.resolution_due_at IS NOT NULL AND conv.resolved_at <= conv.resolution_due_at THEN 'sim' ELSE 'nao' END AS resolution_sla_ok
			FROM conversations conv
			JOIN contacts ct ON conv.contact_id = ct.id
			JOIN customer_companies cc ON COALESCE(conv.customer_company_id, ct.customer_company_id) = cc.id
			WHERE conv.company_id = $1`+filter+`
			ORDER BY cc.name, conv.created_at DESC
		`, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		c.Set("Content-Type", "text/csv; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=relatorio-sla-empresas.csv")
		writer := csv.NewWriter(c.Response().BodyWriter())
		_ = writer.Write([]string{"empresa", "cnpj", "contato", "telefone", "conversa", "inicio", "primeira_resposta", "resolucao", "min_primeira_resposta", "min_resolucao", "sla_inicial_ok", "sla_resolucao_ok"})
		for rows.Next() {
			var companyName, cnpj, contactName, phone, convID, firstOK, resolutionOK string
			var createdAt time.Time
			var firstResponseAt, resolvedAt sql.NullTime
			var firstMinutes, resolutionMinutes int
			rows.Scan(&companyName, &cnpj, &contactName, &phone, &convID, &createdAt, &firstResponseAt, &resolvedAt, &firstMinutes, &resolutionMinutes, &firstOK, &resolutionOK)
			_ = writer.Write([]string{
				companyName, cnpj, contactName, phone, convID, createdAt.Format(time.RFC3339),
				formatNullTime(firstResponseAt), formatNullTime(resolvedAt), fmt.Sprint(firstMinutes), fmt.Sprint(resolutionMinutes), firstOK, resolutionOK,
			})
		}
		writer.Flush()
		return nil
	}
}

func scanCustomerCompany(rows *sql.Rows) (fiber.Map, error) {
	var id, name string
	var cnpj, tradeName, legalName, email, phone, city, state, address sql.NullString
	var initial, resolution int
	var active bool
	var createdAt, updatedAt time.Time
	err := rows.Scan(&id, &name, &cnpj, &tradeName, &legalName, &email, &phone, &city, &state, &address, &initial, &resolution, &active, &createdAt, &updatedAt)
	return fiber.Map{
		"id": id, "name": name, "cnpj": nullString(cnpj), "trade_name": nullString(tradeName), "legal_name": nullString(legalName),
		"email": nullString(email), "phone": nullString(phone), "city": nullString(city), "state": nullString(state), "address": nullString(address),
		"initial_response_sla_minutes": initial, "resolution_sla_minutes": resolution, "is_active": active, "created_at": createdAt, "updated_at": updatedAt,
	}, err
}

func refreshOpenConversationSLA(db *sql.DB, tenantID, customerCompanyID string) {
	db.Exec(`
		UPDATE conversations conv
		SET first_response_due_at = conv.created_at + (cc.initial_response_sla_minutes || ' minutes')::interval,
		    resolution_due_at = conv.created_at + (cc.resolution_sla_minutes || ' minutes')::interval
		FROM customer_companies cc
		WHERE conv.company_id = $1 AND cc.id = $2 AND conv.customer_company_id = cc.id AND conv.status != 'resolved'
	`, tenantID, customerCompanyID)
}

func coalesceString(values ...interface{}) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func formatNullTime(value sql.NullTime) string {
	if value.Valid {
		return value.Time.Format(time.RFC3339)
	}
	return ""
}
