package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"abt-dashboard/internal/errors"
	"abt-dashboard/internal/services"
)

type APIHandlers struct {
	analytics *services.Analytics
	logger    *slog.Logger
}

func NewAPIHandlers(analytics *services.Analytics, logger *slog.Logger) *APIHandlers {
	return &APIHandlers{
		analytics: analytics,
		logger:    logger,
	}
}

func (h *APIHandlers) HandleCountryRevenue(w http.ResponseWriter, r *http.Request) {

	data := h.analytics.CountryRevenue()

	headers := map[string]string{
		"Cache-Control": "public, max-age=300",
	}

	errors.WriteSuccessWithHeaders(w, data, headers)
}

func (h *APIHandlers) HandleTopProducts(w http.ResponseWriter, r *http.Request) {

	data := h.analytics.TopProducts(20)

	headers := map[string]string{
		"Cache-Control": "public, max-age=300",
	}

	errors.WriteSuccessWithHeaders(w, data, headers)
}

func (h *APIHandlers) HandleMonthlySales(w http.ResponseWriter, r *http.Request) {

	data := h.analytics.MonthlySales()

	headers := map[string]string{
		"Cache-Control": "public, max-age=300",
	}

	errors.WriteSuccessWithHeaders(w, data, headers)
}

func (h *APIHandlers) HandleTopRegions(w http.ResponseWriter, r *http.Request) {

	data := h.analytics.TopRegions(30)

	headers := map[string]string{
		"Cache-Control": "public, max-age=300",
	}

	errors.WriteSuccessWithHeaders(w, data, headers)
}

func (h *APIHandlers) HandleHealth(w http.ResponseWriter, r *http.Request) {

	healthData := map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	errors.WriteSuccess(w, healthData)
}

func (h *APIHandlers) HandleStats(w http.ResponseWriter, r *http.Request) {

	stats := h.analytics.Stats()

	errors.WriteSuccess(w, stats)
}
