package upload

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// UploadResponse represents upload API response
type UploadResponse struct {
	Status int `json:"status"`
	Result struct {
		RequestFileID string `json:"request_file_id"`
	} `json:"result"`
}

// UploadRequest represents upload request
type UploadRequest struct {
	Data        []byte
	ContentType types.ContentType
	RequestID   string
}

// Client handles file uploads
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Config represents upload client configuration
type Config struct {
	BaseURL       string
	AllowInsecure bool
	Timeout       time.Duration
	Logger        types.Logger
}

// NewClient creates new upload client
func NewClient(tokenMgr *client.TokenManager, cfg Config) (*Client, error) {
	if tokenMgr == nil {
		return nil, types.ErrTokenManagerRequired
	}

	logger := cfg.Logger
	if logger == nil {
		logger = types.NoopLogger{}
	}

	url := cfg.BaseURL
	if url == "" {
		url = types.DefaultUploadURL
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
		baseURL:    url,
		tokenMgr:   tokenMgr,
		logger:     logger,
	}, nil
}

// UploadFromFile uploads file from local path
func (c *Client) UploadFromFile(ctx context.Context, path string, ct types.ContentType) (*UploadResponse, error) {
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

	return c.Upload(ctx, &UploadRequest{
		Data:        data,
		ContentType: ct,
	})
}

// Upload uploads file data
func (c *Client) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload error %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if uploadResp.Result.RequestFileID == "" {
		return nil, fmt.Errorf("empty request_file_id in response")
	}

	return &uploadResp, nil
}
