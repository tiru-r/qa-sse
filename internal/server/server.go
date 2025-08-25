package server

import (
	"log/slog"
	"net/http"

	"abt-dashboard/internal/handlers"
	"abt-dashboard/internal/services"
)

type Server struct {
	analytics   *services.Analytics
	mux         *http.ServeMux
	logger      *slog.Logger
	apiHandlers *handlers.APIHandlers
	sseHandlers *handlers.SSEHandlers
}

type TemplateHandlers struct {
	Dashboard http.HandlerFunc
}

func NewServer(analytics *services.Analytics, logger *slog.Logger, templateHandlers *TemplateHandlers) *Server {
	s := &Server{
		analytics:   analytics,
		mux:         http.NewServeMux(),
		logger:      logger,
		apiHandlers: handlers.NewAPIHandlers(analytics, logger),
		sseHandlers: handlers.NewSSEHandlers(analytics, logger),
	}
	s.setupRoutes(templateHandlers)
	return s
}

func (s *Server) setupRoutes(templateHandlers *TemplateHandlers) {
	// Dashboard routes
	s.mux.HandleFunc("GET /", templateHandlers.Dashboard)
	s.mux.HandleFunc("GET /health", s.apiHandlers.HandleHealth)
	s.mux.HandleFunc("GET /admin/stats", s.apiHandlers.HandleStats)

	// REST API endpoints
	s.mux.HandleFunc("GET /api/country-revenue", s.apiHandlers.HandleCountryRevenue)
	s.mux.HandleFunc("GET /api/top-products", s.apiHandlers.HandleTopProducts)
	s.mux.HandleFunc("GET /api/monthly-sales", s.apiHandlers.HandleMonthlySales)
	s.mux.HandleFunc("GET /api/top-regions", s.apiHandlers.HandleTopRegions)

	// Datastar SSE endpoints
	s.mux.HandleFunc("GET /sse/country-revenue", s.sseHandlers.HandleCountryRevenue)
	s.mux.HandleFunc("GET /sse/top-products", s.sseHandlers.HandleTopProducts)
	s.mux.HandleFunc("GET /sse/monthly-sales", s.sseHandlers.HandleMonthlySales)
	s.mux.HandleFunc("GET /sse/top-regions", s.sseHandlers.HandleTopRegions)
	s.mux.HandleFunc("GET /sse/refresh-all", s.sseHandlers.HandleRefreshAll)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
