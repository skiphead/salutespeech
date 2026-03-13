package utils

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	OnRetry         func(attempt int, err error, nextInterval time.Duration)
}

// DefaultRetryConfig returns default retry config
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

// RetryWithBackoff executes fn with exponential backoff retry logic.
//
// Retry semantics:
//   - MaxRetries=3 means: 1 initial call + up to 3 retries = 4 total attempts max
//   - Backoff: interval starts at InitialInterval, multiplied by Multiplier each retry
//   - Interval capped at MaxInterval
//   - Optional jitter can be added via WithJitter helper
//
// Context cancellation is checked between retries (not during fn execution).
//
// Example:
//
//	cfg := utils.DefaultRetryConfig()
//	cfg.OnRetry = func(attempt int, err error, next time.Duration) {
//	    log.Printf("retry %d: %v, next in %v", attempt, err, next)
//	}
//
//	err := RetryWithBackoff(ctx, myFunc, cfg, func(err error) bool {
//	    return errors.Is(err, ErrTemporary)
//	})
func RetryWithBackoff(
	ctx context.Context,
	fn func() error,
	cfg RetryConfig,
	isRetryable func(error) bool,
) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("retry config: %w", err)
	}
	if isRetryable == nil {
		isRetryable = func(error) bool { return true } // retry all by default
	}

	var lastErr error
	interval := cfg.InitialInterval

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Не ретраим, если ошибка не подходит под критерий
		if !isRetryable(err) {
			return err
		}

		// Последняя попытка исчерпана
		if attempt == cfg.MaxRetries {
			break
		}

		// Уведомление о ретрае (для логирования/метрик)
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt+1, err, interval)
		}

		// Ожидание с поддержкой отмены
		select {
		case <-time.After(interval):
			// Применяем джиттер (опционально, ~±25%)
			jitter := time.Duration(float64(interval) * 0.25 * (rand.Float64()*2 - 1))
			time.Sleep(jitter)
		case <-ctx.Done():
			return ctx.Err()
		}

		// Экспоненциальный рост с потолком
		interval = time.Duration(float64(interval) * cfg.Multiplier)
		if interval > cfg.MaxInterval {
			interval = cfg.MaxInterval
		}
	}

	return lastErr
}

func (c RetryConfig) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries must be >= 0, got %d", c.MaxRetries)
	}
	if c.InitialInterval <= 0 {
		return fmt.Errorf("InitialInterval must be > 0")
	}
	if c.MaxInterval < c.InitialInterval {
		return fmt.Errorf("MaxInterval (%v) < InitialInterval (%v)",
			c.MaxInterval, c.InitialInterval)
	}
	if c.Multiplier <= 1.0 {
		return fmt.Errorf("Multiplier must be > 1.0 for exponential backoff")
	}
	return nil
}
