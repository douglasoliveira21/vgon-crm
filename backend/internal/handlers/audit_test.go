package handlers

import (
	"testing"
	"time"
)

func TestAuditHashChangesWhenEntryIsModified(t *testing.T) {
	at := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	original := hashAuditEntry("", "id-a", "tenant-a", "user-a", "contact.update",
		"contact", "contact-a", "127.0.0.1", at, []byte(`{"field":"name"}`))
	repeated := hashAuditEntry("", "id-a", "tenant-a", "user-a", "contact.update",
		"contact", "contact-a", "127.0.0.1", at, []byte(`{"field":"name"}`))
	altered := hashAuditEntry("", "id-a", "tenant-a", "user-a", "contact.delete",
		"contact", "contact-a", "127.0.0.1", at, []byte(`{"field":"name"}`))
	if original != repeated {
		t.Fatal("audit hash is not deterministic")
	}
	if original == altered {
		t.Fatal("audit hash did not detect an altered action")
	}
}
