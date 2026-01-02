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
	r.Route("/admin", func(admin chi.Router) {
		admin.Use(middleware.Authenticate(cfg))
		admin.Post("/activity/add", proxy.HandleAdminActivityAdd)
		admin.Post("/payouts", proxy.HandleAdminRewardTransfer)
		admin.Get("/activity/list", proxy.HandleGetAllActivities)
		admin.Post("/user/add", proxy.HandleAdminAddUser)
	})

	// Protected user routes
	r.With(middleware.Authenticate(cfg)).Get("/users/{user_did}/payouts", proxy.HandleUserPayouts)

	// Protected routes
	target := "http://localhost:20000"
	proxyHandler := proxy.NewReverseProxy(target)
	//r.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)
	r.Route("/api", func(api chi.Router) {
		api.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)
	})

	// 404 handler for debugging
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoLogger.Printf("[404] Method: %s, Path: %s, RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
		http.Error(w, "404 page not found", http.StatusNotFound)
	})

	log.Printf("Server running at %s\n", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
