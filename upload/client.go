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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skiphead/oauth/client"
	"github.com/skiphead/salutespeech/types"
)

// HTTPDoer is an interface that performs HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Uploader defines the interface for uploading files.
type Uploader interface {
	// Upload uploads file data.
	Upload(ctx context.Context, req *Request) (*Response, error)
	// UploadFromFile uploads a file from a local path.
	UploadFromFile(ctx context.Context, path string, ct types.ContentType) (*Response, error)
}

// Client handles file uploads.
type Client struct {
	httpClient HTTPDoer
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Compile-time check that Client implements Uploader.
var _ Uploader = (*Client)(nil)

// Config represents upload client configuration.
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

// NewClient creates a new upload client.
func NewClient(tokenMgr *client.TokenManager, cfg Config) (Uploader, error) {
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

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		tokenMgr:   tokenMgr,
		logger:     logger,
	}, nil
}

// UploadFromFile uploads a file from a local path.
func (c *Client) UploadFromFile(ctx context.Context, path string, ct types.ContentType) (*Response, error) {
	// Check file size before reading to avoid loading large files into memory.
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if fileInfo.Size() > types.MaxUploadFileSize {
		return nil, fmt.Errorf("%w: %d bytes (max %d)", types.ErrFileTooLarge, fileInfo.Size(), types.MaxUploadFileSize)
	}

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

// Upload uploads file data.
func (c *Client) Upload(ctx context.Context, req *Request) (*Response, error) {
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Build URL using buildURL for consistency
	fullURL, err := c.buildURL("", nil)
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(req.Data))
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
	defer c.safeClose(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(body))
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

// validateRequest checks if the upload request is valid.
func (c *Client) validateRequest(req *Request) error {
	if req == nil || len(req.Data) == 0 {
		return types.ErrEmptyFileData
	}
	if !req.ContentType.IsValid() {
		return fmt.Errorf("%w: %s", types.ErrInvalidContentType, req.ContentType)
	}
	if len(req.Data) < types.MinFileSize {
		return fmt.Errorf("%w: %d bytes (min %d)", types.ErrFileTooSmall, len(req.Data), types.MinFileSize)
	}
	return nil
}

// safeClose closes the closer and logs any error.
func (c *Client) safeClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		c.logger.Warn("failed to close response body", "error", err)
	}
}

// buildURL constructs a URL with query parameters.
// Handles proper concatenation of base URL and endpoint, avoiding double slashes.
func (c *Client) buildURL(endpoint string, params map[string]string) (string, error) {
	// Normalize base URL: ensure it ends with a single slash
	base := strings.TrimSuffix(c.baseURL, "/") + "/"
	// Normalize endpoint: ensure it doesn't start with a slash
	endpoint = strings.TrimPrefix(endpoint, "/")

	// Parse the combined URL
	u, err := url.Parse(base + endpoint)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	// Add query parameters if any
	if params != nil {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
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
