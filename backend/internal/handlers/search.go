package handlers

import (
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func GlobalSearch(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		query := strings.TrimSpace(c.Query("q"))
		if len(query) < 2 {
			return c.JSON(fiber.Map{
				"contacts":      []fiber.Map{},
				"conversations": []fiber.Map{},
				"companies":     []fiber.Map{},
			})
		}

		like := "%" + query + "%"
		contacts, err := searchContacts(svc, companyID, like)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		conversations, err := searchConversations(svc, companyID, like)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		companies, err := searchCompanies(svc, companyID, like)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"contacts":      contacts,
			"conversations": conversations,
			"companies":     companies,
		})
	}
}

func searchContacts(svc *services.Container, companyID, like string) ([]fiber.Map, error) {
	rows, err := svc.DB.Query(`
		SELECT c.id, COALESCE(NULLIF(c.name, ''), c.phone, c.email, 'Contato'),
		       COALESCE(c.phone, ''), COALESCE(c.email, ''), COALESCE(c.company_name, cc.name, ''),
		       COALESCE(c.origin, '')
		FROM contacts c
		LEFT JOIN customer_companies cc ON cc.id = c.customer_company_id
		WHERE c.company_id = $1
		  AND (
			c.name ILIKE $2 OR c.phone ILIKE $2 OR c.email ILIKE $2 OR
			c.company_name ILIKE $2 OR cc.name ILIKE $2
		  )
		ORDER BY LOWER(COALESCE(NULLIF(c.name, ''), c.phone, c.email, ''))
		LIMIT 10
	`, companyID, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, name, phone, email, companyName, origin string
		if err := rows.Scan(&id, &name, &phone, &email, &companyName, &origin); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "name": name, "phone": phone, "email": email, "company_name": companyName, "origin": origin})
	}
	return items, rows.Err()
}

func searchConversations(svc *services.Container, companyID, like string) ([]fiber.Map, error) {
	rows, err := svc.DB.Query(`
		SELECT conv.id, COALESCE(ct.name, ct.phone, ct.email, 'Contato'),
		       COALESCE(ct.phone, ''), COALESCE(ct.email, ''),
		       COALESCE(conv.subject, ''), COALESCE(conv.last_message_preview, ''),
		       COALESCE(ch.name, 'Sem canal'), COALESCE(conv.status, 'open'),
		       COALESCE(conv.last_message_at, conv.created_at)
		FROM conversations conv
		JOIN contacts ct ON ct.id = conv.contact_id
		LEFT JOIN channels ch ON ch.id = conv.channel_id
		LEFT JOIN customer_companies cc ON cc.id = COALESCE(conv.customer_company_id, ct.customer_company_id)
		WHERE conv.company_id = $1
		  AND (
			ct.name ILIKE $2 OR ct.phone ILIKE $2 OR ct.email ILIKE $2 OR
			conv.subject ILIKE $2 OR conv.last_message_preview ILIKE $2 OR
			cc.name ILIKE $2 OR ch.name ILIKE $2
		  )
		ORDER BY COALESCE(conv.last_message_at, conv.created_at) DESC
		LIMIT 10
	`, companyID, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, contactName, phone, email, subject, preview, channelName, status string
		var lastMessageAt time.Time
		if err := rows.Scan(&id, &contactName, &phone, &email, &subject, &preview, &channelName, &status, &lastMessageAt); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "contact_name": contactName, "phone": phone, "email": email, "subject": subject, "preview": preview, "channel_name": channelName, "status": status, "last_message_at": lastMessageAt})
	}
	return items, rows.Err()
}

func searchCompanies(svc *services.Container, companyID, like string) ([]fiber.Map, error) {
	rows, err := svc.DB.Query(`
		SELECT id, COALESCE(name, trade_name, legal_name, 'Empresa'), COALESCE(cnpj, ''),
		       COALESCE(trade_name, ''), COALESCE(legal_name, ''), COALESCE(email, ''), COALESCE(phone, ''),
		       COALESCE(city, ''), COALESCE(state, '')
		FROM customer_companies
		WHERE company_id = $1
		  AND (
			name ILIKE $2 OR cnpj ILIKE $2 OR trade_name ILIKE $2 OR
			legal_name ILIKE $2 OR email ILIKE $2 OR phone ILIKE $2
		  )
		ORDER BY name
		LIMIT 10
	`, companyID, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, name, cnpj, tradeName, legalName, email, phone, city, state string
		if err := rows.Scan(&id, &name, &cnpj, &tradeName, &legalName, &email, &phone, &city, &state); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "name": name, "cnpj": cnpj, "trade_name": tradeName, "legal_name": legalName, "email": email, "phone": phone, "city": city, "state": state})
	}
	return items, rows.Err()
}
