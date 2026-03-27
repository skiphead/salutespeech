// Package async provides client functionality for asynchronous speech recognition
// using the SaluteSpeech API. It supports long-running recognition tasks,
// task status polling, and result retrieval for processing large audio files
// or batch recognition scenarios.
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
	"os"
	"path"
	"strings"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Recognizer defines the public interface for asynchronous speech recognition operations.
// This interface allows for easy testing and mocking of recognition functionality.
type Recognizer interface {
	// CreateTask creates an asynchronous recognition task.
	CreateTask(ctx context.Context, req *Request) (*Response, error)

	// GetTaskResult retrieves the current status and result of a task.
	GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error)

	// WaitForResult polls until task completion, error, or timeout.
	WaitForResult(ctx context.Context, taskID string, pollInterval, timeout time.Duration) (*TaskResult, error)

	// DownloadTaskResultToFile downloads the result file to disk.
	DownloadTaskResultToFile(ctx context.Context, responseFileID, outputPath string) error

	// DownloadTaskResult downloads the result file as a byte slice (for small files only).
	DownloadTaskResult(ctx context.Context, responseFileID string) ([]byte, error)
}

// Ensure Client implements Recognizer interface
var _ Recognizer = (*Client)(nil)

// HTTPDoer defines the interface for HTTP clients.
// This allows for easy testing and mocking of HTTP operations.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client handles asynchronous speech recognition operations.
// It manages task creation, result retrieval, and status polling
// for long-running recognition jobs through the SaluteSpeech API.
type Client struct {
	httpClient HTTPDoer
	baseURL    *url.URL // URL for API endpoints
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Config represents the configuration options for creating a new async recognition client.
// It allows customization of API endpoints, timeout behavior, TLS settings, and logging.
type Config struct {
	BaseURL       string        // URL for task creation (defaults to DefaultRecognitionURL)
	Timeout       time.Duration // HTTP client timeout (defaults to DefaultAPITimeout)
	AllowInsecure bool          // When true, disables TLS certificate verification
	Logger        types.Logger  // Logger instance for client operations
}

// NewClient creates a new asynchronous speech recognition client with the provided configuration.
// It initializes the HTTP client, validates the token manager, and sets up default values
// for any missing configuration parameters.
//
// Returns an error if token manager is nil or if configuration validation fails.
func NewClient(tokenMgr *client.TokenManager, cfg Config) (*Client, error) {
	if tokenMgr == nil {
		return nil, types.ErrTokenManagerRequired
	}

	logger := cfg.Logger
	if logger == nil {
		logger = types.NoopLogger{}
	}

	baseURLStr := cfg.BaseURL
	if baseURLStr == "" {
		baseURLStr = types.DefaultBaseURL
	}

	// Parse and validate base URL
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Ensure base URL has a trailing slash for proper path joining
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = types.DefaultAPITimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.AllowInsecure},
	}

	if cfg.AllowInsecure {
		logger.Warn("recognition client: TLS verification disabled")
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

// buildURL constructs a proper URL by joining the base path with the given path
// and adding query parameters. Handles proper path joining without replacing.
func (c *Client) buildURL(endpointPath string, queryParams map[string]string) string {
	// Clone the base URL to avoid modifying the original
	urlCopy := *c.baseURL

	// Join paths properly - this preserves the base path and appends the endpoint
	// Use path.Join which handles slashes correctly
	if urlCopy.Path == "" {
		urlCopy.Path = endpointPath
	} else {
		// Remove trailing slash from base if present to avoid double slashes
		basePath := strings.TrimSuffix(urlCopy.Path, "/")
		// Remove leading slash from endpoint if present
		endpointPath = strings.TrimPrefix(endpointPath, "/")
		urlCopy.Path = path.Join(basePath, endpointPath)
	}

	// Add query parameters
	if len(queryParams) > 0 {
		q := urlCopy.Query()
		for key, value := range queryParams {
			q.Set(key, value)
		}
		urlCopy.RawQuery = q.Encode()
	}

	return urlCopy.String()
}

// CreateTask creates an asynchronous recognition task for audio processing.
// It submits the recognition request with the provided audio file ID and options,
// and returns a task response containing the task ID for subsequent result retrieval.
//
// Returns an error if request validation fails, authentication fails, or the API
// returns an error response.
func (c *Client) CreateTask(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, types.ErrRequestNil
	}
	if req.RequestFileID == "" {
		return nil, types.ErrRequestFileIDRequired
	}
	if req.Options == nil {
		return nil, types.ErrOptionsRequired
	}
	if err := c.applyDefaultsAndValidate(req.Options); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.buildURL("speech:async_recognize", nil)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer c.safeClose(resp.Body)

	// Use streaming JSON decoder for successful responses
	switch resp.StatusCode {
	case http.StatusOK:
		var recognitionResp Response
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&recognitionResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if recognitionResp.Result.ID == "" {
			return nil, types.ErrEmptyTaskID
		}
		return &recognitionResp, nil

	case http.StatusBadRequest:
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("%w: %s", types.ErrBadRequest, errorBody)
	case http.StatusUnauthorized:
		_ = c.tokenMgr.ForceRefresh(ctx)
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("%w: %s", types.ErrUnauthorized, errorBody)
	case http.StatusRequestEntityTooLarge:
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("payload too large (413): %s", errorBody)
	case http.StatusTooManyRequests:
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("%w: %s", types.ErrTooManyRequests, errorBody)
	case http.StatusInternalServerError:
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("%w: %s", types.ErrServerError, errorBody)
	default:
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, errorBody)
	}
}

// GetTaskResult retrieves the result of an asynchronous recognition task.
// It fetches the current status and recognition results for the specified task ID.
// The task may still be processing, completed successfully, or failed with an error.
//
// Returns a TaskResult containing the task status, recognition results (if completed),
// or error details (if failed).
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	if taskID == "" {
		return nil, types.ErrEmptyTaskID
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	url := c.buildURL("task:get", map[string]string{"id": taskID})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer c.safeClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("failed to get result (status %d): %s", resp.StatusCode, errorBody)
	}

	// Use streaming JSON decoder
	var result TaskResult
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	return &result, nil
}

// WaitForResult polls the task status until completion, error, or timeout.
// It periodically checks the task status at the specified poll interval
// and returns when the task is completed successfully, fails, or the timeout is reached.
//
// Parameters:
//   - ctx: Context for cancellation
//   - taskID: The ID of the task to wait for
//   - pollInterval: Time between status checks (defaults to DefaultPollInterval if zero)
//   - timeout: Maximum time to wait for completion (defaults to DefaultWaitTimeout if zero)
//
// Returns the completed task result or an error if the task fails or timeout occurs.
func (c *Client) WaitForResult(ctx context.Context, taskID string, pollInterval, timeout time.Duration) (*TaskResult, error) {
	if taskID == "" {
		return nil, types.ErrEmptyTaskID
	}

	if pollInterval == 0 {
		pollInterval = types.DefaultPollInterval
	}
	if timeout == 0 {
		timeout = types.DefaultWaitTimeout
	}

	// Combine the original context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Perform initial check immediately
	result, err := c.GetTaskResult(ctxWithTimeout, taskID)
	if err != nil {
		return nil, err
	}

	for {
		switch result.Result.Status {
		case string(types.StatusDONE):
			return result, nil
		case string(types.StatusERROR):
			return nil, fmt.Errorf("task failed with status: %s", result.Result.Status)
		case string(types.StatusNEW), string(types.StatusPROCESSING), string(types.StatusRUNNING):
			// Continue polling
		default:
			return nil, fmt.Errorf("unknown task status: %s", result.Result.Status)
		}

		select {
		case <-ctxWithTimeout.Done():
			return nil, fmt.Errorf("timeout waiting for task %s: %w", taskID, ctxWithTimeout.Err())
		case <-ticker.C:
			result, err = c.GetTaskResult(ctxWithTimeout, taskID)
			if err != nil {
				return nil, err
			}
		}
	}
}

// DownloadTaskResultToFile downloads the actual result file for a completed task and saves it to disk.
// Uses endpoint: /data:download?response_file_id={taskId}
// Note: The task ID is used as the response_file_id parameter
func (c *Client) DownloadTaskResultToFile(ctx context.Context, responseFileID, outputPath string) error {
	if responseFileID == "" {
		return types.ErrEmptyTaskID
	}
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return fmt.Errorf("get auth token: %w", err)
	}

	// Construct proper URL for result download
	url := c.buildURL("data:download", map[string]string{"response_file_id": responseFileID})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/octet-stream")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer c.safeClose(resp.Body)

	// Check response status
	if resp.StatusCode != http.StatusOK {
		errorBody := c.readErrorBody(resp.Body)
		return fmt.Errorf("failed to download file (status %d): %s", resp.StatusCode, errorBody)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Clean up partial file on error
	var copyErr error
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && copyErr == nil {
			c.logger.Warn("failed to close output file", "error", closeErr)
		}
		if copyErr != nil {
			if removeErr := os.Remove(outputPath); removeErr != nil {
				c.logger.Warn("failed to remove partial file", "path", outputPath, "error", removeErr)
			}
		}
	}()

	// Stream data from response body to file
	_, copyErr = io.Copy(outFile, resp.Body)
	if copyErr != nil {
		return fmt.Errorf("failed to save file: %w", copyErr)
	}

	return nil
}

// DownloadTaskResult downloads the actual result file for a completed task and returns it as a byte slice.
// Warning: This method loads the entire file into memory. For large files, use DownloadTaskResultToFile instead.
// The maximum allowed file size is 10 MB to prevent memory issues.
func (c *Client) DownloadTaskResult(ctx context.Context, responseFileID string) ([]byte, error) {
	if responseFileID == "" {
		return nil, types.ErrEmptyTaskID
	}

	const maxFileSize = 10 * 1024 * 1024 // 10 MB limit

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	// Construct proper URL for result download
	url := c.buildURL("data:download", map[string]string{"response_file_id": responseFileID})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/octet-stream")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer c.safeClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errorBody := c.readErrorBody(resp.Body)
		return nil, fmt.Errorf("failed to get result (status %d): %s", resp.StatusCode, errorBody)
	}

	// Use limited reader to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxFileSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if int64(len(body)) > maxFileSize {
		return nil, fmt.Errorf("response too large (>%d bytes), use DownloadTaskResultToFile instead", maxFileSize)
	}

	return body, nil
}

// readErrorBody safely reads an error response body and logs any read errors.
// Returns the error body as a string, or "<unreadable>" if reading fails.
func (c *Client) readErrorBody(body io.ReadCloser) string {
	errorBody, readErr := io.ReadAll(body)
	if readErr != nil {
		c.logger.Warn("failed to read error response body", "error", readErr)
		return "<unreadable>"
	}
	return string(errorBody)
}

// safeClose safely closes an io.Closer and logs any errors.
func (c *Client) safeClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		c.logger.Warn("failed to close", "error", err)
	}
}

// applyDefaultsAndValidate sets default values and validates recognition options.
// This function mutates the input options struct to apply defaults where needed.
// Returns an error if validation fails after applying defaults.
func (c *Client) applyDefaultsAndValidate(opts *Options) error {
	if opts.Model == "" {
		opts.Model = ModelGeneral
	}
	if opts.AudioEncoding == "" {
		return fmt.Errorf("audio_encoding is required")
	}
	if opts.SampleRate <= 0 {
		return fmt.Errorf("sample_rate must be positive")
	}
	if opts.Language == "" {
		opts.Language = "ru-RU"
	}
	if opts.HypothesesCount < 0 || opts.HypothesesCount > types.MaxHypothesesCount {
		opts.HypothesesCount = 1
	}
	if opts.ChannelsCount < types.MinChannelsCount || opts.ChannelsCount > types.MaxChannelsCount {
		opts.ChannelsCount = 1
	}
	return nil
}
