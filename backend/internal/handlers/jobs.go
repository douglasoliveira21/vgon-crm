package handlers

import (
	"database/sql"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func ListDeadLetterJobs(svc *services.Container, global bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID, _ := c.Locals("company_id").(string)
		query := `
			SELECT dead.id, dead.company_id, COALESCE(company.name, 'Tenant removido'),
			       dead.job_type, dead.attempts, COALESCE(dead.last_error, ''), dead.failed_at
			FROM dead_letter_jobs dead
			LEFT JOIN companies company ON company.id = dead.company_id`
		args := []interface{}{}
		if !global {
			query += ` WHERE dead.company_id = $1`
			args = append(args, companyID)
		}
		query += ` ORDER BY dead.failed_at DESC LIMIT 200`
		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao carregar fila de falhas"})
		}
		defer rows.Close()
		items := []fiber.Map{}
		for rows.Next() {
			var id, jobType, tenantName, lastError string
			var tenantID sql.NullString
			var attempts int
			var failedAt time.Time
			if rows.Scan(&id, &tenantID, &tenantName, &jobType, &attempts, &lastError, &failedAt) == nil {
				items = append(items, fiber.Map{
					"id": id, "company_id": nullableStringMapValue(tenantID), "tenant_name": tenantName,
					"job_type": jobType, "attempts": attempts, "last_error": lastError, "failed_at": failedAt,
				})
			}
		}
		return c.JSON(fiber.Map{"jobs": items})
	}
}

func RetryDeadLetterJob(svc *services.Container, global bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID, _ := c.Locals("company_id").(string)
		err := svc.Jobs.RetryDeadLetter(c.Params("id"), companyID, global)
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Job não encontrado"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao reenfileirar job"})
		}
		return c.JSON(fiber.Map{"message": "Job reenfileirado"})
	}
}
