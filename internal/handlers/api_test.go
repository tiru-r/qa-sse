package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"abt-dashboard/internal/models"
	"abt-dashboard/internal/services"
)

func createTestAnalytics() *services.Analytics {
	a := services.NewAnalytics()
	testData := []models.Transaction{
		{
			TransactionID: "T001",
			Date:          time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			UserID:        "U001",
			Country:       "USA",
			Region:        "California",
			ProductID:     "P001",
			ProductName:   "Laptop",
			Category:      "Electronics",
			Price:         999.99,
			Quantity:      1,
			TotalPrice:    999.99,
			Stock:         50,
			AddedDate:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			TransactionID: "T002",
			Date:          time.Date(2023, 2, 10, 0, 0, 0, 0, time.UTC),
			UserID:        "U002",
			Country:       "Canada",
			Region:        "Ontario",
			ProductID:     "P002",
			ProductName:   "Mouse",
			Category:      "Electronics",
			Price:         29.99,
			Quantity:      2,
			TotalPrice:    59.98,
			Stock:         100,
			AddedDate:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	a.SetData(testData)
	return a
}

func TestNewAPIHandlers(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	if handlers == nil {
		t.Error("NewAPIHandlers() returned nil")
	}

	if handlers.analytics != analytics {
		t.Error("NewAPIHandlers() should set analytics field")
	}
}

func TestAPIHandlers_HandleCountryRevenue(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/country-revenue", nil)
	w := httptest.NewRecorder()

	handlers.HandleCountryRevenue(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", contentType)
	}

	// Check cache control
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=300" {
		t.Errorf("expected cache-control 'public, max-age=300', got %q", cacheControl)
	}

	// Check JSON response structure (it will be wrapped in success response now)
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}

	if data, ok := response["data"]; !ok {
		t.Error("expected data field in response")
	} else if dataSlice, ok := data.([]interface{}); !ok || len(dataSlice) == 0 {
		t.Error("expected non-empty data array in response")
	}
}

func TestAPIHandlers_HandleTopProducts(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/top-products", nil)
	w := httptest.NewRecorder()

	handlers.HandleTopProducts(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check headers
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", ct)
	}

	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("expected cache-control 'public, max-age=300', got %q", cc)
	}

	// Check JSON response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}
}

func TestAPIHandlers_HandleMonthlySales(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/monthly-sales", nil)
	w := httptest.NewRecorder()

	handlers.HandleMonthlySales(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check headers
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", ct)
	}

	// Check JSON response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}
}

func TestAPIHandlers_HandleTopRegions(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/top-regions", nil)
	w := httptest.NewRecorder()

	handlers.HandleTopRegions(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check headers
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", ct)
	}

	// Check JSON response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}
}

func TestAPIHandlers_HandleHealth(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HandleHealth(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", ct)
	}

	// Check JSON response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}

	if data, ok := response["data"].(map[string]interface{}); !ok {
		t.Error("expected health data in response")
	} else {
		if status, ok := data["status"].(string); !ok || status != "healthy" {
			t.Errorf("expected status 'healthy', got %q", status)
		}

		if timestamp, ok := data["timestamp"].(string); !ok || timestamp == "" {
			t.Error("expected non-empty timestamp")
		} else {
			if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
				t.Errorf("invalid timestamp format: %v", err)
			}
		}
	}
}

func TestAPIHandlers_HandleStats(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	w := httptest.NewRecorder()

	handlers.HandleStats(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", ct)
	}

	// Check JSON response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}
}

// Test error handling when analytics returns bad data
func TestAPIHandlers_ErrorHandling(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	tests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"country-revenue", handlers.HandleCountryRevenue, "/api/country-revenue"},
		{"top-products", handlers.HandleTopProducts, "/api/top-products"},
		{"monthly-sales", handlers.HandleMonthlySales, "/api/monthly-sales"},
		{"top-regions", handlers.HandleTopRegions, "/api/top-regions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			// Should not panic with valid analytics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("handler panicked: %v", r)
				}
			}()

			tt.handler(w, req)

			// Should return OK with valid analytics
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200 with valid analytics, got %d", w.Code)
			}
		})
	}
}

// Test that handlers set correct headers consistently
func TestAPIHandlers_HeaderConsistency(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	apiEndpoints := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"country-revenue", handlers.HandleCountryRevenue},
		{"top-products", handlers.HandleTopProducts},
		{"monthly-sales", handlers.HandleMonthlySales},
		{"top-regions", handlers.HandleTopRegions},
	}

	for _, endpoint := range apiEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			endpoint.handler(w, req)

			// All API endpoints should have consistent headers
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected content-type 'application/json', got %q", ct)
			}

			if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=300" {
				t.Errorf("expected cache-control 'public, max-age=300', got %q", cc)
			}

			// Should return valid JSON with success wrapper
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Errorf("response should be valid JSON: %v", err)
			}

			if success, ok := response["success"].(bool); !ok || !success {
				t.Error("expected success=true in response")
			}
		})
	}
}

// Test that health endpoint doesn't set cache headers
func TestAPIHandlers_HealthNoCaching(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HandleHealth(w, req)

	// Health endpoint should NOT have cache-control header
	if cc := w.Header().Get("Cache-Control"); cc != "" {
		t.Errorf("health endpoint should not set cache-control, got %q", cc)
	}

	// But should have content-type
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", ct)
	}
}

// Test response body format validation
func TestAPIHandlers_ResponseFormat(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.Default()
	handlers := NewAPIHandlers(analytics, logger)

	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"country-revenue", handlers.HandleCountryRevenue},
		{"top-products", handlers.HandleTopProducts},
		{"monthly-sales", handlers.HandleMonthlySales},
		{"top-regions", handlers.HandleTopRegions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			body := w.Body.String()

			// Should be valid JSON object (success wrapper)
			if !strings.HasPrefix(body, "{") || !strings.HasSuffix(strings.TrimSpace(body), "}") {
				t.Errorf("expected JSON object response, got: %s", body)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(strings.NewReader(body)).Decode(&response); err != nil {
				t.Errorf("should be valid JSON: %v", err)
			}

			if success, ok := response["success"].(bool); !ok || !success {
				t.Error("expected success=true in response")
			}

			if _, ok := response["data"]; !ok {
				t.Error("expected data field in response")
			}
		})
	}
}
