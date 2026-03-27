package client

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/skiphead/salutespeech/types"
	"golang.org/x/sync/singleflight"
)

// TokenManager manages token lifecycle
type TokenManager struct {
	mu                 sync.RWMutex
	client             *OAuthClient
	currentToken       *types.Token
	refreshMargin      time.Duration
	minRefreshInterval time.Duration
	group              singleflight.Group
	lastRefreshTime    time.Time
	logger             *slog.Logger
	closeCh            chan struct{}
	closeOnce          sync.Once
}

// TokenManagerConfig represents token manager configuration
type TokenManagerConfig struct {
	RefreshMargin      time.Duration
	MinRefreshInterval time.Duration
	Logger             *slog.Logger
}

// NewTokenManager creates new token manager
func NewTokenManager(client *OAuthClient, cfg TokenManagerConfig) *TokenManager {
	if cfg.RefreshMargin == 0 {
		cfg.RefreshMargin = types.DefaultRefreshMargin
	}
	if cfg.MinRefreshInterval == 0 {
		cfg.MinRefreshInterval = types.DefaultMinRefreshInt
	}

	// Unified logger handling: fallback to default if nil
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &TokenManager{
		client:             client,
		refreshMargin:      cfg.RefreshMargin,
		minRefreshInterval: cfg.MinRefreshInterval,
		logger:             logger,
		closeCh:            make(chan struct{}),
	}
}

// GetToken returns valid token
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	// Check if token manager is closed
	select {
	case <-tm.closeCh:
		return "", fmt.Errorf("token manager is closed")
	default:
	}

	tm.mu.RLock()
	if tm.currentToken != nil && tm.currentToken.IsValid(tm.refreshMargin) {
		token := tm.currentToken.Value
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	result, err, _ := tm.group.Do("refresh", func() (interface{}, error) {
		return tm.refreshInternal(ctx)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

func (tm *TokenManager) refreshInternal(ctx context.Context) (string, error) {
	// Check if token manager is closed
	select {
	case <-tm.closeCh:
		return "", fmt.Errorf("token manager is closed")
	default:
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.currentToken != nil && tm.currentToken.IsValid(tm.refreshMargin) {
		return tm.currentToken.Value, nil
	}

	if time.Since(tm.lastRefreshTime) < tm.minRefreshInterval {
		if tm.currentToken != nil && tm.currentToken.Value != "" {
			tm.logger.Debug("rate limit: using cached token")
			return tm.currentToken.Value, nil
		}

		wait := tm.minRefreshInterval - time.Since(tm.lastRefreshTime)
		tm.logger.Debug("rate limit: waiting",
			slog.Duration("wait", wait.Round(time.Second)),
		)

		// Use time.NewTimer instead of time.After for better resource cleanup
		timer := time.NewTimer(wait)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
			return "", ctx.Err()
		case <-tm.closeCh:
			return "", fmt.Errorf("token manager closed during wait")
		}
	}

	newToken, err := tm.client.RequestToken(ctx)
	if err != nil {
		if isRateLimitError(err) && tm.currentToken != nil {
			tm.logger.Debug("429 received, using fallback token")
			return tm.currentToken.Value, nil
		}
		return "", fmt.Errorf("refresh token: %w", err)
	}

	tm.currentToken = newToken
	tm.lastRefreshTime = time.Now()
	tm.logger.Debug("new token acquired",
		slog.Duration("ttl", time.Until(newToken.ExpiresAt).Round(time.Second)),
	)
	return newToken.Value, nil
}

// GetTokenWithHeader returns Authorization header value
// Note: token value is already validated and sanitized by the OAuth server,
// so we don't need to trim spaces. We keep the token as-is.
func (tm *TokenManager) GetTokenWithHeader(ctx context.Context) (string, error) {
	token, err := tm.GetToken(ctx)
	if err != nil {
		return "", err
	}
	return "Bearer " + token, nil // No TrimSpace - token is already clean
}

// ForceRefresh forces token refresh
func (tm *TokenManager) ForceRefresh(ctx context.Context) error {
	// Check if token manager is closed
	select {
	case <-tm.closeCh:
		return fmt.Errorf("token manager is closed")
	default:
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	newToken, err := tm.client.RequestToken(ctx)
	if err != nil {
		return err
	}
	tm.currentToken = newToken
	tm.lastRefreshTime = time.Now()
	return nil
}

// GetTokenInfo returns token information
func (tm *TokenManager) GetTokenInfo() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	info := map[string]interface{}{"has_token": tm.currentToken != nil}
	if tm.currentToken != nil {
		info["ttl_seconds"] = int(time.Until(tm.currentToken.ExpiresAt).Seconds())
		info["is_valid"] = tm.currentToken.IsValid(tm.refreshMargin)
	}
	if !tm.lastRefreshTime.IsZero() {
		info["last_refresh"] = tm.lastRefreshTime.Format(time.RFC3339)
	}
	return info
}

// Close cleans up token manager resources
func (tm *TokenManager) Close() error {
	tm.closeOnce.Do(func() {
		close(tm.closeCh)

		// Cancel any ongoing operations
		tm.mu.Lock()
		defer tm.mu.Unlock()

		// Clear sensitive data
		if tm.currentToken != nil {
			tm.currentToken.Value = "" // Clear token value
			tm.currentToken = nil
		}

		tm.logger.Debug("token manager closed")
	})
	return nil // Always return nil as there's no error case
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "429") || strings.Contains(s, "Too Many Requests")
}
