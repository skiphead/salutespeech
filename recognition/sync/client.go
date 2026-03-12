package sync

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Client handles sync recognition
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Config represents sync recognition client configuration
type Config struct {
	BaseURL       string
	Timeout       time.Duration
	AllowInsecure bool
	Logger        types.Logger
}

// NewClient creates new sync recognition client
func NewClient(tokenMgr *client.TokenManager, cfg Config) (*Client, error) {
	if tokenMgr == nil {
		return nil, types.ErrTokenManagerRequired
	}

	logger := cfg.Logger
	if logger == nil {
		logger = types.NoopLogger{}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = types.DefaultSyncRecognitionURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultAPITimeout * 2 // sync recognition may take longer
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}

	if cfg.AllowInsecure {
		logger.Warn("sync recognition client: TLS verification disabled")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		tokenMgr:   tokenMgr,
		logger:     logger,
	}, nil
}

// Recognize performs sync recognition
func (c *Client) Recognize(ctx context.Context, req *Request) (*Response, error) {
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	q := u.Query()
	if req.Language != "" {
		q.Set("language", string(req.Language))
	}
	q.Set("enable_profanity_filter", fmt.Sprintf("%t", req.EnableProfanityFilter))
	if req.Model != "" {
		q.Set("model", string(req.Model))
	}
	if req.SampleRate > 0 {
		q.Set("sample_rate", fmt.Sprintf("%d", req.SampleRate))
	}
	if req.ChannelsCount > 0 {
		q.Set("channels_count", fmt.Sprintf("%d", req.ChannelsCount))
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(req.Data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", string(req.ContentType))
	httpReq.Header.Set("Accept", "application/json")

	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}
	httpReq.Header.Set("X-Request-ID", requestID)
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var syncResp Response
		if err := json.Unmarshal(body, &syncResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		return &syncResp, nil

	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s", types.ErrBadRequest, string(body))
	case http.StatusUnauthorized:
		_ = c.tokenMgr.ForceRefresh(ctx)
		return nil, fmt.Errorf("%w: %s", types.ErrUnauthorized, string(body))
	case http.StatusNotFound:
		return nil, fmt.Errorf("model not found (404): %s", string(body))
	case http.StatusRequestEntityTooLarge:
		return nil, fmt.Errorf("file too large (413): max 2MB. Response: %s", string(body))
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", types.ErrTooManyRequests, string(body))
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", types.ErrServerError, string(body))
	default:
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
}

// RecognizeFromFile recognizes audio from file
func (c *Client) RecognizeFromFile(ctx context.Context, filePath string, contentType types.ContentType, opts Options) (*Response, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if len(data) > types.MaxSyncFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", len(data), types.MaxSyncFileSize)
	}
	if len(data) < types.MinFileSize {
		return nil, fmt.Errorf("%w: %d bytes (min %d)", types.ErrFileTooSmall, len(data), types.MinFileSize)
	}

	req := &Request{
		Data:                  data,
		ContentType:           contentType,
		Language:              opts.Language,
		EnableProfanityFilter: opts.EnableProfanityFilter,
		Model:                 opts.Model,
		SampleRate:            opts.SampleRate,
		ChannelsCount:         opts.ChannelsCount,
		RequestID:             opts.RequestID,
	}

	return c.Recognize(ctx, req)
}

func (c *Client) validateRequest(req *Request) error {
	if req == nil {
		return types.ErrRequestNil
	}
	if len(req.Data) == 0 {
		return types.ErrEmptyFileData
	}
	if len(req.Data) > types.MaxSyncFileSize {
		return fmt.Errorf("audio too large: %d bytes (max %d)", len(req.Data), types.MaxSyncFileSize)
	}
	if !req.ContentType.IsValid() {
		return fmt.Errorf("%w: %s", types.ErrInvalidContentType, req.ContentType)
	}
	if req.Language == "" {
		req.Language = LangRuRU
	}
	switch req.Language {
	case LangRuRU, LangEnUS, LangKkKZ, LangKyKG, LangUzUZ:
		// valid
	default:
		return fmt.Errorf("unsupported language: %s", req.Language)
	}
	if req.SampleRate < types.MinSampleRate || req.SampleRate > types.MaxSampleRate {
		return fmt.Errorf("sample_rate out of range: %d (min %d, max %d)",
			req.SampleRate, types.MinSampleRate, types.MaxSampleRate)
	}
	if req.ChannelsCount < types.MinChannelsCount || req.ChannelsCount > types.MaxChannelsCount {
		req.ChannelsCount = 1
	}
	return nil
}
