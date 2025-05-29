package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"rubxy/auth"
	"rubxy/config"
	"rubxy/db"
	"rubxy/logger"
	"rubxy/middleware"
	"rubxy/proxy"
)

func main() {
	cfg := config.Load()
	logger.Init("rubxy.log")
	defer logger.LogFile.Close()
	logger.InfoLogger.Println("Starting server...")

	db.Init(cfg.DatabaseURL)
	defer db.DB.Close()

	r := chi.NewRouter()

	// Public routes
	r.Post("/get-token", auth.HandleToken(cfg))
	r.Post("/refresh-token", auth.HandleRefresh(cfg))
	r.Post("/register", auth.HandleRegister())
	r.Post("/logout", auth.HandleLogout())
	// Protected admin routes
	r.With(middleware.Authenticate(cfg)).Post("/admin/activity/add", proxy.HandleAdminActivityAdd)
	r.With(middleware.Authenticate(cfg)).Post("/admin/reward/transfer", proxy.HandleAdminRewardTransfer)
	r.With(middleware.Authenticate(cfg)).Get("/admin/activity/list", proxy.HandleGetAllActivities)

	// Protected routes
	target := "http://localhost:20000"
	proxyHandler := proxy.NewReverseProxy(target)
	//r.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)
	r.Route("/api", func(api chi.Router) {
		api.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)
	})

	log.Printf("Server running at %s\n", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
