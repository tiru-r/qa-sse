package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"abt-dashboard/internal/models"
	"abt-dashboard/internal/server"
	"abt-dashboard/internal/services"
)

// Test helper to create analytics with test data
func newTestAnalytics() *services.Analytics {
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
		{
			TransactionID: "T003",
			Date:          time.Date(2023, 3, 5, 0, 0, 0, 0, time.UTC),
			UserID:        "U003",
			Country:       "UK",
			Region:        "London",
			ProductID:     "P003",
			ProductName:   "Keyboard",
			Category:      "Electronics",
			Price:         79.99,
			Quantity:      1,
			TotalPrice:    79.99,
			Stock:         75,
			AddedDate:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	a.SetData(testData)
	return a
}

// Integration tests for HTTP routes
func TestServer_Routes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	templateHandlers := &server.TemplateHandlers{Dashboard: handleDashboard}
	srv := server.NewServer(newTestAnalytics(), logger, templateHandlers)

	tests := []struct {
		path           string
		expectedStatus int
		contentType    string
	}{
		{"/", http.StatusOK, "text/html"},
		{"/api/country-revenue", http.StatusOK, "application/json"},
		{"/api/top-products", http.StatusOK, "application/json"},
		{"/api/monthly-sales", http.StatusOK, "application/json"},
		{"/api/top-regions", http.StatusOK, "application/json"},
		{"/health", http.StatusOK, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tt.path, nil)

			srv.ServeHTTP(w, r)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}

			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, tt.contentType) {
				t.Errorf("content-type = %q, want %q", ct, tt.contentType)
			}

			// Validate JSON responses
			if tt.contentType == "application/json" {
				var result any
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Errorf("invalid json: %v", err)
				}
			}
		})
	}
}

// Test JSON API responses
func TestServer_JSONResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	templateHandlers := &server.TemplateHandlers{Dashboard: handleDashboard}
	srv := server.NewServer(newTestAnalytics(), logger, templateHandlers)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/top-products", nil)
	srv.ServeHTTP(w, r)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array in response")
	}

	if len(data) == 0 {
		t.Error("expected products data")
		return
	}

	// Verify structure of first item
	if item, ok := data[0].(map[string]interface{}); ok {
		if name, hasName := item["product_name"].(string); !hasName || name == "" {
			t.Error("product should have non-empty product_name field")
		}
		if category, hasCat := item["category"].(string); !hasCat || category == "" {
			t.Error("product should have non-empty category field")
		}
		if freq, hasFreq := item["frequency"].(float64); !hasFreq || freq < 0 {
			t.Error("product should have non-negative frequency field")
		}
	} else {
		t.Error("invalid product structure")
	}
}

// Test Server-Sent Events routes
func TestServer_SSERoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	templateHandlers := &server.TemplateHandlers{Dashboard: handleDashboard}
	srv := server.NewServer(newTestAnalytics(), logger, templateHandlers)

	sseRoutes := []string{
		"/sse/country-revenue",
		"/sse/top-products",
		"/sse/monthly-sales",
		"/sse/top-regions",
	}

	for _, route := range sseRoutes {
		t.Run(route, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", route, nil)

			srv.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			// Check for SSE headers
			if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
				t.Errorf("content-type = %q, should contain 'text/event-stream'", ct)
			}

			if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
				t.Errorf("cache-control = %q, want 'no-cache'", cc)
			}
		})
	}
}

// Test health endpoint
func TestServer_HandleHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	templateHandlers := &server.TemplateHandlers{Dashboard: handleDashboard}
	srv := server.NewServer(newTestAnalytics(), logger, templateHandlers)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode health JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success=true in response")
	}

	healthData, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected health data in response")
	}

	if status, ok := healthData["status"].(string); !ok || status != "healthy" {
		t.Errorf("health status = %v, want 'healthy'", healthData["status"])
	}

	if _, ok := healthData["timestamp"]; !ok {
		t.Error("health response should include timestamp")
	}
}

// Test error handling for invalid methods
func TestServer_ErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	templateHandlers := &server.TemplateHandlers{Dashboard: handleDashboard}
	srv := server.NewServer(newTestAnalytics(), logger, templateHandlers)

	tests := []struct {
		method string
		path   string
		status int
	}{
		{"POST", "/api/country-revenue", http.StatusMethodNotAllowed},
		{"PUT", "/", http.StatusMethodNotAllowed},
		{"DELETE", "/health", http.StatusMethodNotAllowed},
		{"PATCH", "/api/top-products", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, tt.path, nil)

			srv.ServeHTTP(w, r)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
		})
	}
}

// Test dashboard template rendering
func TestDashboardTemplate(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Test the template handler directly
	handleDashboard(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "ABT Corporation Dashboard") {
		t.Error("dashboard should contain title")
	}

	if !strings.Contains(body, "Real-time business analytics") {
		t.Error("dashboard should contain subtitle")
	}

	// Check for key dashboard components
	expectedComponents := []string{
		"Country Revenue Analysis",
		"Top 20 Products by Transactions",
		"Monthly Sales Volume",
		"Top 30 Regions by Revenue",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(body, component) {
			t.Errorf("dashboard should contain '%s'", component)
		}
	}
}
