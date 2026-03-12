package client

import (
	"github.com/skiphead/salutespeech/types"
)

// Client is the main client interface
type Client interface {
	GetTokenManager() *TokenManager
}

// SaluteSpeechClient implements Client interface
type SaluteSpeechClient struct {
	oauthClient *OAuthClient
	tokenMgr    *TokenManager
}

// NewSaluteSpeechClient creates new SaluteSpeech client
func NewSaluteSpeechClient(cfg Config) (*SaluteSpeechClient, error) {
	oauthClient, err := NewOAuthClient(cfg)
	if err != nil {
		return nil, err
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

// GetTokenManager returns token manager
func (c *SaluteSpeechClient) GetTokenManager() *TokenManager {
	return c.tokenMgr
}
