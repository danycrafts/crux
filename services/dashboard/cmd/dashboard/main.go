package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static
var static embed.FS

func main() {
	port := os.Getenv("CRUX_DASHBOARD_PORT")
	if port == "" {
		port = "3001"
	}
	apiURL := os.Getenv("CRUX_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	content, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"api_url":"%s"}`+"\n", apiURL)
	})
	mux.Handle("/", http.FileServer(http.FS(content)))

	addr := ":" + port
	log.Printf("Crux Dashboard listening on %s (proxying to %s)", addr, apiURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}
