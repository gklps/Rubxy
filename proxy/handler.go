package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
	"rubxy/logger"
	"rubxy/middleware"
)

func NewReverseProxy(target string) http.Handler {
	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Configure the reverse proxy's transport with proper connection pooling
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,              // Maximum number of idle connections
		MaxIdleConnsPerHost: 25,               // Maximum idle connections per host
		MaxConnsPerHost:     50,               // Maximum total connections per host
		IdleConnTimeout:     90 * time.Second, // How long idle connections are kept
		DisableKeepAlives:   false,            // Enable connection reuse
		ForceAttemptHTTP2:   true,              // Enable HTTP/2 for better performance
	}

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
