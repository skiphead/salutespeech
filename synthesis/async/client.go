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

	"github.com/skiphead/oauth/client"
	"github.com/skiphead/salutespeech/types"
)

// HTTPDoer defines the interface for making HTTP requests.
// This allows for easier testing by mocking HTTP clients.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Synthesizer is the public interface for async speech synthesis operations.
type Synthesizer interface {
	CreateTask(ctx context.Context, req *Request) (*Response, error)
	GetTaskStatus(ctx context.Context, taskID string) (*Response, error)
	DownloadResult(ctx context.Context, responseFileID string) ([]byte, error)
	WaitForTask(ctx context.Context, taskID string, pollInterval, timeout time.Duration, downloadAudio bool) (*TaskResult, error)
}

// Client handles asynchronous speech synthesis operations.
// It manages task creation, status polling, and audio file retrieval
// for long-running text-to-speech conversion jobs through the SaluteSpeech API.
type Client struct {
	httpClient    HTTPDoer
	synthesizeURL string // URL for creating synthesis tasks
	taskURL       string // URL for checking task status
	downloadURL   string // URL for downloading synthesized audio
	tokenMgr      *client.TokenManager
	logger        types.Logger
}

// Ensure Client implements Synthesizer interface
var _ Synthesizer = (*Client)(nil)

const (
	maxAudioSize = 100 * 1024 * 1024 // 100 MB limit for audio downloads
)

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

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.BaseURL == "" && c.TaskURL == "" && c.DownloadURL == "" {
		return fmt.Errorf("at least one URL must be specified")
	}
	return nil
}

// NewClient creates a new asynchronous speech synthesis client with the provided configuration.
// It initializes the HTTP client, validates the token manager, and sets up default values
// for any missing configuration parameters.
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

// safeClose safely closes an io.Closer and logs any error.
func (c *Client) safeClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		c.logger.Warn(fmt.Sprintf("failed to close: %v", err))
	}
}

// buildURL constructs a URL with query parameters.
func (c *Client) buildURL(base string, params map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse URL %s: %w", base, err)
	}

	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// CreateTask creates an asynchronous speech synthesis task.
// It submits a text-to-speech request with the specified parameters including
// voice, audio encoding, and text content via the request file ID.
//
// Returns a task response containing the task ID for subsequent status checking
// and result retrieval, or an error if validation or API request fails.
func (c *Client) CreateTask(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.RequestFileID == "" {
		return nil, fmt.Errorf("request_file_id is required")
	}
	if req.AudioEncoding == "" {
		return nil, fmt.Errorf("audio_encoding is required")
	}
	if req.Voice == "" {
		return nil, fmt.Errorf("voice is required")
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
	defer c.safeClose(resp.Body)

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
func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (*Response, error) {
	if taskID == "" {
		return nil, types.ErrEmptyTaskID
	}

	buildURL, err := c.buildURL(c.taskURL, map[string]string{"id": taskID})
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer c.safeClose(resp.Body)

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
func (c *Client) DownloadResult(ctx context.Context, responseFileID string) ([]byte, error) {
	if responseFileID == "" {
		return nil, fmt.Errorf("response file ID cannot be empty")
	}

	buildURL, err := c.buildURL(c.downloadURL, map[string]string{"response_file_id": responseFileID})
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}
	defer c.safeClose(resp.Body)

	// Limit response size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxAudioSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	// Check if file exceeded the limit
	if int64(len(body)) > maxAudioSize {
		return nil, fmt.Errorf("audio file exceeds maximum size of %d bytes", maxAudioSize)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// handleCompletedTask processes a completed task, optionally downloading audio.
func (c *Client) handleCompletedTask(ctx context.Context, taskID string, status *Response, downloadAudio bool) (*TaskResult, error) {
	result := &TaskResult{
		ID:             taskID,
		Status:         types.StatusDONE,
		ResponseFileID: status.Result.ResponseFileID,
	}

	if !downloadAudio {
		return result, nil
	}

	audio, err := c.DownloadResult(ctx, status.Result.ResponseFileID)
	if err != nil {
		return result, fmt.Errorf("download audio: %w", err)
	}

	result.AudioData = audio
	return result, nil
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
func (c *Client) WaitForTask(ctx context.Context, taskID string, pollInterval, timeout time.Duration, downloadAudio bool) (*TaskResult, error) {
	if taskID == "" {
		return nil, types.ErrEmptyTaskID
	}

	if pollInterval == 0 {
		pollInterval = types.DefaultPollInterval
	}
	if timeout == 0 {
		timeout = types.DefaultWaitTimeout
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Initial check before entering ticker loop
	initialStatus, err := c.GetTaskStatus(ctxWithTimeout, taskID)
	if err != nil {
		return nil, fmt.Errorf("initial status check: %w", err)
	}

	// Check if task is already completed
	switch initialStatus.Result.Status {
	case types.StatusDONE:
		return c.handleCompletedTask(ctxWithTimeout, taskID, initialStatus, downloadAudio)
	case types.StatusERROR:
		return nil, fmt.Errorf("task already failed: status=%s", initialStatus.Result.Status)
	case types.StatusCANCELED:
		return nil, fmt.Errorf("task already canceled")
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return nil, fmt.Errorf("timeout waiting for task %s: %w", taskID, ctxWithTimeout.Err())
		case <-ticker.C:
			status, err := c.GetTaskStatus(ctxWithTimeout, taskID)
			if err != nil {
				return nil, err
			}

			switch status.Result.Status {
			case types.StatusDONE:
				return c.handleCompletedTask(ctxWithTimeout, taskID, status, downloadAudio)
			case types.StatusERROR:
				return nil, fmt.Errorf("task failed: status=%s", status.Result.Status)
			case types.StatusCANCELED:
				return nil, fmt.Errorf("task canceled")
			}
			// Continue polling for other statuses (NEW, PROCESSING)
		}
	}
}
