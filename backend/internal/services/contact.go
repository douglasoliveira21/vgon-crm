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
	Name        *string `json:"name"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	CompanyName *string `json:"company_name"`
	Position    *string `json:"position"`
	City        *string `json:"city"`
	State       *string `json:"state"`
	Origin      *string `json:"origin"`
	Notes       *string `json:"notes"`
}

type UpdateContactRequest struct {
	Name        *string `json:"name"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	CompanyName *string `json:"company_name"`
	Position    *string `json:"position"`
	City        *string `json:"city"`
	State       *string `json:"state"`
	Origin      *string `json:"origin"`
	Notes       *string `json:"notes"`
	AssignedTo  *string `json:"assigned_to"`
}

// GetContacts returns contacts for a company
func (s *ContactService) GetContacts(companyID string, search string, limit, offset int) ([]models.Contact, int, error) {
	if limit == 0 {
		limit = 50
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM contacts WHERE company_id = $1"
	args := []interface{}{companyID}
	argIdx := 2

	if search != "" {
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	var total int
	s.db.QueryRow(countQuery, args...).Scan(&total)

	// Data query
	query := `
		SELECT id, company_id, name, phone, email, company_name, position, city, state, origin, 
			   avatar_url, notes, assigned_to, is_opted_out, created_at, updated_at
		FROM contacts WHERE company_id = $1
	`
	dataArgs := []interface{}{companyID}
	dataArgIdx := 2

	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", dataArgIdx, dataArgIdx, dataArgIdx)
		dataArgs = append(dataArgs, "%"+search+"%")
		dataArgIdx++
	}

	query += " ORDER BY updated_at DESC"
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
			&c.ID, &c.CompanyID, &c.Name, &c.Phone, &c.Email, &c.CompanyName,
			&c.Position, &c.City, &c.State, &c.Origin, &c.AvatarURL, &c.Notes,
			&c.AssignedTo, &c.IsOptedOut, &c.CreatedAt, &c.UpdatedAt,
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
		SELECT id, company_id, name, phone, email, company_name, position, city, state, origin,
			   avatar_url, notes, assigned_to, is_opted_out, created_at, updated_at
		FROM contacts WHERE id = $1 AND company_id = $2
	`, contactID, companyID).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.Phone, &c.Email, &c.CompanyName,
		&c.Position, &c.City, &c.State, &c.Origin, &c.AvatarURL, &c.Notes,
		&c.AssignedTo, &c.IsOptedOut, &c.CreatedAt, &c.UpdatedAt,
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
		INSERT INTO contacts (id, company_id, name, phone, email, company_name, position, city, state, origin, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, id, companyID, req.Name, req.Phone, req.Email, req.CompanyName, req.Position, req.City, req.State, req.Origin, req.Notes)
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
			company_name = COALESCE($6, company_name),
			position = COALESCE($7, position),
			city = COALESCE($8, city),
			state = COALESCE($9, state),
			origin = COALESCE($10, origin),
			notes = COALESCE($11, notes),
			assigned_to = COALESCE($12, assigned_to),
			updated_at = NOW()
		WHERE id = $1 AND company_id = $2
	`, contactID, companyID, req.Name, req.Phone, req.Email, req.CompanyName,
		req.Position, req.City, req.State, req.Origin, req.Notes, req.AssignedTo)
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
