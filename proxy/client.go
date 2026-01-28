package proxy

import (
	"net/http"
	"time"
)

// SharedHTTPClient is a shared HTTP client with proper connection pooling
// to prevent connection exhaustion under high load
var SharedHTTPClient = &http.Client{
	Timeout: 5 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:        100,              // Maximum number of idle connections
		MaxIdleConnsPerHost: 25,               // Maximum idle connections per host
		MaxConnsPerHost:     50,               // Maximum total connections per host
		IdleConnTimeout:     90 * time.Second, // How long idle connections are kept
		DisableKeepAlives:   false,            // Enable connection reuse
		ForceAttemptHTTP2:   true,              // Enable HTTP/2 for better performance
	},
}

