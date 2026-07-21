package services

import (
	"database/sql"
	"fmt"

	"github.com/evocrm/backend/internal/models"
	"github.com/google/uuid"
)

type ContactService struct {
	db *sql.DB
}

func NewContactService(db *sql.DB) *ContactService {
	return &ContactService{db: db}
}

type CreateContactRequest struct {
	Name              *string `json:"name"`
	Phone             *string `json:"phone"`
	Email             *string `json:"email"`
	CustomerCompanyID *string `json:"customer_company_id"`
	CompanyName       *string `json:"company_name"`
	Position          *string `json:"position"`
	City              *string `json:"city"`
	State             *string `json:"state"`
	Origin            *string `json:"origin"`
	Notes             *string `json:"notes"`
	ConsentStatus     *string `json:"consent_status"`
	ConsentSource     *string `json:"consent_source"`
	ConsentText       *string `json:"consent_text"`
}

type UpdateContactRequest struct {
	Name              *string `json:"name"`
	Phone             *string `json:"phone"`
	Email             *string `json:"email"`
	CustomerCompanyID *string `json:"customer_company_id"`
	CompanyName       *string `json:"company_name"`
	Position          *string `json:"position"`
	City              *string `json:"city"`
	State             *string `json:"state"`
	Origin            *string `json:"origin"`
	Notes             *string `json:"notes"`
	AssignedTo        *string `json:"assigned_to"`
	IsOptedOut        *bool   `json:"is_opted_out"`
	OptOutReason      *string `json:"opt_out_reason"`
	OptOutSource      *string `json:"opt_out_source"`
	ConsentStatus     *string `json:"consent_status"`
	ConsentSource     *string `json:"consent_source"`
	ConsentText       *string `json:"consent_text"`
}

// GetContacts returns contacts for a company
func (s *ContactService) GetContacts(companyID string, search string, limit, offset int, blocked bool) ([]models.Contact, int, error) {
	if limit == 0 {
		limit = 50
	}

	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM contacts c
		LEFT JOIN customer_companies cc ON c.customer_company_id = cc.id
		WHERE c.company_id = $1 AND c.is_blocked = $2
	`
	args := []interface{}{companyID, blocked}
	argIdx := 3

	if search != "" {
		countQuery += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.phone ILIKE $%d OR c.email ILIKE $%d OR c.company_name ILIKE $%d OR cc.name ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	var total int
	s.db.QueryRow(countQuery, args...).Scan(&total)

	// Data query
	query := `
		SELECT c.id, c.company_id, c.name, c.phone, c.email, c.customer_company_id, cc.name,
			   c.company_name, c.position, c.city, c.state, c.origin, c.avatar_url, c.notes,
			   c.assigned_to, c.is_opted_out, c.is_blocked, c.blocked_at, c.blocked_by, c.opted_out_at, c.opt_out_reason, c.opt_out_source,
			   c.consent_status, c.consent_source, c.consent_text, c.consent_given_at, c.consent_revoked_at,
			   c.created_at, c.updated_at
		FROM contacts c
		LEFT JOIN customer_companies cc ON c.customer_company_id = cc.id
		WHERE c.company_id = $1 AND c.is_blocked = $2
	`
	dataArgs := []interface{}{companyID, blocked}
	dataArgIdx := 3

	if search != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.phone ILIKE $%d OR c.email ILIKE $%d OR c.company_name ILIKE $%d OR cc.name ILIKE $%d)", dataArgIdx, dataArgIdx, dataArgIdx, dataArgIdx, dataArgIdx)
		dataArgs = append(dataArgs, "%"+search+"%")
		dataArgIdx++
	}

	query += " ORDER BY LOWER(COALESCE(NULLIF(c.name, ''), NULLIF(c.phone, ''), NULLIF(c.email, ''), '')), c.created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", dataArgIdx, dataArgIdx+1)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := s.db.Query(query, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch contacts: %w", err)
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		err := rows.Scan(
			&c.ID, &c.CompanyID, &c.Name, &c.Phone, &c.Email, &c.CustomerCompanyID, &c.CustomerCompanyName, &c.CompanyName,
			&c.Position, &c.City, &c.State, &c.Origin, &c.AvatarURL, &c.Notes,
			&c.AssignedTo, &c.IsOptedOut, &c.IsBlocked, &c.BlockedAt, &c.BlockedBy, &c.OptedOutAt, &c.OptOutReason, &c.OptOutSource,
			&c.ConsentStatus, &c.ConsentSource, &c.ConsentText, &c.ConsentGivenAt, &c.ConsentRevokedAt,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			continue
		}
		contacts = append(contacts, c)
	}

	return contacts, total, nil
}

// GetContactByID returns a single contact
func (s *ContactService) GetContactByID(contactID, companyID string) (*models.Contact, error) {
	var c models.Contact
	err := s.db.QueryRow(`
		SELECT c.id, c.company_id, c.name, c.phone, c.email, c.customer_company_id, cc.name,
			   c.company_name, c.position, c.city, c.state, c.origin,
			   c.avatar_url, c.notes, c.assigned_to, c.is_opted_out, c.is_blocked, c.blocked_at, c.blocked_by, c.opted_out_at, c.opt_out_reason, c.opt_out_source,
			   c.consent_status, c.consent_source, c.consent_text, c.consent_given_at, c.consent_revoked_at,
			   c.created_at, c.updated_at
		FROM contacts c
		LEFT JOIN customer_companies cc ON c.customer_company_id = cc.id
		WHERE c.id = $1 AND c.company_id = $2
	`, contactID, companyID).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.Phone, &c.Email, &c.CustomerCompanyID, &c.CustomerCompanyName, &c.CompanyName,
		&c.Position, &c.City, &c.State, &c.Origin, &c.AvatarURL, &c.Notes,
		&c.AssignedTo, &c.IsOptedOut, &c.IsBlocked, &c.BlockedAt, &c.BlockedBy, &c.OptedOutAt, &c.OptOutReason, &c.OptOutSource,
		&c.ConsentStatus, &c.ConsentSource, &c.ConsentText, &c.ConsentGivenAt, &c.ConsentRevokedAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	// Get tags
	tagRows, err := s.db.Query(`
		SELECT t.id, t.company_id, t.name, t.color
		FROM tags t
		INNER JOIN contact_tags ct ON t.id = ct.tag_id
		WHERE ct.contact_id = $1
	`, contactID)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var tag models.Tag
			tagRows.Scan(&tag.ID, &tag.CompanyID, &tag.Name, &tag.Color)
			c.Tags = append(c.Tags, tag)
		}
	}

	return &c, nil
}

// CreateContact creates a new contact
func (s *ContactService) CreateContact(companyID string, req *CreateContactRequest) (*models.Contact, error) {
	// Check for duplicate phone
	if req.Phone != nil && *req.Phone != "" {
		var exists bool
		s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE company_id = $1 AND phone = $2)", companyID, *req.Phone).Scan(&exists)
		if exists {
			return nil, fmt.Errorf("contact with this phone already exists")
		}
	}

	id := uuid.New().String()
	_, err := s.db.Exec(`
		INSERT INTO contacts (id, company_id, name, phone, email, customer_company_id, company_name, position, city, state, origin, notes,
			consent_status, consent_source, consent_text, consent_given_at, consent_revoked_at, is_opted_out, opted_out_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, $7, $8, $9, $10, $11, $12,
			COALESCE(NULLIF($13, ''), 'unknown'), NULLIF($14, ''), NULLIF($15, ''),
			CASE WHEN NULLIF($13, '') = 'granted' THEN NOW() ELSE NULL END,
			CASE WHEN NULLIF($13, '') IN ('revoked', 'opted_out') THEN NOW() ELSE NULL END,
			CASE WHEN NULLIF($13, '') IN ('revoked', 'opted_out') THEN true ELSE false END,
			CASE WHEN NULLIF($13, '') IN ('revoked', 'opted_out') THEN NOW() ELSE NULL END)
	`, id, companyID, req.Name, req.Phone, req.Email, stringPtrValue(req.CustomerCompanyID), req.CompanyName, req.Position, req.City, req.State, req.Origin, req.Notes,
		stringPtrValue(req.ConsentStatus), stringPtrValue(req.ConsentSource), stringPtrValue(req.ConsentText))
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	return s.GetContactByID(id, companyID)
}

// UpdateContact updates an existing contact
func (s *ContactService) UpdateContact(contactID, companyID string, req *UpdateContactRequest) (*models.Contact, error) {
	_, err := s.db.Exec(`
		UPDATE contacts SET
			name = COALESCE($3, name),
			phone = COALESCE($4, phone),
			email = COALESCE($5, email),
			customer_company_id = COALESCE(NULLIF($6, '')::uuid, customer_company_id),
			company_name = COALESCE($7, company_name),
			position = COALESCE($8, position),
			city = COALESCE($9, city),
			state = COALESCE($10, state),
			origin = COALESCE($11, origin),
			notes = COALESCE($12, notes),
			assigned_to = COALESCE($13, assigned_to),
			is_opted_out = CASE
				WHEN NULLIF($17, '') IN ('revoked', 'opted_out') THEN true
				WHEN NULLIF($17, '') = 'granted' THEN false
				ELSE COALESCE($14, is_opted_out)
			END,
			opted_out_at = CASE
				WHEN NULLIF($17, '') IN ('revoked', 'opted_out') THEN COALESCE(opted_out_at, NOW())
				WHEN NULLIF($17, '') = 'granted' THEN NULL
				WHEN COALESCE($14, false) = true THEN COALESCE(opted_out_at, NOW())
				WHEN $14 = false THEN NULL
				ELSE opted_out_at
			END,
			opt_out_reason = COALESCE($15, opt_out_reason),
			opt_out_source = COALESCE($16, opt_out_source),
			consent_status = COALESCE(NULLIF($17, ''), consent_status),
			consent_source = COALESCE($18, consent_source),
			consent_text = COALESCE($19, consent_text),
			consent_given_at = CASE WHEN NULLIF($17, '') = 'granted' THEN COALESCE(consent_given_at, NOW()) ELSE consent_given_at END,
			consent_revoked_at = CASE WHEN NULLIF($17, '') IN ('revoked', 'opted_out') THEN COALESCE(consent_revoked_at, NOW()) WHEN NULLIF($17, '') = 'granted' THEN NULL ELSE consent_revoked_at END,
			updated_at = NOW()
		WHERE id = $1 AND company_id = $2
	`, contactID, companyID, req.Name, req.Phone, req.Email, stringPtrValue(req.CustomerCompanyID), req.CompanyName,
		req.Position, req.City, req.State, req.Origin, req.Notes, req.AssignedTo, req.IsOptedOut, req.OptOutReason, req.OptOutSource,
		stringPtrValue(req.ConsentStatus), req.ConsentSource, req.ConsentText)
	if err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	return s.GetContactByID(contactID, companyID)
}

// DeleteContact soft-deletes a contact
func (s *ContactService) DeleteContact(contactID, companyID string) error {
	_, err := s.db.Exec("DELETE FROM contacts WHERE id = $1 AND company_id = $2", contactID, companyID)
	return err
}

// AddTagToContact adds a tag to a contact
func (s *ContactService) AddTagToContact(contactID, tagID string) error {
	_, err := s.db.Exec(`
		INSERT INTO contact_tags (contact_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, contactID, tagID)
	return err
}

// RemoveTagFromContact removes a tag from a contact
func (s *ContactService) RemoveTagFromContact(contactID, tagID string) error {
	_, err := s.db.Exec("DELETE FROM contact_tags WHERE contact_id = $1 AND tag_id = $2", contactID, tagID)
	return err
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
