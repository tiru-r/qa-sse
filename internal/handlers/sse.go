package handlers

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"abt-dashboard/internal/models"
	"abt-dashboard/internal/services"
	"github.com/starfederation/datastar-go/datastar"
)

const (
	maxTableRows = 50
	maxProducts  = 20
	maxRegions   = 30
)

var countryTableTemplate = template.Must(template.New("countryTable").Parse(`
<div id="country-content">
<table class="modern-table">
<thead><tr><th>Country</th><th>Product</th><th>Category</th><th>Revenue</th><th>Orders</th></tr></thead>
<tbody>
{{range $i, $item := .Data}}{{if lt $i $.MaxRows}}<tr>
<td>{{.Country}}</td>
<td>{{.ProductName}}</td>
<td><span class="category-badge">{{.Category}}</span></td>
<td><strong>${{printf "%.2f" .TotalRevenue}}</strong></td>
<td>{{.Transactions}}</td>
</tr>{{end}}{{end}}
</tbody>
</table>
</div>`))

type SSEHandlers struct {
	analytics *services.Analytics
	logger    *slog.Logger
}

func NewSSEHandlers(analytics *services.Analytics, logger *slog.Logger) *SSEHandlers {
	return &SSEHandlers{
		analytics: analytics,
		logger:    logger,
	}
}

type templateData struct {
	Data    interface{}
	MaxRows int
}

func (h *SSEHandlers) renderCountryTable(data interface{}) (string, error) {
	var buf strings.Builder

	// Limit data slice to avoid processing unnecessary records
	var limitedData interface{}
	if slice, ok := data.([]models.CountryRevenue); ok && len(slice) > maxTableRows {
		limitedData = slice[:maxTableRows]
	} else {
		limitedData = data
	}

	tmplData := templateData{Data: limitedData, MaxRows: maxTableRows}
	err := countryTableTemplate.Execute(&buf, tmplData)
	return buf.String(), err
}

func (h *SSEHandlers) HandleCountryRevenue(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	data := h.analytics.CountryRevenue()
	html, err := h.renderCountryTable(data)
	if err != nil {
		h.logger.Error("render country table", "error", err)
		return
	}

	sse.PatchElements(html)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *SSEHandlers) HandleTopProducts(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	data := h.analytics.TopProducts(maxProducts)
	jsonData, err := json.Marshal(map[string]any{
		"productsData": data,
	})
	if err != nil {
		h.logger.Error("marshal products data", "error", err)
		return
	}
	sse.PatchSignals(jsonData)

	sse.PatchElements(`<div id="products-content">✅ Products chart data loaded</div>`)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *SSEHandlers) HandleMonthlySales(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	data := h.analytics.MonthlySales()
	jsonData, err := json.Marshal(map[string]any{
		"monthlyData": data,
	})
	if err != nil {
		h.logger.Error("marshal monthly data", "error", err)
		return
	}
	sse.PatchSignals(jsonData)

	sse.PatchElements(`<div id="monthly-content">✅ Monthly sales chart data loaded</div>`)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *SSEHandlers) HandleTopRegions(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	data := h.analytics.TopRegions(maxRegions)
	jsonData, err := json.Marshal(map[string]any{
		"regionsData": data,
	})
	if err != nil {
		h.logger.Error("marshal regions data", "error", err)
		return
	}
	sse.PatchSignals(jsonData)
	sse.PatchElements(`<div id="regions-content">✅ Regions chart data loaded</div>`)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *SSEHandlers) HandleRefreshAll(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	// Get fresh data for country revenue
	countryData := h.analytics.CountryRevenue()
	html, err := h.renderCountryTable(countryData)
	if err != nil {
		h.logger.Error("render country table", "error", err)
		return
	}
	sse.PatchElements(html)

	// Get fresh data for products, monthly sales, and regions
	productsData := h.analytics.TopProducts(maxProducts)
	monthlyData := h.analytics.MonthlySales()
	regionsData := h.analytics.TopRegions(maxRegions)

	// Send all signals in one call
	allSignals, err := json.Marshal(map[string]any{
		"productsData": productsData,
		"monthlyData":  monthlyData,
		"regionsData":  regionsData,
	})
	if err != nil {
		h.logger.Error("marshal all signals data", "error", err)
		return
	}
	sse.PatchSignals(allSignals)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
