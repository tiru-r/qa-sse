package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"abt-dashboard/internal/config"
)

type GracefulServer struct {
	server     *http.Server
	logger     *slog.Logger
	config     *config.Config
	shutdownFn []func(ctx context.Context) error
	mu         sync.RWMutex
}

func NewGracefulServer(server *http.Server, logger *slog.Logger, config *config.Config) *GracefulServer {
	return &GracefulServer{
		server:     server,
		logger:     logger,
		config:     config,
		shutdownFn: make([]func(ctx context.Context) error, 0),
	}
}

func (gs *GracefulServer) RegisterShutdownHook(fn func(ctx context.Context) error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.shutdownFn = append(gs.shutdownFn, fn)
}

func (gs *GracefulServer) ListenAndServe() error {
	serverErrors := make(chan error, 1)

	go func() {
		gs.logger.Info("starting server",
			"addr", gs.server.Addr,
			"read_timeout", gs.config.Server.ReadTimeout,
			"write_timeout", gs.config.Server.WriteTimeout,
		)
		serverErrors <- gs.server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil

	case sig := <-shutdown:
		gs.logger.Info("shutdown signal received", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), gs.config.Server.ShutdownTimeout)
		defer cancel()

		return gs.shutdown(ctx)
	}
}

func (gs *GracefulServer) shutdown(ctx context.Context) error {
	gs.logger.Info("starting graceful shutdown",
		"timeout", gs.config.Server.ShutdownTimeout,
	)

	gs.mu.RLock()
	hooks := make([]func(ctx context.Context) error, len(gs.shutdownFn))
	copy(hooks, gs.shutdownFn)
	gs.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks)+1)

	for i, hook := range hooks {
		wg.Add(1)
		go func(idx int, fn func(ctx context.Context) error) {
			defer wg.Done()

			hookCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			gs.logger.Debug("executing shutdown hook", "hook_index", idx)
			if err := fn(hookCtx); err != nil {
				gs.logger.Error("shutdown hook failed",
					"hook_index", idx,
					"error", err,
				)
				errChan <- fmt.Errorf("shutdown hook %d failed: %w", idx, err)
			} else {
				gs.logger.Debug("shutdown hook completed", "hook_index", idx)
			}
		}(i, hook)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.logger.Info("stopping HTTP server")
		if err := gs.server.Shutdown(ctx); err != nil {
			gs.logger.Error("HTTP server shutdown failed", "error", err)
			errChan <- fmt.Errorf("HTTP server shutdown failed: %w", err)
		} else {
			gs.logger.Info("HTTP server stopped gracefully")
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		gs.logger.Info("graceful shutdown completed")

		select {
		case err := <-errChan:
			return err
		default:
			return nil
		}

	case <-ctx.Done():
		gs.logger.Warn("shutdown timeout exceeded, forcing exit")
		return ctx.Err()
	}
}
