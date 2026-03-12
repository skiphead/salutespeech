package utils

import (
	"crypto/tls"
	"net/http"
	"time"
)

// DefaultHTTPClient returns default HTTP client with timeout
func DefaultHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// TLSHTTPClient returns HTTP client with custom TLS config
func TLSHTTPClient(timeout time.Duration, allowInsecure bool) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: allowInsecure,
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// IsSuccessStatus checks if HTTP status code is success
func IsSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

// IsRetryableStatus checks if status code indicates retryable error
func IsRetryableStatus(code int) bool {
	return code == 429 || code == 500 || code == 502 || code == 503 || code == 504
}
