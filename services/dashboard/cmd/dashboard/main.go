package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/danycrafts/crux/pkg/logger"
	"github.com/danycrafts/crux/services/dashboard/internal/config"
)

//go:embed static
var static embed.FS

func main() {
	cfg, err := config.Load(config.Path())
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Override from env
	if p := os.Getenv("CRUX_DASHBOARD_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &cfg.Port)
	}
	if u := os.Getenv("CRUX_API_URL"); u != "" {
		cfg.APIURL = u
	}

	// Setup logging
	logCfg := cfg.Logging
	if logCfg.File == "" {
		home, _ := os.UserHomeDir()
		logCfg.File = home + "/.crux/logs/dashboard.log"
	}
	logger.Init(logCfg)
	logger.SetDefault()
	logger.Info("dashboard starting", "version", "0.1.0", "port", cfg.Port)

	content, err := fs.Sub(static, "static")
	if err != nil {
		logger.Fatal("embed static", "err", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"api_url":"%s","refresh_interval":%d}`+"\n", cfg.APIURL, cfg.RefreshInterval)
	})
	mux.Handle("/", http.FileServer(http.FS(content)))

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("dashboard listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatal("server error", "err", err)
	}
}
