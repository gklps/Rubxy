package main

import (
	"fmt"
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

	// Clean paths (trim trailing spaces) - apply globally
	r.Use(middleware.CleanPath)

	// Public routes
	r.Post("/get-token", auth.HandleToken(cfg))
	r.Post("/refresh-token", auth.HandleRefresh(cfg))
	r.Post("/register", auth.HandleRegister())
	r.Post("/logout", auth.HandleLogout())

	// Protected admin routes - register /admin/payouts directly first
	r.With(middleware.Authenticate(cfg)).Post("/admin/payouts", proxy.HandleAdminRewardTransfer)

	r.Route("/admin", func(admin chi.Router) {
		admin.Use(middleware.Authenticate(cfg))
		admin.Post("/activity/add", proxy.HandleAdminActivityAdd)
		admin.Get("/activity/list", proxy.HandleGetAllActivities)
		admin.Post("/user/add", proxy.HandleAdminAddUser)
	})

	// Protected user routes
	r.With(middleware.Authenticate(cfg)).Get("/users/{user_did}/payouts", proxy.HandleUserPayouts)

	// Protected routes
	target := "http://localhost:20050"
	proxyHandler := proxy.NewReverseProxy(target)
	//r.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)
	r.Route("/api", func(api chi.Router) {
		api.With(middleware.Authenticate(cfg)).Handle("/*", proxyHandler)
	})

	// 404 handler for debugging
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		logMsg := fmt.Sprintf("[404] Method: %s, Path: %s, RemoteAddr: %s, Headers: %v",
			r.Method, r.URL.Path, r.RemoteAddr, r.Header)
		logger.InfoLogger.Printf(logMsg)
		log.Printf(logMsg) // Also log to console
		http.Error(w, "404 page not found", http.StatusNotFound)
	})

	// Log registered routes
	logger.InfoLogger.Println("Registered routes:")
	logger.InfoLogger.Println("  POST /get-token")
	logger.InfoLogger.Println("  POST /refresh-token")
	logger.InfoLogger.Println("  POST /register")
	logger.InfoLogger.Println("  POST /logout")
	logger.InfoLogger.Println("  POST /admin/activity/add (protected)")
	logger.InfoLogger.Println("  POST /admin/payouts (protected)")
	logger.InfoLogger.Println("  GET  /admin/activity/list (protected)")
	logger.InfoLogger.Println("  POST /admin/user/add (protected)")
	logger.InfoLogger.Println("  GET  /users/{user_did}/payouts (protected)")
	logger.InfoLogger.Println("  *    /api/* (protected, proxied)")

	log.Println("Registered routes:")
	log.Println("  POST /admin/payouts (protected)")
	log.Printf("Server running at %s\n", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
