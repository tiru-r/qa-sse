package services

import (
	"context"
	"os"
	"testing"
	"time"

	"abt-dashboard/internal/models"
)

func createTempCSV(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestNewAnalytics(t *testing.T) {
	a := NewAnalytics()
	if a == nil {
		t.Error("NewAnalytics() returned nil")
	}
	if a.precomputed == nil {
		t.Error("precomputed should be initialized")
	}
	if a.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestAnalytics_SetData(t *testing.T) {
	a := NewAnalytics()
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
			Date:          time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC),
			UserID:        "U002",
			Country:       "USA",
			Region:        "Texas",
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

	// Test that data was set
	if a.precomputed.RecordCount != 2 {
		t.Errorf("Expected RecordCount = 2, got %d", a.precomputed.RecordCount)
	}

	// Test analytics methods work
	countryRevenue := a.CountryRevenue()
	if len(countryRevenue) == 0 {
		t.Error("CountryRevenue() should return data")
	}

	topProducts := a.TopProducts(20)
	if len(topProducts) == 0 {
		t.Error("TopProducts() should return data")
	}

	monthlySales := a.MonthlySales()
	if len(monthlySales) == 0 {
		t.Error("MonthlySales() should return data")
	}

	topRegions := a.TopRegions(30)
	if len(topRegions) == 0 {
		t.Error("TopRegions() should return data")
	}
}

func TestAnalytics_LoadFromCSV_ValidData(t *testing.T) {
	validCSV := `transaction_id,transaction_date,user_id,country,region,product_id,product_name,category,price,quantity,total_price,stock,added_date
T001,2023-01-15,U001,USA,California,P001,Laptop,Electronics,999.99,1,999.99,50,2023-01-01
T002,2023-01-16,U002,Canada,Ontario,P002,Mouse,Electronics,29.99,2,59.98,100,2023-01-01`

	f := createTempCSV(t, validCSV)
	defer os.Remove(f)

	a := NewAnalytics()
	err := a.LoadFromCSV(context.Background(), f)

	if err != nil {
		t.Errorf("LoadFromCSV() with valid data should not error, got: %v", err)
	}

	// Verify data was loaded
	countryRevenue := a.CountryRevenue()
	if len(countryRevenue) == 0 {
		t.Error("Should have loaded country revenue data")
	}
}

func TestAnalytics_LoadFromCSV_InvalidData(t *testing.T) {
	tests := []struct {
		name    string
		csv     string
		wantErr bool
	}{
		{
			name:    "empty file",
			csv:     "",
			wantErr: true,
		},
		{
			name:    "header only",
			csv:     "h1,h2,h3,h4,h5,h6,h7,h8,h9,h10,h11,h12,h13",
			wantErr: true,
		},
		{
			name:    "invalid date format",
			csv:     "h1,date,h3,country,region,h6,product,h8,price,qty,total,stock,added\n1,invalid-date,u1,US,CA,p1,Laptop,cat,100.0,2,200.0,50,2022-01-01",
			wantErr: true,
		},
		{
			name:    "invalid price",
			csv:     "h1,date,h3,country,region,h6,product,h8,price,qty,total,stock,added\n1,2023-01-01,u1,US,CA,p1,Laptop,cat,invalid,2,200.0,50,2022-01-01",
			wantErr: true,
		},
		{
			name:    "invalid quantity",
			csv:     "h1,date,h3,country,region,h6,product,h8,price,qty,total,stock,added\n1,2023-01-01,u1,US,CA,p1,Laptop,cat,100.0,invalid,200.0,50,2022-01-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := createTempCSV(t, tt.csv)
			defer os.Remove(f)

			a := NewAnalytics()
			err := a.LoadFromCSV(context.Background(), f)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromCSV() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalytics_CountryRevenue(t *testing.T) {
	a := NewAnalytics()
	testData := []models.Transaction{
		{
			Country:     "USA",
			ProductName: "Laptop",
			Category:    "Electronics",
			TotalPrice:  999.99,
		},
		{
			Country:     "USA",
			ProductName: "Laptop",
			Category:    "Electronics",
			TotalPrice:  999.99,
		},
		{
			Country:     "Canada",
			ProductName: "Mouse",
			Category:    "Electronics",
			TotalPrice:  29.99,
		},
	}

	a.SetData(testData)
	result := a.CountryRevenue()

	if len(result) == 0 {
		t.Error("CountryRevenue() should return data")
	}

	// Should be sorted by total revenue descending
	if len(result) >= 2 && result[0].TotalRevenue < result[1].TotalRevenue {
		t.Error("CountryRevenue() should be sorted by total revenue descending")
	}
}

func TestAnalytics_TopProducts(t *testing.T) {
	a := NewAnalytics()
	testData := []models.Transaction{
		{
			ProductName: "Laptop",
			Category:    "Electronics",
			Stock:       50,
		},
		{
			ProductName: "Laptop", // Same product, should aggregate
			Category:    "Electronics",
			Stock:       50,
		},
		{
			ProductName: "Mouse",
			Category:    "Electronics",
			Stock:       100,
		},
	}

	a.SetData(testData)
	result := a.TopProducts(20)

	if len(result) == 0 {
		t.Error("TopProducts() should return data")
	}

	// Should be sorted by frequency descending
	if len(result) >= 2 && result[0].Frequency < result[1].Frequency {
		t.Error("TopProducts() should be sorted by frequency descending")
	}

	// Should limit to 20
	if len(result) > 20 {
		t.Error("TopProducts() should limit to 20 results")
	}
}

func TestAnalytics_MonthlySales(t *testing.T) {
	a := NewAnalytics()
	testData := []models.Transaction{
		{
			Date:       time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			TotalPrice: 999.99,
		},
		{
			Date:       time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC),
			TotalPrice: 29.99,
		},
		{
			Date:       time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
			TotalPrice: 199.99,
		},
	}

	a.SetData(testData)
	result := a.MonthlySales()

	if len(result) == 0 {
		t.Error("MonthlySales() should return data")
	}

	// Should aggregate by month
	found2023_01 := false
	found2023_02 := false
	for _, r := range result {
		if r.Month == "2023-01" {
			found2023_01 = true
			// Should sum January transactions: 999.99 + 29.99 = 1029.98
			if r.Volume < 1029.0 {
				t.Errorf("January volume should be ~1029.98, got %f", r.Volume)
			}
		}
		if r.Month == "2023-02" {
			found2023_02 = true
		}
	}

	if !found2023_01 {
		t.Error("Should find 2023-01 data")
	}
	if !found2023_02 {
		t.Error("Should find 2023-02 data")
	}
}

func TestAnalytics_TopRegions(t *testing.T) {
	a := NewAnalytics()
	testData := []models.Transaction{
		{
			Region:     "California",
			TotalPrice: 999.99,
			Quantity:   1,
		},
		{
			Region:     "California", // Same region, should aggregate
			TotalPrice: 29.99,
			Quantity:   1,
		},
		{
			Region:     "Texas",
			TotalPrice: 199.99,
			Quantity:   2,
		},
	}

	a.SetData(testData)
	result := a.TopRegions(30)

	if len(result) == 0 {
		t.Error("TopRegions() should return data")
	}

	// Should be sorted by revenue descending
	if len(result) >= 2 && result[0].Revenue < result[1].Revenue {
		t.Error("TopRegions() should be sorted by revenue descending")
	}

	// Should limit to 30
	if len(result) > 30 {
		t.Error("TopRegions() should limit to 30 results")
	}
}

func TestAnalytics_ConcurrentAccess(t *testing.T) {
	a := NewAnalytics()
	testData := []models.Transaction{
		{
			Country:     "USA",
			ProductName: "Laptop",
			Category:    "Electronics",
			TotalPrice:  999.99,
			Region:      "California",
			Date:        time.Now(),
			Quantity:    1,
		},
	}

	a.SetData(testData)

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// These should not panic or return inconsistent data
			_ = a.CountryRevenue()
			_ = a.TopProducts(20)
			_ = a.MonthlySales()
			_ = a.TopRegions(30)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestAnalytics_EmptyData(t *testing.T) {
	a := NewAnalytics()

	// Test methods with no data - they should return empty slices, not nil
	countryRevenue := a.CountryRevenue()
	if len(countryRevenue) != 0 {
		t.Errorf("CountryRevenue() should return empty slice, got length %d", len(countryRevenue))
	}

	topProducts := a.TopProducts(20)
	if len(topProducts) != 0 {
		t.Errorf("TopProducts() should return empty slice, got length %d", len(topProducts))
	}

	monthlySales := a.MonthlySales()
	if len(monthlySales) != 0 {
		t.Errorf("MonthlySales() should return empty slice, got length %d", len(monthlySales))
	}

	topRegions := a.TopRegions(30)
	if len(topRegions) != 0 {
		t.Errorf("TopRegions() should return empty slice, got length %d", len(topRegions))
	}
}

// Benchmark tests for performance validation
func BenchmarkAnalytics_CountryRevenue(b *testing.B) {
	a := NewAnalytics()
	testData := make([]models.Transaction, 1000)
	for i := 0; i < 1000; i++ {
		testData[i] = models.Transaction{
			Country:     "USA",
			ProductName: "Product" + string(rune(i%100)),
			Category:    "Electronics",
			TotalPrice:  float64(i) * 10.0,
		}
	}
	a.SetData(testData)

	b.ResetTimer()
	for b.Loop() {
		_ = a.CountryRevenue()
	}
}

func BenchmarkAnalytics_TopProducts(b *testing.B) {
	a := NewAnalytics()
	testData := make([]models.Transaction, 1000)
	for i := 0; i < 1000; i++ {
		testData[i] = models.Transaction{
			ProductName: "Product" + string(rune(i%50)),
			Category:    "Electronics",
			Stock:       i % 100,
		}
	}
	a.SetData(testData)

	b.ResetTimer()
	for b.Loop() {
		_ = a.TopProducts(20)
	}
}
