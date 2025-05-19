package main

import (
	"log"
	"net/http"
	"rubxy/auth"
	"rubxy/config"
	"rubxy/middleware"
	"rubxy/proxy"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.Load()
	r := chi.NewRouter()

	// Public routes: token get and refresh
	r.Post("/get-token", auth.HandleToken(cfg))
	r.Post("/refresh-token", auth.HandleRefresh(cfg))

	// Proxy to rubixgoplatform running at localhost:20000
	target := "http://localhost:20000"
	proxyHandler := proxy.NewReverseProxy(target)

	// Protected proxy routes with auth middleware
	r.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)

	log.Printf("Starting server on %s\n", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
