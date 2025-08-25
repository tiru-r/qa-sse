package services

import (
	"bufio"
	"context"
	"encoding/gob"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"abt-dashboard/internal/models"
	"golang.org/x/sync/errgroup"
)

const (
	batchSize    = 10000
	maxWorkers   = 10
	cacheVersion = "v1"
	cacheDir     = ".cache"
)

type PrecomputedData struct {
	CountryRevenue []models.CountryRevenue   `json:"country_revenue"`
	TopProducts    []models.ProductFrequency `json:"top_products"`
	MonthlySales   []models.MonthlyData      `json:"monthly_sales"`
	TopRegions     []models.RegionRevenue    `json:"top_regions"`
	LastModified   time.Time                 `json:"last_modified"`
	RecordCount    int64                     `json:"record_count"`
}

type Analytics struct {
	mu               sync.RWMutex
	precomputed      *PrecomputedData
	csvPath          string
	recordsProcessed atomic.Int64
	logger           *slog.Logger
}

func NewAnalytics() *Analytics {
	logger := slog.Default()
	return &Analytics{
		precomputed: &PrecomputedData{},
		logger:      logger,
	}
}

func (a *Analytics) SetData(data []models.Transaction) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Convert transaction data to precomputed format for tests
	a.precomputed = a.computeAnalytics(data)
	a.precomputed.RecordCount = int64(len(data))
	a.precomputed.LastModified = time.Now()
}

func (a *Analytics) LoadFromCSV(ctx context.Context, filename string) error {
	a.csvPath = filename

	// Check if we have a valid cache
	if cached, err := a.loadFromCache(filename); err == nil {
		fileInfo, err := os.Stat(filename)
		if err == nil && fileInfo.ModTime().Before(cached.LastModified) {
			a.mu.Lock()
			a.precomputed = cached
			a.mu.Unlock()
			a.logger.Info("loaded from cache", "records", cached.RecordCount)
			return nil
		}
	}

	start := time.Now()
	a.logger.Info("processing CSV file", "filename", filename)

	// Stream process the CSV file
	if err := a.streamProcessCSV(ctx, filename); err != nil {
		return fmt.Errorf("process csv: %w", err)
	}

	// Save to cache
	if err := a.saveToCache(filename); err != nil {
		a.logger.Warn("failed to save cache", "error", err)
	}

	duration := time.Since(start)
	count := a.recordsProcessed.Load()
	a.logger.Info("csv processing complete",
		"records", count,
		"duration", duration,
		"rate", fmt.Sprintf("%.0f records/sec", float64(count)/duration.Seconds()))

	return nil
}

func (a *Analytics) streamProcessCSV(ctx context.Context, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer

	// Skip header
	if !scanner.Scan() {
		return fmt.Errorf("empty file")
	}

	// Aggregation maps for efficient processing
	countryGroups := make(map[string]*models.CountryRevenue)
	productGroups := make(map[string]*models.ProductFrequency)
	monthlyGroups := make(map[string]float64)
	regionGroups := make(map[string]*models.RegionRevenue)

	var mu sync.Mutex
	recordCount := int64(0)

	// Process in batches
	batch := make([]string, 0, batchSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batch = append(batch, scanner.Text())

		if len(batch) >= batchSize {
			if err := a.processBatch(ctx, batch, &mu, countryGroups, productGroups, monthlyGroups, regionGroups, &recordCount); err != nil {
				return err
			}
			batch = batch[:0] // Reset batch
		}
	}

	// Process remaining records
	if len(batch) > 0 {
		if err := a.processBatch(ctx, batch, &mu, countryGroups, productGroups, monthlyGroups, regionGroups, &recordCount); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	// Check if we processed any valid records
	if recordCount == 0 {
		return fmt.Errorf("no valid records found")
	}

	// Convert maps to sorted slices
	precomputed := &PrecomputedData{
		CountryRevenue: a.sortCountryRevenue(countryGroups),
		TopProducts:    a.sortTopProducts(productGroups),
		MonthlySales:   a.sortMonthlySales(monthlyGroups),
		TopRegions:     a.sortTopRegions(regionGroups),
		RecordCount:    recordCount,
		LastModified:   time.Now(),
	}

	a.mu.Lock()
	a.precomputed = precomputed
	a.mu.Unlock()

	a.recordsProcessed.Store(recordCount)
	return nil
}

func (a *Analytics) processBatch(ctx context.Context, batch []string, mu *sync.Mutex,
	countryGroups map[string]*models.CountryRevenue,
	productGroups map[string]*models.ProductFrequency,
	monthlyGroups map[string]float64,
	regionGroups map[string]*models.RegionRevenue,
	recordCount *int64) error {

	var wg errgroup.Group
	wg.SetLimit(maxWorkers)

	// Channel to collect processed transactions
	type processedTx struct {
		tx    models.Transaction
		valid bool
	}

	txChan := make(chan processedTx, len(batch))

	for _, line := range batch {
		wg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			record := strings.Split(line, ",")
			tx, err := parseTransactionFast(record)
			if err != nil {
				txChan <- processedTx{valid: false}
				return nil // Skip invalid records
			}

			txChan <- processedTx{tx: tx, valid: true}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		close(txChan)
		return err
	}
	close(txChan)

	// Process all transactions sequentially to avoid race conditions
	localCountry := make(map[string]*models.CountryRevenue)
	localProduct := make(map[string]*models.ProductFrequency)
	localMonthly := make(map[string]float64)
	localRegion := make(map[string]*models.RegionRevenue)
	localCount := int64(0)

	for ptx := range txChan {
		if ptx.valid {
			a.aggregateTransaction(ptx.tx, localCountry, localProduct, localMonthly, localRegion)
			localCount++
		}
	}

	// Merge local results into global maps
	mu.Lock()
	a.mergeResults(localCountry, countryGroups)
	a.mergeProductResults(localProduct, productGroups)
	a.mergeMonthlyResults(localMonthly, monthlyGroups)
	a.mergeRegionResults(localRegion, regionGroups)
	*recordCount += localCount
	mu.Unlock()

	return nil
}

func parseTransactionFast(record []string) (models.Transaction, error) {
	if len(record) < 13 {
		return models.Transaction{}, fmt.Errorf("insufficient columns")
	}

	// Only parse fields we actually need for aggregation
	transactionDate, err := time.Parse("2006-01-02", strings.TrimSpace(record[1]))
	if err != nil {
		return models.Transaction{}, err
	}

	price, err := strconv.ParseFloat(strings.TrimSpace(record[8]), 64)
	if err != nil {
		return models.Transaction{}, err
	}

	quantity, err := strconv.Atoi(strings.TrimSpace(record[9]))
	if err != nil {
		return models.Transaction{}, err
	}

	totalPrice, err := strconv.ParseFloat(strings.TrimSpace(record[10]), 64)
	if err != nil {
		return models.Transaction{}, err
	}

	stock, err := strconv.Atoi(strings.TrimSpace(record[11]))
	if err != nil {
		return models.Transaction{}, err
	}

	return models.Transaction{
		Date:        transactionDate,
		Country:     strings.TrimSpace(record[3]),
		Region:      strings.TrimSpace(record[4]),
		ProductName: strings.TrimSpace(record[6]),
		Category:    strings.TrimSpace(record[7]),
		Price:       price,
		Quantity:    quantity,
		TotalPrice:  totalPrice,
		Stock:       stock,
	}, nil
}

func (a *Analytics) aggregateTransaction(tx models.Transaction,
	countryGroups map[string]*models.CountryRevenue,
	productGroups map[string]*models.ProductFrequency,
	monthlyGroups map[string]float64,
	regionGroups map[string]*models.RegionRevenue) {

	// Country revenue aggregation
	countryKey := tx.Country + "|" + tx.ProductName + "|" + tx.Category
	if countryGroups[countryKey] == nil {
		countryGroups[countryKey] = &models.CountryRevenue{
			Country:     tx.Country,
			ProductName: tx.ProductName,
			Category:    tx.Category,
		}
	}
	countryGroups[countryKey].TotalRevenue += tx.TotalPrice
	countryGroups[countryKey].Transactions++

	// Product frequency aggregation
	if productGroups[tx.ProductName] == nil {
		productGroups[tx.ProductName] = &models.ProductFrequency{
			ProductName:   tx.ProductName,
			Category:      tx.Category,
			StockQuantity: tx.Stock,
		}
	}
	productGroups[tx.ProductName].Frequency++

	// Monthly sales aggregation
	month := tx.Date.Format("2006-01")
	monthlyGroups[month] += tx.TotalPrice

	// Region revenue aggregation
	if regionGroups[tx.Region] == nil {
		regionGroups[tx.Region] = &models.RegionRevenue{Region: tx.Region}
	}
	regionGroups[tx.Region].Revenue += tx.TotalPrice
	regionGroups[tx.Region].ItemsSold += tx.Quantity
}

func (a *Analytics) mergeResults(local, global map[string]*models.CountryRevenue) {
	for k, v := range local {
		if global[k] == nil {
			global[k] = &models.CountryRevenue{
				Country:     v.Country,
				ProductName: v.ProductName,
				Category:    v.Category,
			}
		}
		global[k].TotalRevenue += v.TotalRevenue
		global[k].Transactions += v.Transactions
	}
}

func (a *Analytics) mergeProductResults(local, global map[string]*models.ProductFrequency) {
	for k, v := range local {
		if global[k] == nil {
			global[k] = &models.ProductFrequency{
				ProductName:   v.ProductName,
				Category:      v.Category,
				StockQuantity: v.StockQuantity,
			}
		}
		global[k].Frequency += v.Frequency
	}
}

func (a *Analytics) mergeMonthlyResults(local, global map[string]float64) {
	for k, v := range local {
		global[k] += v
	}
}

func (a *Analytics) mergeRegionResults(local, global map[string]*models.RegionRevenue) {
	for k, v := range local {
		if global[k] == nil {
			global[k] = &models.RegionRevenue{Region: v.Region}
		}
		global[k].Revenue += v.Revenue
		global[k].ItemsSold += v.ItemsSold
	}
}

func (a *Analytics) computeAnalytics(data []models.Transaction) *PrecomputedData {
	countryGroups := make(map[string]*models.CountryRevenue)
	productGroups := make(map[string]*models.ProductFrequency)
	monthlyGroups := make(map[string]float64)
	regionGroups := make(map[string]*models.RegionRevenue)

	for _, tx := range data {
		a.aggregateTransaction(tx, countryGroups, productGroups, monthlyGroups, regionGroups)
	}

	return &PrecomputedData{
		CountryRevenue: a.sortCountryRevenue(countryGroups),
		TopProducts:    a.sortTopProducts(productGroups),
		MonthlySales:   a.sortMonthlySales(monthlyGroups),
		TopRegions:     a.sortTopRegions(regionGroups),
		LastModified:   time.Now(),
		RecordCount:    int64(len(data)),
	}
}

func (a *Analytics) sortCountryRevenue(groups map[string]*models.CountryRevenue) []models.CountryRevenue {
	result := make([]models.CountryRevenue, 0, len(groups))
	for _, cr := range groups {
		result = append(result, *cr)
	}
	slices.SortFunc(result, func(a, b models.CountryRevenue) int {
		if a.TotalRevenue > b.TotalRevenue {
			return -1
		}
		if a.TotalRevenue < b.TotalRevenue {
			return 1
		}
		return 0
	})
	return result
}

func (a *Analytics) sortTopProducts(groups map[string]*models.ProductFrequency) []models.ProductFrequency {
	result := make([]models.ProductFrequency, 0, len(groups))
	for _, pf := range groups {
		result = append(result, *pf)
	}
	slices.SortFunc(result, func(a, b models.ProductFrequency) int {
		if a.Frequency > b.Frequency {
			return -1
		}
		if a.Frequency < b.Frequency {
			return 1
		}
		return 0
	})
	return result
}

func (a *Analytics) sortMonthlySales(groups map[string]float64) []models.MonthlyData {
	result := make([]models.MonthlyData, 0, len(groups))
	for month, volume := range groups {
		result = append(result, models.MonthlyData{Month: month, Volume: volume})
	}
	slices.SortFunc(result, func(a, b models.MonthlyData) int {
		if a.Volume > b.Volume {
			return -1
		}
		if a.Volume < b.Volume {
			return 1
		}
		return 0
	})
	return result
}

func (a *Analytics) sortTopRegions(groups map[string]*models.RegionRevenue) []models.RegionRevenue {
	result := make([]models.RegionRevenue, 0, len(groups))
	for _, rr := range groups {
		result = append(result, *rr)
	}
	slices.SortFunc(result, func(a, b models.RegionRevenue) int {
		if a.Revenue > b.Revenue {
			return -1
		}
		if a.Revenue < b.Revenue {
			return 1
		}
		return 0
	})
	return result
}

// Cache management
func (a *Analytics) getCacheFilename(csvPath string) string {
	return fmt.Sprintf("%s/%s_%s.gob", cacheDir, strings.ReplaceAll(csvPath, "/", "_"), cacheVersion)
}

func (a *Analytics) saveToCache(csvPath string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	filename := a.getCacheFilename(csvPath)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	a.mu.RLock()
	defer a.mu.RUnlock()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(a.precomputed)
}

func (a *Analytics) loadFromCache(csvPath string) (*PrecomputedData, error) {
	filename := a.getCacheFilename(csvPath)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data PrecomputedData
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// Fast query methods - O(1) lookups from precomputed data
func (a *Analytics) CountryRevenue() []models.CountryRevenue {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.precomputed.CountryRevenue
}

func (a *Analytics) TopProducts(limit int) []models.ProductFrequency {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.precomputed.TopProducts) <= limit {
		return a.precomputed.TopProducts
	}
	return a.precomputed.TopProducts[:limit]
}

func (a *Analytics) MonthlySales() []models.MonthlyData {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.precomputed.MonthlySales
}

func (a *Analytics) TopRegions(limit int) []models.RegionRevenue {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.precomputed.TopRegions) <= limit {
		return a.precomputed.TopRegions
	}
	return a.precomputed.TopRegions[:limit]
}

// Utility method for monitoring
func (a *Analytics) Stats() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]any{
		"record_count":   a.precomputed.RecordCount,
		"last_processed": a.precomputed.LastModified,
		"countries":      len(a.precomputed.CountryRevenue),
		"products":       len(a.precomputed.TopProducts),
		"months":         len(a.precomputed.MonthlySales),
		"regions":        len(a.precomputed.TopRegions),
	}
}
