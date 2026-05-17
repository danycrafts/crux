package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/danycrafts/crux/pkg/logger"
	"github.com/danycrafts/crux/services/daemon/internal/api"
	"github.com/danycrafts/crux/services/daemon/internal/config"
	"github.com/danycrafts/crux/services/daemon/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("cruxd 0.1.0")
		return
	}

	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}
	if err := cfg.EnsureDirs(); err != nil {
		logger.Fatalf("ensure dirs: %v", err)
	}

	// Setup logging
	logCfg := cfg.Logging
	if logCfg.File == "" {
		logCfg.File = filepath.Join(cfg.DataDir, "logs", "cruxd.log")
	}
	logger.Init(logCfg)
	logger.SetDefault()
	logger.Info("cruxd starting", "version", "0.1.0", "data_dir", cfg.DataDir)

	// PID file
	pidPath := config.PIDPath()
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o644)
	defer os.Remove(pidPath)

	st, err := store.New(config.DBPath())
	if err != nil {
		logger.Fatalf("open store: %v", err)
	}
	defer st.Close()
	logger.Info("store opened", "path", config.DBPath())

	addr := fmt.Sprintf(":%d", cfg.APIPort)
	if p := os.Getenv("CRUX_API_PORT"); p != "" {
		addr = ":" + p
	}

	srv := api.NewServer(cfg, st)
	go func() {
		logger.Info("api listening", "addr", addr)
		if err := srv.Start(addr); err != nil {
			logger.Error("server error", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
	logger.Info("shutdown complete")
}
