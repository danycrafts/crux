package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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
		log.Fatalf("load config: %v", err)
	}
	if err := cfg.EnsureDirs(); err != nil {
		log.Fatalf("ensure dirs: %v", err)
	}

	// PID file
	pidPath := config.PIDPath()
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o644)
	defer os.Remove(pidPath)

	st, err := store.New(config.DBPath())
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	addr := fmt.Sprintf(":%d", cfg.APIPort)
	if p := os.Getenv("CRUX_API_PORT"); p != "" {
		addr = ":" + p
	}

	srv := api.NewServer(cfg, st)
	go func() {
		log.Printf("cruxd listening on %s", addr)
		if err := srv.Start(addr); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}
