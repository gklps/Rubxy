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

	// Public routes
	r.Post("/auth/token", auth.HandleToken(cfg))
	r.Post("/auth/refresh", auth.HandleRefresh(cfg))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(cfg))
		r.Handle("/*", proxy.NewReverseProxy("http://localhost:20000"))
	})

	log.Printf("Proxy server started on %s", cfg.Port)
	http.ListenAndServe(cfg.Port, r)
}
