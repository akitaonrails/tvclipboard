// TV Clipboard - Peer-to-peer clipboard sharing
// Copyright (C) 2026 Fabio Akita
// Licensed under GNU Affero General Public License v3.0

package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tvclipboard/i18n"
	"tvclipboard/pkg/config"
	"tvclipboard/pkg/hub"
	"tvclipboard/pkg/qrcode"
	"tvclipboard/pkg/server"
	"tvclipboard/pkg/token"
)

//go:embed static
var staticFiles embed.FS

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize i18n
	i18nInstance := i18n.GetInstance()
	if err := i18nInstance.SetLanguage(cfg.Language); err != nil {
		log.Printf("Failed to set language %s, falling back to en: %v", cfg.Language, err)
	}
	if err := i18nInstance.LoadAllLanguages(); err != nil {
		log.Printf("Warning: failed to load translation files: %v", err)
	}

	// Initialize components
	h := hub.NewHub(cfg.MaxMessageSize, cfg.RateLimitPerSec)
	go h.Run()

	tokenManager := token.NewTokenManager(
		cfg.PrivateKeyHex,
		int(cfg.SessionTimeout.Minutes()),
	)

	// Determine host:port for QR code
	// If GetQRHost already includes a port (from PublicURL), use it as-is
	// Otherwise, append the listening port (for LocalIP case)
	qrHost := cfg.GetQRHost()
	if cfg.PublicURL == "" {
		// Only add port when using LocalIP (no PublicURL set)
		qrHost += ":" + cfg.Port
	}

	qrGen := qrcode.NewGenerator(
		qrHost,
		cfg.GetQRScheme(),
		cfg.SessionTimeout,
	)

	srv := server.NewServer(h, tokenManager, qrGen, staticFiles, cfg.AllowedOrigins, i18nInstance)
	srv.RegisterRoutes()

	// Log startup information
	cfg.LogStartup()

	// Start server with graceful shutdown
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("Server listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down server...")
	h.Stop()
	srv.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
