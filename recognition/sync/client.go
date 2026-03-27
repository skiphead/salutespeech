// Package sync provides client functionality for synchronous speech recognition
// using the SaluteSpeech API. It handles audio file validation, HTTP requests,
// authentication, and response parsing for real-time recognition scenarios.
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
	"github.com/skiphead/oauth/client"
	"github.com/skiphead/salutespeech/types"
)

const (
	// DefaultSampleRate is the default sample rate for audio recognition
	DefaultSampleRate = 16000

	// DefaultChannelsCount is the default number of audio channels
	DefaultChannelsCount = 1

	// DefaultTimeoutMultiplier is the multiplier for sync recognition timeout
	DefaultTimeoutMultiplier = 2
)

// HTTPDoer defines the interface for making HTTP requests.
// Allows for easier testing by mocking HTTP client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Recognizer defines the public interface for synchronous speech recognition.
type Recognizer interface {
	Recognize(ctx context.Context, req *Request) (*Response, error)
	RecognizeFromFile(ctx context.Context, filePath string, contentType types.ContentType, opts Options) (*Response, error)
}

// Client handles synchronous speech recognition operations.
// It manages HTTP communication, authentication, and request validation
// for real-time audio transcription through the SaluteSpeech API.
type Client struct {
	httpClient HTTPDoer
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Ensure Client implements Recognizer
var _ Recognizer = (*Client)(nil)

// Config represents the configuration options for creating a new sync recognition client.
// It allows customization of the API endpoint, timeout behavior, TLS settings, and logging.
type Config struct {
	BaseURL       string        // API endpoint URL (defaults to DefaultSyncRecognitionURL)
	Timeout       time.Duration // Request timeout (defaults to 2x DefaultAPITimeout)
	AllowInsecure bool          // When true, disables TLS certificate verification
	Logger        types.Logger  // Logger instance for client operations
	HTTPClient    HTTPDoer      // Optional custom HTTP client (for testing)
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.BaseURL == "" {
		// BaseURL can be empty, will use default
		return nil
	}

	// Validate URL format
	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	return nil
}

// NewClient creates a new synchronous speech recognition client with the provided configuration.
// It initializes the HTTP client, validates the token manager, and sets up default values
// for any missing configuration parameters.
//
// Returns an error if token manager is nil or if configuration validation fails.
func NewClient(tokenMgr *client.TokenManager, cfg Config) (Recognizer, error) {
	if tokenMgr == nil {
		return nil, types.ErrTokenManagerRequired
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
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
		timeout = types.DefaultAPITimeout * DefaultTimeoutMultiplier // sync recognition may take longer
	}

	var httpClient HTTPDoer
	if cfg.HTTPClient != nil {
		httpClient = cfg.HTTPClient
	} else {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
		}

		if cfg.AllowInsecure {
			logger.Warn("sync recognition client: TLS verification disabled")
		}

		httpClient = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		tokenMgr:   tokenMgr,
		logger:     logger,
	}, nil
}

// buildURL constructs the request URL with query parameters.
func (c *Client) buildURL(req *Request) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
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
	return u.String(), nil
}

// Recognize performs synchronous speech recognition on the provided audio data.
// It handles authentication, request construction, and response parsing.
// The context can be used for cancellation and timeout control.
//
// Returns the recognition response containing transcribed text and metadata,
// or an error if the request fails or validation checks are not met.
func (c *Client) Recognize(ctx context.Context, req *Request) (*Response, error) {
	if err := c.applyDefaultsAndValidate(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	buildURL, err := c.buildURL(req)
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, buildURL, bytes.NewReader(req.Data))
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
	defer c.safeClose(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return c.handleResponse(resp.StatusCode, body)
}

// handleResponse processes the HTTP response and returns appropriate result or error.
func (c *Client) handleResponse(statusCode int, body []byte) (*Response, error) {
	switch statusCode {
	case http.StatusOK:
		var syncResp Response
		if err := json.Unmarshal(body, &syncResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		syncResp.Status = statusCode
		return &syncResp, nil

	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s", types.ErrBadRequest, string(body))

	case http.StatusUnauthorized:
		// Token expired or invalid - caller should refresh
		return nil, fmt.Errorf("%w: %s", types.ErrUnauthorized, string(body))

	case http.StatusNotFound:
		return nil, fmt.Errorf("model not found (404): %s", string(body))

	case http.StatusRequestEntityTooLarge:
		return nil, fmt.Errorf("file too large (413): max %d bytes. Response: %s", types.MaxSyncFileSize, string(body))

	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", types.ErrTooManyRequests, string(body))

	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", types.ErrServerError, string(body))

	default:
		return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}
}

// RecognizeFromFile reads and recognizes audio from a file.
// It validates file size against API limits (max 2MB), reads the file content,
// and performs synchronous recognition with the provided options.
//
// Returns the recognition response or an error if file reading fails,
// validation checks fail, or the recognition request fails.
func (c *Client) RecognizeFromFile(ctx context.Context, filePath string, contentType types.ContentType, opts Options) (*Response, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Get file info for size validation
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Check file size before reading to avoid memory issues
	if fileInfo.Size() > int64(types.MaxSyncFileSize) {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", fileInfo.Size(), types.MaxSyncFileSize)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
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

// applyDefaultsAndValidate performs validation and applies default values.
// It checks for nil request, non-empty audio data, file size limits,
// valid content type, supported language, and valid sample rate.
// Default values are applied for missing optional parameters.
func (c *Client) applyDefaultsAndValidate(req *Request) error {
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

	// Apply and validate language
	if req.Language == "" {
		req.Language = LangRuRU
	}
	if !req.Language.IsValid() {
		return fmt.Errorf("unsupported language: %s", req.Language)
	}

	// Apply and validate sample rate
	if req.SampleRate == 0 {
		req.SampleRate = DefaultSampleRate
	}
	if req.SampleRate < types.MinSampleRate || req.SampleRate > types.MaxSampleRate {
		return fmt.Errorf("sample_rate out of range: %d (min %d, max %d)",
			req.SampleRate, types.MinSampleRate, types.MaxSampleRate)
	}

	// Apply and validate channels count
	if req.ChannelsCount == 0 {
		req.ChannelsCount = DefaultChannelsCount
	}
	if req.ChannelsCount < types.MinChannelsCount || req.ChannelsCount > types.MaxChannelsCount {
		return fmt.Errorf("channels_count out of range: %d (min %d, max %d)",
			req.ChannelsCount, types.MinChannelsCount, types.MaxChannelsCount)
	}

	// Validate model if provided
	if req.Model != "" && !req.Model.IsValid() {
		return fmt.Errorf("unsupported model: %s", req.Model)
	}

	return nil
}

// safeClose safely closes an io.Closer and logs any errors.
func (c *Client) safeClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		c.logger.Warn("failed to close response body", "error", err)
	}
}
