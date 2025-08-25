package handlers

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"abt-dashboard/internal/models"
)

func TestNewSSEHandlers(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	handlers := NewSSEHandlers(analytics, logger)

	if handlers == nil {
		t.Error("NewSSEHandlers() returned nil")
	}

	if handlers.analytics != analytics {
		t.Error("NewSSEHandlers() should set analytics field")
	}

	if handlers.logger != logger {
		t.Error("NewSSEHandlers() should set logger field")
	}
}

func TestSSEHandlers_renderCountryTable(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	testData := []models.CountryRevenue{
		{
			Country:      "USA",
			ProductName:  "Laptop",
			Category:     "Electronics",
			TotalRevenue: 999.99,
			Transactions: 1,
		},
		{
			Country:      "Canada",
			ProductName:  "Mouse",
			Category:     "Electronics",
			TotalRevenue: 59.98,
			Transactions: 2,
		},
	}

	html, err := handlers.renderCountryTable(testData)
	if err != nil {
		t.Fatalf("renderCountryTable() failed: %v", err)
	}

	// Check that HTML contains expected elements
	expectedContent := []string{
		"<table class=\"modern-table\">",
		"<thead>",
		"<th>Country</th>",
		"<th>Product</th>",
		"<th>Category</th>",
		"<th>Revenue</th>",
		"<th>Orders</th>",
		"USA",
		"Laptop",
		"Electronics",
		"999.99",
		"Canada",
		"Mouse",
		"59.98",
	}

	for _, content := range expectedContent {
		if !strings.Contains(html, content) {
			t.Errorf("expected HTML to contain %q", content)
		}
	}
}

func TestSSEHandlers_renderCountryTable_LargeDataset(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	// Create dataset larger than maxTableRows (50)
	testData := make([]models.CountryRevenue, 75)
	for i := 0; i < 75; i++ {
		testData[i] = models.CountryRevenue{
			Country:      "Country" + string(rune(i)),
			ProductName:  "Product" + string(rune(i)),
			Category:     "Category",
			TotalRevenue: float64(i * 10),
			Transactions: i,
		}
	}

	html, err := handlers.renderCountryTable(testData)
	if err != nil {
		t.Fatalf("renderCountryTable() failed: %v", err)
	}

	// Count table rows - should be limited to maxTableRows (50)
	rowCount := strings.Count(html, "<tr>") - 1 // Subtract header row
	if rowCount > maxTableRows {
		t.Errorf("expected max %d rows, got %d", maxTableRows, rowCount)
	}
}

func TestSSEHandlers_HandleCountryRevenue(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/sse/country-revenue", nil)
	w := httptest.NewRecorder()

	handlers.HandleCountryRevenue(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check SSE headers (DataStar sets these)
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected content-type to contain 'text/event-stream', got %q", ct)
	}

	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected cache-control 'no-cache', got %q", cc)
	}

	// The DataStar library formats SSE events, just check we got some response
	body := w.Body.String()
	if body == "" {
		t.Error("response should not be empty")
	}

	// The response should contain HTML table data somewhere in the SSE stream
	if !strings.Contains(body, "<table") {
		t.Error("response should contain HTML table")
	}
}

func TestSSEHandlers_HandleTopProducts(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/sse/top-products", nil)
	w := httptest.NewRecorder()

	handlers.HandleTopProducts(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()

	// Should contain some SSE data
	if body == "" {
		t.Error("response should not be empty")
	}

	// Should contain products data or success message somewhere in SSE stream
	if !strings.Contains(body, "productsData") && !strings.Contains(body, "Products chart data loaded") {
		t.Error("response should contain productsData signal or success message")
	}
}

func TestSSEHandlers_HandleMonthlySales(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/sse/monthly-sales", nil)
	w := httptest.NewRecorder()

	handlers.HandleMonthlySales(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()

	// Should contain monthly data signal
	if !strings.Contains(body, "monthlyData") {
		t.Error("response should contain monthlyData signal")
	}

	// Should contain success message
	if !strings.Contains(body, "Monthly sales chart data loaded") {
		t.Error("response should contain success message")
	}
}

func TestSSEHandlers_HandleTopRegions(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/sse/top-regions", nil)
	w := httptest.NewRecorder()

	handlers.HandleTopRegions(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()

	// Should contain regions data signal
	if !strings.Contains(body, "regionsData") {
		t.Error("response should contain regionsData signal")
	}

	// Should contain success message
	if !strings.Contains(body, "Regions chart data loaded") {
		t.Error("response should contain success message")
	}
}

func TestSSEHandlers_HandleRefreshAll(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	req := httptest.NewRequest(http.MethodGet, "/sse/refresh-all", nil)
	w := httptest.NewRecorder()

	handlers.HandleRefreshAll(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()

	// Should contain all data signals
	expectedSignals := []string{
		"productsData",
		"monthlyData",
		"regionsData",
	}

	for _, signal := range expectedSignals {
		if !strings.Contains(body, signal) {
			t.Errorf("response should contain %q signal", signal)
		}
	}

	// Should also contain HTML table data (from country revenue)
	if !strings.Contains(body, "<table") {
		t.Error("response should contain HTML table for country revenue")
	}
}

// Test template data structure
func TestTemplateData(t *testing.T) {
	data := templateData{
		Data:    []string{"test1", "test2"},
		MaxRows: 50,
	}

	if data.MaxRows != 50 {
		t.Errorf("expected MaxRows=50, got %d", data.MaxRows)
	}

	slice, ok := data.Data.([]string)
	if !ok {
		t.Error("Data should be of type []string")
	}

	if len(slice) != 2 {
		t.Errorf("expected 2 items, got %d", len(slice))
	}
}

// Test SSE headers consistency
func TestSSEHandlers_HeaderConsistency(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	sseEndpoints := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"country-revenue", handlers.HandleCountryRevenue},
		{"top-products", handlers.HandleTopProducts},
		{"monthly-sales", handlers.HandleMonthlySales},
		{"top-regions", handlers.HandleTopRegions},
		{"refresh-all", handlers.HandleRefreshAll},
	}

	for _, endpoint := range sseEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			endpoint.handler(w, req)

			// All SSE endpoints should have consistent headers
			if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
				t.Errorf("expected content-type to contain 'text/event-stream', got %q", ct)
			}

			if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
				t.Errorf("expected cache-control 'no-cache', got %q", cc)
			}

			// Should return some SSE data
			body := w.Body.String()
			if !strings.Contains(body, "event:") || !strings.Contains(body, "data:") {
				t.Error("response should contain SSE event format")
			}
		})
	}
}

// Test data limits and constants
func TestSSEConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"maxTableRows", maxTableRows, 50},
		{"maxProducts", maxProducts, 20},
		{"maxRegions", maxRegions, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s=%d, got %d", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

// Test basic handler functionality without nil analytics
// (handlers expect valid analytics and will panic otherwise, which is acceptable)
func TestSSEHandlers_BasicFunctionality(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"country-revenue", handlers.HandleCountryRevenue},
		{"top-products", handlers.HandleTopProducts},
		{"monthly-sales", handlers.HandleMonthlySales},
		{"top-regions", handlers.HandleTopRegions},
		{"refresh-all", handlers.HandleRefreshAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Should not panic with valid analytics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("handler panicked: %v", r)
				}
			}()

			tt.handler(w, req)

			// Should return OK status
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			// Should return some content
			if w.Body.Len() == 0 {
				t.Error("expected non-empty response")
			}
		})
	}
}

// Test data signal content (simplified)
func TestSSEHandlers_DataSignals(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		signalKey string
	}{
		{"top-products", handlers.HandleTopProducts, "productsData"},
		{"monthly-sales", handlers.HandleMonthlySales, "monthlyData"},
		{"top-regions", handlers.HandleTopRegions, "regionsData"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			body := w.Body.String()

			// Just check that the signal key appears somewhere in the response
			// (DataStar formats the SSE events, so we don't need to parse the exact format)
			if !strings.Contains(body, tt.signalKey) {
				t.Errorf("response should contain %q signal", tt.signalKey)
			}
		})
	}
}

// Test template execution edge cases
func TestSSEHandlers_TemplateEdgeCases(t *testing.T) {
	analytics := createTestAnalytics()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := NewSSEHandlers(analytics, logger)

	tests := []struct {
		name string
		data interface{}
	}{
		{"empty slice", []models.CountryRevenue{}},
		{"nil data", nil},
		{"single item", []models.CountryRevenue{
			{
				Country:      "Test",
				ProductName:  "Test Product",
				Category:     "Test Category",
				TotalRevenue: 100.0,
				Transactions: 1,
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := handlers.renderCountryTable(tt.data)

			// Should not error (template should handle edge cases gracefully)
			if err != nil {
				t.Errorf("renderCountryTable should not error with %s: %v", tt.name, err)
			}

			// Should still produce valid HTML structure
			if !strings.Contains(html, "<table") || !strings.Contains(html, "</table>") {
				t.Errorf("should produce valid table HTML for %s", tt.name)
			}
		})
	}
}
