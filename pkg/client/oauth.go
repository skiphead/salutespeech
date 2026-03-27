package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skiphead/salutespeech/types"
)

// OAuthClient handles OAuth authentication
type OAuthClient struct {
	httpClient *http.Client
	oauthURL   string
	authHeader string
	scope      types.Scope
	logger     *slog.Logger
	closeCh    chan struct{}
	closeOnce  sync.Once
}

// Config represents OAuth client configuration
type Config struct {
	AuthKey       string
	Scope         types.Scope
	OAuthURL      string
	Timeout       time.Duration
	AllowInsecure bool
	Logger        *slog.Logger
}

// NewOAuthClient creates new OAuth client
func NewOAuthClient(cfg Config) (*OAuthClient, error) {
	// Validate required fields
	if cfg.AuthKey == "" {
		return nil, fmt.Errorf("%w: auth key is required", types.ErrAuthKeyRequired)
	}
	if cfg.Scope == "" {
		return nil, fmt.Errorf("%w: scope is required", types.ErrScopeRequired)
	}

	// Unified logger handling: fallback to default if nil
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
		logger.Warn("logger not provided, using slog.Default()")
	}

	oauthURL, err := sanitizeURL(cfg.OAuthURL)
	if err != nil {
		logger.Warn("invalid OAuth URL, using default", slog.String("error", err.Error()))
		oauthURL = types.DefaultOAuthURL
	}
	if oauthURL == "" {
		oauthURL = types.DefaultOAuthURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultTimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}

	if cfg.AllowInsecure {
		logger.Warn("TLS verification disabled")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	authHeader := cfg.AuthKey
	if !strings.HasPrefix(strings.ToLower(cfg.AuthKey), "basic ") {
		authHeader = "Basic " + cfg.AuthKey
	}

	return &OAuthClient{
		httpClient: httpClient,
		oauthURL:   oauthURL,
		authHeader: authHeader,
		scope:      cfg.Scope,
		logger:     logger,
		closeCh:    make(chan struct{}),
	}, nil
}

// RequestToken requests new token from OAuth server
func (c *OAuthClient) RequestToken(ctx context.Context) (*types.Token, error) {
	// Check if client is closed
	select {
	case <-c.closeCh:
		return nil, fmt.Errorf("oauth client is closed")
	default:
	}

	rqUID := uuid.New().String()

	form := url.Values{}
	form.Add("scope", string(c.scope))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.oauthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", rqUID)
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		select {
		case <-c.closeCh:
			return nil, fmt.Errorf("oauth client closed during request")
		default:
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token error %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp types.TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	expiresIn := time.Duration(tokenResp.ExpiresIn) * time.Second
	if expiresIn == 0 {
		expiresIn = 30 * time.Minute
	}

	return &types.Token{
		Value:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

// Close cleans up OAuth client resources
func (c *OAuthClient) Close() error {
	c.closeOnce.Do(func() {
		close(c.closeCh)

		// Close idle connections
		if c.httpClient != nil && c.httpClient.Transport != nil {
			if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
		}

		c.logger.Debug("oauth client closed")
	})
	return nil // Always return nil as there's no error case
}

// GenerateBasicAuthKey generates Basic Auth key from client ID and secret
func GenerateBasicAuthKey(clientID, clientSecret string) string {
	cred := fmt.Sprintf("%s:%s", clientID, clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(cred))
}

// sanitizeURL validates and sanitizes URL
func sanitizeURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", nil
	}

	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme: %s (must be http or https)", parsedURL.Scheme)
	}

	// Validate host
	if parsedURL.Host == "" {
		return "", fmt.Errorf("missing host in URL")
	}

	// Return normalized URL
	return parsedURL.String(), nil
}
