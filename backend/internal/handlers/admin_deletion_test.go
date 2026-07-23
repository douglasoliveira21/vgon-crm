package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func TestDeleteTenantRollsBackWhenTenantDoesNotExist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT name FROM companies").WithArgs("tenant-missing").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectRollback()

	app := fiber.New()
	app.Delete("/admin/tenants/:id", DeleteTenant(&services.Container{DB: db}))
	response, err := app.Test(httptest.NewRequest("DELETE", "/admin/tenants/tenant-missing", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminDeleteUserProtectsSuperAdmin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT company_id, COALESCE").
		WithArgs("user-a").
		WillReturnRows(sqlmock.NewRows([]string{"company_id", "is_super_admin"}).AddRow("tenant-a", true))

	app := fiber.New()
	app.Delete("/admin/users/:userId", AdminDeleteUser(&services.Container{DB: db}))
	response, err := app.Test(httptest.NewRequest("DELETE", "/admin/users/user-a", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
