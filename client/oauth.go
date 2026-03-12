package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	logger     types.Logger
}

// Config represents OAuth client configuration
type Config struct {
	AuthKey       string
	Scope         types.Scope
	OAuthURL      string
	Timeout       time.Duration
	AllowInsecure bool
	Logger        types.Logger
}

// NewOAuthClient creates new OAuth client
func NewOAuthClient(cfg Config) (*OAuthClient, error) {
	if cfg.AuthKey == "" {
		return nil, types.ErrAuthKeyRequired
	}
	if cfg.Scope == "" {
		return nil, types.ErrScopeRequired
	}

	oauthURL := sanitizeURL(cfg.OAuthURL)
	if oauthURL == "" {
		oauthURL = types.DefaultOAuthURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultTimeout
	}

	logger := cfg.Logger
	if logger == nil {
		logger = types.NoopLogger{}
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}

	if cfg.AllowInsecure {
		logger.Warn("TLS verification disabled (AllowInsecure=true)")
	}

	httpClient := &http.Client{Transport: transport, Timeout: timeout}

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
	}, nil
}

// RequestToken requests new token from OAuth server
func (c *OAuthClient) RequestToken(ctx context.Context) (*types.Token, error) {
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

// GenerateBasicAuthKey generates Basic Auth key from client ID and secret
func GenerateBasicAuthKey(clientID, clientSecret string) string {
	cred := fmt.Sprintf("%s:%s", clientID, clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(cred))
}

func sanitizeURL(s string) string {
	return strings.Join(strings.Fields(s), "")
}
