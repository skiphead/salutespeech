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
	"os"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Client handles asynchronous speech recognition operations.
// It manages task creation, result retrieval, and status polling
// for long-running recognition jobs through the SaluteSpeech API.
type Client struct {
	httpClient *http.Client
	baseURL    string // URL for creating recognition tasks
	resultURL  string // URL for retrieving task results
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Config represents the configuration options for creating a new async recognition client.
// It allows customization of API endpoints, timeout behavior, TLS settings, and logging.
type Config struct {
	BaseURL       string        // URL for task creation (defaults to DefaultRecognitionURL)
	ResultURL     string        // URL for result retrieval (defaults to DefaultResultURL)
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

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = types.DefaultBaseURL
	}

	resultURL := cfg.ResultURL
	if resultURL == "" {
		resultURL = types.DefaultResultURL
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
		resultURL:  resultURL,
		tokenMgr:   tokenMgr,
		logger:     logger,
	}, nil
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
	if err := c.validateOptions(req.Options); err != nil {
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

	url := fmt.Sprintf("%s%s", c.baseURL, "speech:async_recognize")

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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var recognitionResp Response
		if err := json.Unmarshal(respBody, &recognitionResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if recognitionResp.Result.ID == "" {
			return nil, types.ErrEmptyTaskID
		}
		return &recognitionResp, nil

	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s", types.ErrBadRequest, string(respBody))
	case http.StatusUnauthorized:
		_ = c.tokenMgr.ForceRefresh(ctx)
		return nil, fmt.Errorf("%w: %s", types.ErrUnauthorized, string(respBody))
	case http.StatusRequestEntityTooLarge:
		return nil, fmt.Errorf("payload too large (413): %s", string(respBody))
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", types.ErrTooManyRequests, string(respBody))
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", types.ErrServerError, string(respBody))
	default:
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
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
	//task:get
	url := fmt.Sprintf("%stask:get?id=%s", c.baseURL, taskID)

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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get result (status %d): %s", resp.StatusCode, string(body))
	}

	var result TaskResult
	if err := json.Unmarshal(body, &result); err != nil {
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
			return nil, fmt.Errorf("timeout waiting for task %s: %w", taskID, ctxWithTimeout.Err())

		case <-ticker.C:
			result, err := c.GetTaskResult(ctxWithTimeout, taskID)
			if err != nil {
				return nil, err
			}

			switch result.Result.Status {
			case string(types.StatusDONE):
				return result, nil
			case string(types.StatusERROR):
				return nil, fmt.Errorf("task failed")

			case string(types.StatusNEW), string(types.StatusPROCESSING):
				continue
			default:
				return nil, fmt.Errorf("unknown task status: %s", result.Result.Status)
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

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return fmt.Errorf("get auth token: %w", err)
	}

	// Construct proper URL for result download
	// The task ID becomes the response_file_id parameter
	url := fmt.Sprintf("%sdata:download?response_file_id=%s", c.baseURL, responseFileID)

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
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		// Пытаемся прочитать тело ошибки для диагностики
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download file (status %d): %s", resp.StatusCode, string(errorBody))
	}

	// Создаем выходной файл
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Копируем данные из response body в файл
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (c *Client) DownloadTaskResult(ctx context.Context, responseFileID string) ([]byte, error) {
	if responseFileID == "" {
		return nil, types.ErrEmptyTaskID
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	// Construct proper URL for result download
	// The task ID becomes the response_file_id parameter
	url := fmt.Sprintf("%sdata:download?response_file_id=%s", c.baseURL, responseFileID)

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
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			c.logger.Warn("close response body error:", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get result (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil

}

// validateOptions performs validation of recognition options and applies default values.
// It checks required fields like audio encoding and sample rate, and sets defaults
// for optional parameters like model, language, hypotheses count, and channels count.
func (c *Client) validateOptions(opts *Options) error {
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
