package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFindEntitiesByCNPJDiscoversPluginSearchField(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Session-Token") != "session-a" {
			t.Fatalf("Session-Token = %q", request.Header.Get("Session-Token"))
		}
		switch request.URL.Path {
		case "/listSearchOptions/Entity":
			_ = json.NewEncoder(response).Encode(map[string]interface{}{
				"2":   map[string]string{"name": "ID", "uid": "Entity.id"},
				"1":   map[string]string{"name": "Nome", "uid": "Entity.name"},
				"766": map[string]string{"name": "Plugins - Campos adicionais - CNPJ", "uid": "PluginFieldsEntity.cnpj"},
			})
		case "/search/Entity":
			if got := request.URL.Query().Get("criteria[0][field]"); got != "766" {
				t.Fatalf("criteria field = %q, want 766", got)
			}
			if got := request.URL.Query().Get("criteria[0][value]"); got != "^12345678000199$" {
				t.Fatalf("criteria value = %q", got)
			}
			_ = json.NewEncoder(response).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"2": 42, "766": "12345678000199"},
				},
			})
		case "/Entity/42":
			_ = json.NewEncoder(response).Encode(GLPIEntity{ID: 42, Name: "Empresa Teste", CompleteName: "Matriz > Empresa Teste"})
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	service := NewGLPIService(server.URL, "app-token")
	entities, err := service.FindEntitiesByCNPJ("session-a", "12.345.678/0001-99")
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 1 || entities[0].ID != 42 {
		t.Fatalf("entities = %#v", entities)
	}
}

func TestFindEntitiesByCNPJValidatesInputAndField(t *testing.T) {
	service := NewGLPIService("http://unused", "app-token")
	if _, err := service.FindEntitiesByCNPJ("session", "123"); err == nil {
		t.Fatal("expected invalid CNPJ error")
	}

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(response).Encode(map[string]interface{}{
			"2": map[string]string{"name": "ID", "uid": "Entity.id"},
		})
	}))
	defer server.Close()

	service = NewGLPIService(server.URL, "app-token")
	_, err := service.FindEntitiesByCNPJ("session", "12345678000199")
	if err == nil || !strings.Contains(err.Error(), "campo CNPJ") {
		t.Fatalf("error = %v", err)
	}
}

func TestFormatCNPJ(t *testing.T) {
	if got := formatCNPJ("12345678000199"); got != "12.345.678/0001-99" {
		t.Fatalf("formatCNPJ = %q", got)
	}
}
