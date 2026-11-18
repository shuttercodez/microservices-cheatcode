package product

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestProductHandler_Endpoints tests HTTP controller endpoints (equivalent to WebApplicationFactory / MockMvc).
func TestProductHandler_Endpoints(t *testing.T) {
	// Arrange: wire up Mock Repo + Mock Cache + Mock Producer -> Service -> Handler -> Fiber App
	repo := newMockRepository()
	cache := newMockCache()
	producer := &mockProducer{}
	svc := NewService(repo, cache, producer)
	handler := NewHandler(svc)

	app := fiber.New()
	handler.RegisterRoutes(app)

	var createdID string

	t.Run("POST /api/v1/products - creates product via HTTP (201 Created)", func(t *testing.T) {
		payload := `{"sku":"IPHONE-16","name":"iPhone 16 Pro","price":39900,"stock":50}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", resp.StatusCode)
		}

		var body Product
		_ = json.NewDecoder(resp.Body).Decode(&body)
		createdID = body.ID.String()

		if body.SKU != "IPHONE-16" || body.Stock != 50 {
			t.Errorf("unexpected response body: %+v", body)
		}
	})

	t.Run("GET /api/v1/products/:id - retrieves newly created product (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+createdID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("PATCH /api/v1/products/:id/stock - updates product stock (200 OK)", func(t *testing.T) {
		payload := `{"stock":45}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/products/"+createdID+"/stock", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var body Product
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.Stock != 45 || body.Version != 2 {
			t.Errorf("expected stock 45 and version 2, got stock=%d version=%d", body.Stock, body.Version)
		}
	})
}
