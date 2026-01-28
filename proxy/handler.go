package proxy

import (
	"net"
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

	// Configure transport with connection pooling to prevent 502 errors
	// ResponseHeaderTimeout set to 6 minutes to handle 3-5 minute API responses
	proxy.Transport = &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 6 * time.Minute, // Wait up to 6 minutes for response headers
		DisableKeepAlives:     false,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Removed verbose request logging - errors are logged in endpoints
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Log only errors (non-2xx status codes)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			username := middleware.GetUserFromContext(resp.Request)
			logger.ErrorLogger.Printf("[PROXY ERROR] User: %s Status: %d Path: %s", username, resp.StatusCode, resp.Request.URL.Path)
		}
		return nil
	}

	return proxy
}
