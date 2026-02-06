package async

import "time"

// Default timeout values for async operations
const (
	// DefaultTimeout is the default timeout for async operations
	DefaultTimeout = 10 * time.Minute

	// DefaultPollInterval is the default interval between status checks
	DefaultPollInterval = 5 * time.Second

	// ShortTimeout is a shorter timeout for quick operations
	ShortTimeout = 2 * time.Minute

	// LongTimeout is a longer timeout for slow operations
	LongTimeout = 30 * time.Minute

	// MinPollInterval is the minimum recommended poll interval
	MinPollInterval = 1 * time.Second

	// MaxPollInterval is the maximum recommended poll interval
	MaxPollInterval = 30 * time.Second
)

// TimeoutConfig provides timeout configuration helpers
type TimeoutConfig struct {
	Create time.Duration
	Read   time.Duration
	Update time.Duration
	Delete time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Create: DefaultTimeout,
		Read:   ShortTimeout,
		Update: DefaultTimeout,
		Delete: DefaultTimeout,
	}
}

// WithCreate returns a new TimeoutConfig with custom create timeout
func (tc TimeoutConfig) WithCreate(timeout time.Duration) TimeoutConfig {
	tc.Create = timeout
	return tc
}

// WithRead returns a new TimeoutConfig with custom read timeout
func (tc TimeoutConfig) WithRead(timeout time.Duration) TimeoutConfig {
	tc.Read = timeout
	return tc
}

// WithUpdate returns a new TimeoutConfig with custom update timeout
func (tc TimeoutConfig) WithUpdate(timeout time.Duration) TimeoutConfig {
	tc.Update = timeout
	return tc
}

// WithDelete returns a new TimeoutConfig with custom delete timeout
func (tc TimeoutConfig) WithDelete(timeout time.Duration) TimeoutConfig {
	tc.Delete = timeout
	return tc
}

// CalculatePollInterval calculates an appropriate poll interval based on timeout
// Uses a heuristic: poll interval should be roughly 1/20th of timeout, bounded by min/max
func CalculatePollInterval(timeout time.Duration) time.Duration {
	interval := timeout / 20
	
	if interval < MinPollInterval {
		return MinPollInterval
	}
	
	if interval > MaxPollInterval {
		return MaxPollInterval
	}
	
	return interval
}
