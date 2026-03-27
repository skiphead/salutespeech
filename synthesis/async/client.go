// Package async provides client functionality for asynchronous speech synthesis
// using the SaluteSpeech API. It supports creating synthesis tasks, monitoring
// task status, and downloading generated audio files for text-to-speech conversion.
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

	"github.com/skiphead/oauth"
	"github.com/skiphead/salutespeech/types"
)

type Client interface {
	CreateTask(ctx context.Context, req *Request) (*Response, error)
	GetTaskStatus(ctx context.Context, taskID string) (*Response, error)
	DownloadResult(ctx context.Context, responseFileID string) ([]byte, error)
	WaitForTask(ctx context.Context, taskID string, pollInterval, timeout time.Duration, downloadAudio bool) (*TaskResult, error)
}

// asyncClient handles asynchronous speech synthesis operations.
// It manages task creation, status polling, and audio file retrieval
// for long-running text-to-speech conversion jobs through the SaluteSpeech API.
type asyncClient struct {
	httpClient    *http.Client
	synthesizeURL string // URL for creating synthesis tasks
	taskURL       string // URL for checking task status
	downloadURL   string // URL for downloading synthesized audio
	tokenMgr      *client.TokenManager
	logger        types.Logger
}

// Config represents the configuration options for creating a new async synthesis client.
// It allows customization of API endpoints, timeout behavior, TLS settings, and logging.
type Config struct {
	BaseURL       string        // URL for task creation (defaults to DefaultSynthesizeURL)
	TaskURL       string        // URL for status checking (defaults to DefaultTaskURL)
	DownloadURL   string        // URL for audio download (defaults to DefaultDownloadURL)
	Timeout       time.Duration // HTTP client timeout (defaults to DefaultAPITimeout)
	AllowInsecure bool          // When true, disables TLS certificate verification
	Logger        types.Logger  // Logger instance for client operations
}

// NewClient creates a new asynchronous speech synthesis client with the provided configuration.
// It initializes the HTTP client, validates the token manager, and sets up default values
// for any missing configuration parameters.
//
// Returns an error if token manager is nil or if configuration validation fails.
func NewClient(tokenMgr *client.TokenManager, cfg Config) (Client, error) {
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

	return &asyncClient{
		httpClient:    httpClient,
		synthesizeURL: synURL,
		taskURL:       taskURL,
		downloadURL:   dlURL,
		tokenMgr:      tokenMgr,
		logger:        logger,
	}, nil
}

// CreateTask creates an asynchronous speech synthesis task.
// It submits a text-to-speech request with the specified parameters including
// voice, audio encoding, and text content via the request file ID.
//
// Returns a task response containing the task ID for subsequent status checking
// and result retrieval, or an error if validation or API request fails.
func (c *asyncClient) CreateTask(ctx context.Context, req *Request) (*Response, error) {
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			c.logger.Warn("failed to close response body")
		}
	}(resp.Body)

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

// GetTaskStatus retrieves the current status of an asynchronous synthesis task.
// It returns a task status response containing information about the task's progress,
// including whether it's pending, processing, completed, or failed.
func (c *asyncClient) GetTaskStatus(ctx context.Context, taskID string) (*Response, error) {
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			c.logger.Warn("close response body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status error %d: %s", resp.StatusCode, string(body))
	}

	var statusResp Response
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	return &statusResp, nil
}

// DownloadResult downloads the synthesized audio for a completed task.
// It retrieves the audio file using the response file ID provided in the task status.
// Returns the audio data as a byte slice or an error if download fails.
func (c *asyncClient) DownloadResult(ctx context.Context, responseFileID string) ([]byte, error) {
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// WaitForTask polls the task status until completion, error, or timeout.
// It periodically checks the task status at the specified poll interval
// and returns when the task is completed. Optionally downloads the audio
// if downloadAudio is true.
//
// Parameters:
//   - ctx: Context for cancellation
//   - taskID: The ID of the task to wait for
//   - pollInterval: Time between status checks (defaults to DefaultPollInterval if zero)
//   - timeout: Maximum time to wait for completion (defaults to DefaultWaitTimeout if zero)
//   - downloadAudio: If true, automatically downloads the synthesized audio
//
// Returns a TaskResult containing task metadata and optionally the audio data,
// or an error if the task fails or timeout occurs.
func (c *asyncClient) WaitForTask(ctx context.Context, taskID string, pollInterval, timeout time.Duration, downloadAudio bool) (*TaskResult, error) {
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

				audio, err := c.DownloadResult(ctxWithTimeout, status.Result.ResponseFileId)
				if err != nil {
					return result, fmt.Errorf("download audio: %w", err)
				}

				result.AudioData = audio
				return result, nil

			case types.StatusERROR:
				return nil, fmt.Errorf("task failed: response %s", status.Result.Status)
			case types.StatusCANCELED:
				return nil, fmt.Errorf("task canceled")
			}
		}
	}
}
