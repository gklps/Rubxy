package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"rubxy/logger"
	"rubxy/middleware"
)

func NewReverseProxy(target string) http.Handler {
	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		username := middleware.GetUserFromContext(req)
		logger.InfoLogger.Printf("[REQUEST] User: %s Method: %s Path: %s", username, req.Method, req.URL.Path)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		username := middleware.GetUserFromContext(resp.Request)
		logger.InfoLogger.Printf("[RESPONSE] User: %s Status: %d URL: %s", username, resp.StatusCode, resp.Request.URL)
		return nil
	}

	return proxy
}
