package async

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Client handles async recognition
type Client struct {
	httpClient *http.Client
	baseURL    string
	resultURL  string
	tokenMgr   *client.TokenManager
	logger     types.Logger
}

// Config represents recognition client configuration
type Config struct {
	BaseURL       string
	ResultURL     string
	Timeout       time.Duration
	AllowInsecure bool
	Logger        types.Logger
}

// NewClient creates new recognition client
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
		baseURL = types.DefaultRecognitionURL
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

// CreateTask creates async recognition task
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
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
	defer resp.Body.Close()

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

// GetTaskResult gets recognition task result
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	if taskID == "" {
		return nil, types.ErrEmptyTaskID
	}

	authHeader, err := c.tokenMgr.GetTokenWithHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	url := fmt.Sprintf("%s?id=%s", c.resultURL, taskID)

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
	defer resp.Body.Close()

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

	// Map status to unified type
	switch result.Status {
	case "PROCESSING":
		result.UnifiedStatus = types.StatusPROCESSING
	case "DONE":
		result.UnifiedStatus = types.StatusDONE
	case "ERROR":
		result.UnifiedStatus = types.StatusERROR
	case "NEW":
		result.UnifiedStatus = types.StatusNEW
	default:
		result.UnifiedStatus = types.TaskStatus(result.Status)
	}

	return &result, nil
}

// WaitForResult waits for task completion
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

			switch result.UnifiedStatus {
			case types.StatusDONE:
				return result, nil
			case types.StatusERROR:
				msg := "unknown error"
				if result.ErrorMessage != nil {
					msg = *result.ErrorMessage
				}
				return nil, fmt.Errorf("task failed [%s]: %s",
					func() string {
						if result.ErrorCode != nil {
							return *result.ErrorCode
						}
						return "ERROR"
					}(),
					msg)
			case types.StatusNEW, types.StatusPROCESSING:
				continue
			default:
				return nil, fmt.Errorf("unknown task status: %s", result.UnifiedStatus)
			}
		}
	}
}

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
