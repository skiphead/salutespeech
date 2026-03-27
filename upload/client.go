package upload

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

// Client defines the interface for uploading files.
type Client interface {
	Upload(ctx context.Context, req *Request) (*Response, error)
	UploadFromFile(ctx context.Context, path string, ct types.ContentType) (*Response, error)
}

// uploadClient handles file uploads
type uploadClient struct {
	httpClient *http.Client
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// NewClient creates new upload client
func NewClient(tokenMgr *client.TokenManager, cfg Config) (Client, error) {
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
		baseURL = types.DefaultUploadURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultUploadTimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}

	if cfg.AllowInsecure {
		logger.Warn("upload client: TLS verification disabled")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &uploadClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		tokenMgr:   tokenMgr,
		logger:     logger,
	}, nil
}

// UploadFromFile uploads file from local path
func (c *uploadClient) UploadFromFile(ctx context.Context, path string, ct types.ContentType) (*Response, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if len(data) < types.MinFileSize {
		return nil, fmt.Errorf("%w: %d bytes (min %d)", types.ErrFileTooSmall, len(data), types.MinFileSize)
	}

	if !ct.IsValid() {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidContentType, ct)
	}

	return c.Upload(ctx, &Request{
		Data:        data,
		ContentType: ct,
	})
}

// Upload uploads file data
func (c *uploadClient) Upload(ctx context.Context, req *Request) (*Response, error) {
	if req == nil || len(req.Data) == 0 {
		return nil, types.ErrEmptyFileData
	}

	if !req.ContentType.IsValid() {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidContentType, req.ContentType)
	}

	if len(req.Data) < types.MinFileSize {
		return nil, fmt.Errorf("%w: %d bytes (min %d)", types.ErrFileTooSmall, len(req.Data), types.MinFileSize)
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Используем baseURL как есть, без модификации (как в исходном коде)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(req.Data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", string(req.ContentType))
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Request-ID", requestID)
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload error %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp Response
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if uploadResp.Result.RequestFileID == "" {
		return nil, fmt.Errorf("empty request_file_id in response")
	}

	return &uploadResp, nil
}

// Config represents upload client configuration
type Config struct {
	BaseURL       string
	AllowInsecure bool
	Timeout       time.Duration
	Logger        types.Logger
}

// Validate checks if the configuration is valid.
// BaseURL is optional - if empty, types.DefaultUploadURL will be used.
func (c Config) Validate() error {
	// BaseURL can be empty — types.DefaultUploadURL will be used
	if c.BaseURL != "" {
		if _, err := url.Parse(c.BaseURL); err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
	}
	return nil
}

// Request represents a file upload request containing the file data, content type,
// and optional request ID for tracing and debugging.
type Request struct {
	Data        []byte
	ContentType types.ContentType
	RequestID   string
}

// Response represents the API response for a file upload request.
type Response struct {
	Status int `json:"status"`
	Result struct {
		RequestFileID string `json:"request_file_id"`
	} `json:"result"`
}
