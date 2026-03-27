package client

import (
	"context"
	"fmt"

	"github.com/skiphead/salutespeech/types"
)

// Client is the main client interface for business operations
type Client interface {
	GetToken(ctx context.Context) (string, error)
	Close() error
}

// SaluteSpeechClient implements Client interface
type SaluteSpeechClient struct {
	oauthClient *OAuthClient
	tokenMgr    *TokenManager
}

// NewSaluteSpeechClient creates new SaluteSpeech client
func NewSaluteSpeechClient(cfg Config) (*SaluteSpeechClient, error) {
	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	oauthClient, err := NewOAuthClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	tokenMgr := NewTokenManager(oauthClient, TokenManagerConfig{
		RefreshMargin:      types.DefaultRefreshMargin,
		MinRefreshInterval: types.DefaultMinRefreshInt,
		Logger:             cfg.Logger,
	})

	return &SaluteSpeechClient{
		oauthClient: oauthClient,
		tokenMgr:    tokenMgr,
	}, nil
}

// GetToken returns a valid token for business operations
func (c *SaluteSpeechClient) GetToken(ctx context.Context) (string, error) {
	return c.tokenMgr.GetToken(ctx)
}

// Close cleans up resources
func (c *SaluteSpeechClient) Close() error {
	var errs []error

	if c.tokenMgr != nil {
		if err := c.tokenMgr.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close token manager: %w", err))
		}
	}

	if c.oauthClient != nil {
		if err := c.oauthClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close OAuth client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// validateConfig validates the client configuration
func validateConfig(cfg Config) error {
	if cfg.AuthKey == "" {
		return types.ErrAuthKeyRequired
	}
	if cfg.Scope == "" {
		return types.ErrScopeRequired
	}
	// Logger is optional - will use slog.Default() if nil
	return nil
}
