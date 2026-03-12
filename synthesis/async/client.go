package async

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Client handles async synthesis
type Client struct {
	httpClient    *http.Client
	synthesizeURL string
	taskURL       string
	downloadURL   string
	tokenMgr      *client.TokenManager
	logger        types.Logger
}

// Config represents synthesis client configuration
type Config struct {
	BaseURL       string
	TaskURL       string
	DownloadURL   string
	Timeout       time.Duration
	AllowInsecure bool
	Logger        types.Logger
}

// NewClient creates new synthesis client
func NewClient(tokenMgr *client.TokenManager, cfg Config) (*Client, error) {
	if tokenMgr == nil {
		return nil, types.ErrTokenManagerRequired
	}

	logger := cfg.Logger
	if logger == nil {
		logger = types.NoopLogger{}
	}

	synURL := cfg.BaseURL
	if synURL == "" {
		synURL = types.DefaultSynthesizeURL
	}

	taskURL := cfg.TaskURL
	if taskURL == "" {
		taskURL = types.DefaultTaskURL
	}

	dlURL := cfg.DownloadURL
	if dlURL == "" {
		dlURL = types.DefaultDownloadURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultAPITimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}
	if cfg.AllowInsecure {
		logger.Warn("synthesis client: TLS verification disabled")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &Client{
		httpClient:    httpClient,
		synthesizeURL: synURL,
		taskURL:       taskURL,
		downloadURL:   dlURL,
		tokenMgr:      tokenMgr,
		logger:        logger,
	}, nil
}

// CreateTask creates async synthesis task
func (c *Client) CreateTask(ctx context.Context, req *Request) (*Response, error) {
	if req == nil || req.RequestFileID == "" || req.AudioEncoding == "" || req.Voice == "" {
		return nil, fmt.Errorf("invalid synthesis request")
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.synthesizeURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synthesize error %d: %s", resp.StatusCode, string(respBody))
	}

	var synthResp Response
	if err := json.Unmarshal(respBody, &synthResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if synthResp.Result.ID == "" {
		return nil, types.ErrEmptyTaskID
	}
	return &synthResp, nil
}

// GetTaskStatus gets synthesis task status
func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (*types.TaskStatusResponse, error) {
	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	u, _ := url.Parse(c.taskURL)
	q := u.Query()
	q.Set("id", taskID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status error %d: %s", resp.StatusCode, string(body))
	}

	var statusResp types.TaskStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	return &statusResp, nil
}

// DownloadResult downloads synthesis result
func (c *Client) DownloadResult(ctx context.Context, responseFileID string) ([]byte, error) {
	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	u, _ := url.Parse(c.downloadURL)
	q := u.Query()
	q.Set("response_file_id", responseFileID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// WaitForTask waits for synthesis completion
func (c *Client) WaitForTask(ctx context.Context, taskID string, pollInterval, timeout time.Duration, downloadAudio bool) (*TaskResult, error) {
	if pollInterval == 0 {
		pollInterval = types.DefaultPollInterval
	}
	if timeout == 0 {
		timeout = types.DefaultWaitTimeout
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return nil, fmt.Errorf("timeout waiting for task %s", taskID)
		case <-ticker.C:
			status, err := c.GetTaskStatus(ctxWithTimeout, taskID)
			if err != nil {
				return nil, err
			}

			switch status.Result.Status {
			case types.StatusDONE:
				result := &TaskResult{
					ID:             taskID,
					Status:         types.StatusDONE,
					ResponseFileID: status.Result.ID,
				}
				if !downloadAudio {
					return result, nil
				}

				audio, err := c.DownloadResult(ctxWithTimeout, status.Result.ID)
				if err != nil {
					return result, fmt.Errorf("download audio: %w", err)
				}
				result.AudioData = audio
				return result, nil

			case types.StatusERROR:
				return nil, fmt.Errorf("task failed: unknown error")
			case types.StatusCANCELED:
				return nil, fmt.Errorf("task canceled")
			}
		}
	}
}
