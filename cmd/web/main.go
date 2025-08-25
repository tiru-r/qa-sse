package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"abt-dashboard/internal/config"
	"abt-dashboard/internal/middleware"
	"abt-dashboard/internal/observability"
	"abt-dashboard/internal/server"
	"abt-dashboard/internal/services"
	"abt-dashboard/internal/ui/templates"
)

const (
	renderTimeout  = 10 * time.Second
	csvLoadTimeout = 30 * time.Second
	cacheMaxAge    = "public, max-age=300"
)

// Template handler functions that can access the template functions
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), renderTimeout)
	defer cancel()

	w.Header().Set("Cache-Control", cacheMaxAge)
	if err := templates.Dashboard().Render(ctx, w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Logger)
	slog.SetDefault(logger)

	logger.Info("starting application",
		"version", "1.0.0",
		"config", cfg,
	)

	analytics := services.NewAnalytics()
	ctx, cancel := context.WithTimeout(context.Background(), csvLoadTimeout)
	defer cancel()

	start := time.Now()
	if err := analytics.LoadFromCSV(ctx, cfg.Database.CSVFile); err != nil {
		logger.Error("failed to load CSV data", "error", err)
		os.Exit(1)
	}
	duration := time.Since(start)
	logger.Info("CSV data loaded successfully", "duration", duration)

	templateHandlers := &server.TemplateHandlers{
		Dashboard: handleDashboard,
	}

	srv := server.NewServer(analytics, logger, templateHandlers)

	rateLimiter := middleware.NewRateLimiter(cfg.Security)

	middlewareChain := middleware.Chain(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Tracing(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.Security),
		middleware.TrustedProxy(cfg.Security),
		middleware.RateLimit(rateLimiter, logger),
	)

	handler := middlewareChain(srv)

	httpServer := &http.Server{
		Addr:         cfg.Address(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	gracefulServer := server.NewGracefulServer(httpServer, logger, cfg)

	gracefulServer.RegisterShutdownHook(func(ctx context.Context) error {
		logger.Info("shutting down analytics service")
		return nil
	})

	logger.Info("starting graceful server")
	if err := gracefulServer.ListenAndServe(); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("application stopped gracefully")
}
