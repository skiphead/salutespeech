package sync

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
	"github.com/skiphead/salutespeech/utils"
)

// Client handles sync synthesis
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Config represents sync synthesis client configuration
type Config struct {
	BaseURL       string
	Timeout       time.Duration
	AllowInsecure bool
	Logger        types.Logger
}

// NewClient creates new sync synthesis client
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
		baseURL = types.DefaultSyncSynthesisURL
	}
	baseURL = utils.SanitizeURL(baseURL)

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultAPITimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}

	if cfg.AllowInsecure {
		logger.Warn("sync synthesis client: TLS verification disabled")
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

// Synthesize performs sync synthesis
func (c *Client) Synthesize(ctx context.Context, req *Request) (*Response, error) {
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

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.String(),
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
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return &Response{
			AudioData:     audioData,
			ContentType:   resp.Header.Get("Content-Type"),
			ContentLength: len(audioData),
		}, nil

	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s. Check text length (max %d) and format",
			types.ErrBadRequest, string(audioData), types.MaxTextLength)
	case http.StatusUnauthorized:
		_ = c.tokenMgr.ForceRefresh(ctx)
		return nil, fmt.Errorf("%w: %s", types.ErrUnauthorized, string(audioData))
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", types.ErrTooManyRequests, string(audioData))
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", types.ErrServerError, string(audioData))
	default:
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(audioData))
	}
}

// SynthesizeText synthesizes plain text
func (c *Client) SynthesizeText(ctx context.Context, text string, opts Options) (*Response, error) {
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

// SynthesizeSSML synthesizes SSML text
func (c *Client) SynthesizeSSML(ctx context.Context, ssml string, opts Options) (*Response, error) {
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

func (c *Client) validateRequest(req *Request) error {
	if req == nil {
		return types.ErrRequestNil
	}
	if req.Text == "" {
		return types.ErrEmptyText
	}
	if utils.CountRunes(req.Text) > types.MaxTextLength {
		return fmt.Errorf("text too long: %d runes (max %d)",
			utils.CountRunes(req.Text), types.MaxTextLength)
	}
	if req.ContentType == "" {
		req.ContentType = ContentTypeText
	}
	if req.ContentType != ContentTypeText && req.ContentType != ContentTypeSSML {
		return fmt.Errorf("invalid content type: %s", req.ContentType)
	}
	if req.Format == "" {
		req.Format = FormatWAV16
	}
	switch req.Format {
	case FormatWAV16, FormatPCM16, FormatOpus, FormatALaw, FormatG729:
		// valid
	default:
		return fmt.Errorf("unsupported format: %s", req.Format)
	}
	if req.Voice == "" {
		req.Voice = types.VoiceMay24000
	}
	return nil
}
