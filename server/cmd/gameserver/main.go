package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/gateway"
)

func main() {
	addr := os.Getenv("MIS_AUTHORITY_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8787"
	}

	app, err := gateway.NewAuthorityHTTPServer()
	if err != nil {
		log.Fatalf("create authority server: %v", err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: app.Handler(),
	}

	log.Printf("M1 authority server listening on http://%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("authority server stopped: %v", err)
	}
}
