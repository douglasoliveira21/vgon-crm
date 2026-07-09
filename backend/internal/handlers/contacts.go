package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetContacts(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		search := c.Query("search")
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		contacts, total, err := svc.Contact.GetContacts(companyID, search, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"contacts": contacts,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

func GetContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		contact, err := svc.Contact.GetContactByID(contactID, companyID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
		}

		return c.JSON(contact)
	}
}

func CreateContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var req services.CreateContactRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		contact, err := svc.Contact.CreateContact(companyID, &req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		logAuditEvent(svc.DB, c, "contact.create", "contact", contact.ID, fiber.Map{"name": contact.Name, "email": contact.Email, "phone": contact.Phone})
		if req.ConsentStatus != nil && *req.ConsentStatus != "" && *req.ConsentStatus != "unknown" {
			recordContactConsent(svc.DB, companyID, contact.ID, c.Locals("user_id").(string), *req.ConsentStatus, stringPtrValue(req.ConsentSource), "", stringPtrValue(req.ConsentText), c.IP(), c.Get("User-Agent"))
		}

		return c.Status(fiber.StatusCreated).JSON(contact)
	}
}

func UpdateContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		var req services.UpdateContactRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		contact, err := svc.Contact.UpdateContact(contactID, companyID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		logAuditEvent(svc.DB, c, "contact.update", "contact", contactID, fiber.Map{"name": contact.Name, "email": contact.Email, "phone": contact.Phone})
		if req.ConsentStatus != nil && *req.ConsentStatus != "" {
			recordContactConsent(svc.DB, companyID, contactID, c.Locals("user_id").(string), *req.ConsentStatus, stringPtrValue(req.ConsentSource), stringPtrValue(req.OptOutReason), stringPtrValue(req.ConsentText), c.IP(), c.Get("User-Agent"))
		}

		return c.JSON(contact)
	}
}

func DeleteContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		if err := svc.Contact.DeleteContact(contactID, companyID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		logAuditEvent(svc.DB, c, "contact.delete", "contact", contactID, nil)

		return c.JSON(fiber.Map{"message": "Contact deleted"})
	}
}

func UpdateContactConsent(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		contactID := c.Params("id")

		var body struct {
			Status      string `json:"status"`
			Source      string `json:"source"`
			Reason      string `json:"reason"`
			ConsentText string `json:"consent_text"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.Status != "granted" && body.Status != "revoked" && body.Status != "opted_out" && body.Status != "unknown" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Status de consentimento inválido"})
		}

		contact, err := updateContactConsentState(svc.DB, companyID, contactID, userID, body.Status, body.Source, body.Reason, body.ConsentText)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		recordContactConsent(svc.DB, companyID, contactID, userID, body.Status, body.Source, body.Reason, body.ConsentText, c.IP(), c.Get("User-Agent"))
		logAuditEvent(svc.DB, c, "contact.consent.update", "contact", contactID, fiber.Map{"status": body.Status, "source": body.Source, "reason": body.Reason})

		return c.JSON(contact)
	}
}

func GetContactAudit(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		consents, err := listContactConsents(svc.DB, companyID, contactID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		audit, err := listContactAuditLogs(svc.DB, companyID, contactID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"consents": consents, "audit": audit})
	}
}

func ExportContactData(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		contact, err := svc.Contact.GetContactByID(contactID, companyID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
		}
		consents, err := listContactConsents(svc.DB, companyID, contactID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		audit, err := listContactAuditLogs(svc.DB, companyID, contactID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		conversations, err := listContactConversationExport(svc.DB, companyID, contactID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		messages, err := listContactMessagesExport(svc.DB, companyID, contactID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		logAuditEvent(svc.DB, c, "contact.export", "contact", contactID, nil)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=contato-%s-lgpd.json", contactID))
		return c.JSON(fiber.Map{
			"exported_at":    time.Now(),
			"contact":        contact,
			"consents":       consents,
			"audit":          audit,
			"conversations":  conversations,
			"messages":       messages,
			"legal_notice":   "Exportação de dados pessoais solicitada no CRM.",
			"data_subject_id": contactID,
		})
	}
}

func AddContactTag(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		contactID := c.Params("id")

		var body struct {
			TagID string `json:"tag_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := svc.Contact.AddTagToContact(contactID, body.TagID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Tag added"})
	}
}

func RemoveContactTag(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		contactID := c.Params("id")
		tagID := c.Params("tagId")

		if err := svc.Contact.RemoveTagFromContact(contactID, tagID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Tag removed"})
	}
}

func updateContactConsentState(db *sql.DB, companyID, contactID, userID, status, source, reason, consentText string) (fiber.Map, error) {
	isOptedOut := status == "opted_out" || status == "revoked"
	_, err := db.Exec(`
		UPDATE contacts
		SET consent_status = $1,
			consent_source = NULLIF($2, ''),
			consent_text = NULLIF($3, ''),
			consent_given_at = CASE WHEN $1 = 'granted' THEN COALESCE(consent_given_at, NOW()) ELSE consent_given_at END,
			consent_revoked_at = CASE WHEN $1 IN ('revoked', 'opted_out') THEN NOW() WHEN $1 = 'granted' THEN NULL ELSE consent_revoked_at END,
			consent_updated_by = NULLIF($4, '')::uuid,
			is_opted_out = $5,
			opted_out_at = CASE WHEN $5 THEN NOW() ELSE NULL END,
			opt_out_reason = CASE WHEN $5 THEN NULLIF($6, '') ELSE NULL END,
			opt_out_source = CASE WHEN $5 THEN NULLIF($2, '') ELSE NULL END,
			updated_at = NOW()
		WHERE id = $7 AND company_id = $8
	`, status, source, consentText, userID, isOptedOut, reason, contactID, companyID)
	if err != nil {
		return nil, err
	}

	var id, consentStatus string
	var isOptedOut bool
	var consentGivenAt, consentRevokedAt, optedOutAt sql.NullTime
	var optOutReason, optOutSource sql.NullString
	err = db.QueryRow(`
		SELECT id, COALESCE(consent_status, 'unknown'), COALESCE(is_opted_out, false),
		       consent_given_at, consent_revoked_at, opted_out_at, opt_out_reason, opt_out_source
		FROM contacts WHERE id = $1 AND company_id = $2
	`, contactID, companyID).Scan(&id, &consentStatus, &isOptedOut, &consentGivenAt, &consentRevokedAt, &optedOutAt, &optOutReason, &optOutSource)
	if err != nil {
		return nil, err
	}
	return fiber.Map{
		"id":                 id,
		"consent_status":     consentStatus,
		"is_opted_out":       isOptedOut,
		"consent_given_at":   nullableTimeMapValue(consentGivenAt),
		"consent_revoked_at": nullableTimeMapValue(consentRevokedAt),
		"opted_out_at":       nullableTimeMapValue(optedOutAt),
		"opt_out_reason":     nullableStringMapValue(optOutReason),
		"opt_out_source":     nullableStringMapValue(optOutSource),
	}, nil
}

func recordContactConsent(db *sql.DB, companyID, contactID, userID, status, source, reason, consentText, ipAddress, userAgent string) {
	_, _ = db.Exec(`
		INSERT INTO contact_consents (id, company_id, contact_id, user_id, status, source, reason, consent_text, ip_address, user_agent)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), $9, $10)
	`, uuid.New().String(), companyID, contactID, userID, status, source, reason, consentText, ipAddress, userAgent)
}

func listContactConsents(db *sql.DB, companyID, contactID string) ([]fiber.Map, error) {
	rows, err := db.Query(`
		SELECT cc.id, cc.status, COALESCE(cc.source, ''), COALESCE(cc.reason, ''), COALESCE(cc.consent_text, ''),
		       COALESCE(u.name, 'Sistema'), cc.created_at
		FROM contact_consents cc
		LEFT JOIN users u ON u.id = cc.user_id
		WHERE cc.company_id = $1 AND cc.contact_id = $2
		ORDER BY cc.created_at DESC
	`, companyID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, status, source, reason, consentText, userName string
		var createdAt time.Time
		if err := rows.Scan(&id, &status, &source, &reason, &consentText, &userName, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "status": status, "source": source, "reason": reason, "consent_text": consentText, "user_name": userName, "created_at": createdAt})
	}
	return items, rows.Err()
}

func listContactAuditLogs(db *sql.DB, companyID, contactID string) ([]fiber.Map, error) {
	rows, err := db.Query(`
		SELECT al.id, al.action, COALESCE(u.name, 'Sistema'), COALESCE(al.ip_address, ''), COALESCE(al.metadata, '{}'::jsonb), al.created_at
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		WHERE al.company_id = $1 AND al.entity_type = 'contact' AND al.entity_id = $2
		ORDER BY al.created_at DESC
		LIMIT 100
	`, companyID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, action, userName, ipAddress string
		var metadata []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &action, &userName, &ipAddress, &metadata, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "action": action, "user_name": userName, "ip_address": ipAddress, "metadata": string(metadata), "created_at": createdAt})
	}
	return items, rows.Err()
}

func listContactConversationExport(db *sql.DB, companyID, contactID string) ([]fiber.Map, error) {
	rows, err := db.Query(`
		SELECT conv.id, COALESCE(conv.subject, ''), COALESCE(ch.name, ''), COALESCE(ch.type, ''),
		       COALESCE(conv.status, ''), conv.created_at, COALESCE(conv.last_message_at, conv.created_at)
		FROM conversations conv
		LEFT JOIN channels ch ON ch.id = conv.channel_id
		WHERE conv.company_id = $1 AND conv.contact_id = $2
		ORDER BY COALESCE(conv.last_message_at, conv.created_at) DESC
	`, companyID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, subject, channelName, channelType, status string
		var createdAt, lastMessageAt time.Time
		if err := rows.Scan(&id, &subject, &channelName, &channelType, &status, &createdAt, &lastMessageAt); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "subject": subject, "channel_name": channelName, "channel_type": channelType, "status": status, "created_at": createdAt, "last_message_at": lastMessageAt})
	}
	return items, rows.Err()
}

func listContactMessagesExport(db *sql.DB, companyID, contactID string) ([]fiber.Map, error) {
	rows, err := db.Query(`
		SELECT m.id, m.conversation_id, COALESCE(m.sender_type, ''), COALESCE(m.content, ''),
		       COALESCE(m.message_type, ''), COALESCE(m.media_url, ''), COALESCE(m.status, ''), m.created_at
		FROM messages m
		JOIN conversations conv ON conv.id = m.conversation_id
		WHERE m.company_id = $1 AND conv.company_id = $1 AND conv.contact_id = $2
		ORDER BY m.created_at ASC
	`, companyID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []fiber.Map{}
	for rows.Next() {
		var id, conversationID, senderType, content, messageType, mediaURL, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &conversationID, &senderType, &content, &messageType, &mediaURL, &status, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, fiber.Map{"id": id, "conversation_id": conversationID, "sender_type": senderType, "content": content, "message_type": messageType, "media_url": mediaURL, "status": status, "created_at": createdAt})
	}
	return items, rows.Err()
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableTimeMapValue(value sql.NullTime) interface{} {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func nullableStringMapValue(value sql.NullString) interface{} {
	if !value.Valid {
		return nil
	}
	return value.String
}
