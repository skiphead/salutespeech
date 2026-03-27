// Package sync provides client functionality for synchronous speech synthesis
// using the SaluteSpeech API. It supports real-time text-to-speech conversion
// with multiple audio formats, voice options, and both plain text and SSML input.
package sync

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skiphead/oauth/client"
	"github.com/skiphead/salutespeech/types"
	"github.com/skiphead/salutespeech/utils"
)

const (
	// MaxAudioSize limits the maximum audio response size (50 MB)
	MaxAudioSize = 50 * 1024 * 1024
)

// HTTPDoer defines the interface for making HTTP requests.
// Allows for easier testing by mocking HTTP client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Synthesizer defines the public interface for synchronous speech synthesis.
type Synthesizer interface {
	Synthesize(ctx context.Context, req *Request) (*Response, error)
	SynthesizeText(ctx context.Context, text string, opts Options) (*Response, error)
	SynthesizeSSML(ctx context.Context, ssml string, opts Options) (*Response, error)
}

// Client handles synchronous speech synthesis operations.
// It manages HTTP communication, authentication, and request validation
// for real-time text-to-speech conversion through the SaluteSpeech API.
type Client struct {
	httpClient HTTPDoer
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Ensure Client implements Synthesizer interface
var _ Synthesizer = (*Client)(nil)

// Config represents the configuration options for creating a new sync synthesis client.
// It allows customization of the API endpoint, timeout behavior, TLS settings, and logging.
type Config struct {
	BaseURL       string        // API endpoint URL (defaults to DefaultSyncSynthesisURL)
	Timeout       time.Duration // Request timeout (defaults to DefaultAPITimeout)
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

// NewClient creates a new synchronous speech synthesis client with the provided configuration.
// It initializes the HTTP client, validates the token manager, sanitizes the base URL,
// and sets up default values for any missing configuration parameters.
//
// Returns an error if token manager is nil or if configuration validation fails.
func NewClient(tokenMgr *client.TokenManager, cfg Config) (Synthesizer, error) {
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
		baseURL = types.DefaultSyncSynthesisURL
	}
	baseURL = utils.SanitizeURL(baseURL)

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultAPITimeout
	}

	var httpClient HTTPDoer
	if cfg.HTTPClient != nil {
		httpClient = cfg.HTTPClient
	} else {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
		}

		if cfg.AllowInsecure {
			logger.Warn("sync synthesis client: TLS verification disabled")
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

	if req.Format != "" {
		q.Set("format", string(req.Format))
	}

	if req.Voice != "" {
		q.Set("voice", string(req.Voice))
	}

	if req.RebuildCache {
		q.Set("rebuild_cache", "true")
	}

	if req.BypassCache {
		q.Set("bypass_cache", "true")
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// handleResponse processes the HTTP response and returns appropriate result or error.
func (c *Client) handleResponse(statusCode int, body []byte, contentType string) (*Response, error) {
	switch statusCode {
	case http.StatusOK:
		return &Response{
			AudioData:     body,
			ContentType:   contentType,
			ContentLength: len(body),
		}, nil

	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s. Check text length (max %d) and format",
			types.ErrBadRequest, string(body), types.MaxTextLength)

	case http.StatusUnauthorized:
		// Token expired or invalid - caller should refresh
		return nil, fmt.Errorf("%w: %s", types.ErrUnauthorized, string(body))

	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", types.ErrTooManyRequests, string(body))

	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", types.ErrServerError, string(body))

	default:
		return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}
}

// Synthesize performs synchronous speech synthesis on the provided text.
// It handles authentication, request construction, audio format negotiation,
// and returns the synthesized audio data or an error if synthesis fails.
//
// The response includes the audio data, content type, and content length.
// Audio formats supported include WAV, PCM, Opus, A-Law, and G.729.
func (c *Client) Synthesize(ctx context.Context, req *Request) (*Response, error) {
	if err := c.validateRequest(req); err != nil {
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

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		buildURL,
		bytes.NewReader([]byte(req.Text)),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", string(req.ContentType))
	httpReq.Header.Set("Accept", "audio/*, application/octet-stream")

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

	// Limit response size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, MaxAudioSize+1)
	audioData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	// Check if response exceeded the limit
	if int64(len(audioData)) > MaxAudioSize {
		return nil, fmt.Errorf("audio response exceeds maximum size of %d bytes", MaxAudioSize)
	}

	return c.handleResponse(resp.StatusCode, audioData, resp.Header.Get("Content-Type"))
}

// SynthesizeText converts plain text to speech using synchronous synthesis.
// It's a convenience method that wraps Synthesize with the appropriate content type.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - text: Plain text to synthesize (max length defined by types.MaxTextLength)
//   - opts: Synthesis options including voice, format, and cache settings
//
// Returns synthesized audio response or an error if validation or synthesis fails.
func (c *Client) SynthesizeText(ctx context.Context, text string, opts Options) (*Response, error) {
	if text == "" {
		return nil, types.ErrEmptyText
	}

	req := &Request{
		Text:         text,
		ContentType:  ContentTypeText,
		Format:       opts.Format,
		Voice:        opts.Voice,
		RebuildCache: opts.RebuildCache,
		BypassCache:  opts.BypassCache,
		RequestID:    opts.RequestID,
	}
	return c.Synthesize(ctx, req)
}

// SynthesizeSSML converts SSML (Speech Synthesis Markup Language) text to speech.
// SSML allows fine-grained control over pronunciation, pitch, rate, and other
// speech attributes. This method sets the appropriate content type for SSML input.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - ssml: SSML-formatted text to synthesize
//   - opts: Synthesis options including voice, format, and cache settings
//
// Returns synthesized audio response or an error if validation or synthesis fails.
func (c *Client) SynthesizeSSML(ctx context.Context, ssml string, opts Options) (*Response, error) {
	if ssml == "" {
		return nil, types.ErrEmptyText
	}

	// Basic SSML validation (can be extended)
	if !strings.HasPrefix(strings.TrimSpace(ssml), "<speak>") {
		c.logger.Warn("SSML content may be invalid: missing <speak> tag")
	}

	req := &Request{
		Text:         ssml,
		ContentType:  ContentTypeSSML,
		Format:       opts.Format,
		Voice:        opts.Voice,
		RebuildCache: opts.RebuildCache,
		BypassCache:  opts.BypassCache,
		RequestID:    opts.RequestID,
	}
	return c.Synthesize(ctx, req)
}

// validateRequest performs comprehensive validation of the synthesis request.
// It checks for nil request, non-empty text, text length limits (using rune count),
// valid content type, supported audio format, and applies default values
// for missing optional parameters.
func (c *Client) validateRequest(req *Request) error {
	if req == nil {
		return types.ErrRequestNil
	}

	if req.Text == "" {
		return types.ErrEmptyText
	}

	// Check text length using rune count for proper Unicode handling
	runeCount := utils.CountRunes(req.Text)
	if runeCount > types.MaxTextLength {
		return fmt.Errorf("text too long: %d runes (max %d)",
			runeCount, types.MaxTextLength)
	}

	// Apply and validate content type
	if req.ContentType == "" {
		req.ContentType = ContentTypeText
	}
	if !req.ContentType.IsValid() {
		return fmt.Errorf("invalid content type: %s", req.ContentType)
	}

	// Apply and validate audio format
	if req.Format == "" {
		req.Format = FormatWAV16
	}
	if !req.Format.IsValid() {
		return fmt.Errorf("unsupported format: %s", req.Format)
	}

	// Apply default voice if not specified
	if req.Voice == "" {
		req.Voice = types.VoiceMay24000
	}

	return nil
}

// safeClose safely closes an io.Closer and logs any errors.
func (c *Client) safeClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		c.logger.Warn("failed to close response body", "error", err)
	}
}
