package async

import (
	"testing"
	"time"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	if config.Create != DefaultTimeout {
		t.Errorf("Expected Create timeout %v, got %v", DefaultTimeout, config.Create)
	}

	if config.Read != ShortTimeout {
		t.Errorf("Expected Read timeout %v, got %v", ShortTimeout, config.Read)
	}

	if config.Update != DefaultTimeout {
		t.Errorf("Expected Update timeout %v, got %v", DefaultTimeout, config.Update)
	}

	if config.Delete != DefaultTimeout {
		t.Errorf("Expected Delete timeout %v, got %v", DefaultTimeout, config.Delete)
	}
}

func TestTimeoutConfig_WithCreate(t *testing.T) {
	config := DefaultTimeoutConfig()
	customTimeout := 15 * time.Minute

	newConfig := config.WithCreate(customTimeout)

	if newConfig.Create != customTimeout {
		t.Errorf("Expected Create timeout %v, got %v", customTimeout, newConfig.Create)
	}

	// Verify other timeouts unchanged
	if newConfig.Read != config.Read {
		t.Error("Read timeout should not change")
	}
}

func TestTimeoutConfig_WithRead(t *testing.T) {
	config := DefaultTimeoutConfig()
	customTimeout := 1 * time.Minute

	newConfig := config.WithRead(customTimeout)

	if newConfig.Read != customTimeout {
		t.Errorf("Expected Read timeout %v, got %v", customTimeout, newConfig.Read)
	}
}

func TestTimeoutConfig_WithUpdate(t *testing.T) {
	config := DefaultTimeoutConfig()
	customTimeout := 20 * time.Minute

	newConfig := config.WithUpdate(customTimeout)

	if newConfig.Update != customTimeout {
		t.Errorf("Expected Update timeout %v, got %v", customTimeout, newConfig.Update)
	}
}

func TestTimeoutConfig_WithDelete(t *testing.T) {
	config := DefaultTimeoutConfig()
	customTimeout := 5 * time.Minute

	newConfig := config.WithDelete(customTimeout)

	if newConfig.Delete != customTimeout {
		t.Errorf("Expected Delete timeout %v, got %v", customTimeout, newConfig.Delete)
	}
}

func TestTimeoutConfig_Chaining(t *testing.T) {
	config := DefaultTimeoutConfig().
		WithCreate(15 * time.Minute).
		WithRead(1 * time.Minute).
		WithUpdate(20 * time.Minute).
		WithDelete(5 * time.Minute)

	if config.Create != 15*time.Minute {
		t.Errorf("Expected Create timeout 15m, got %v", config.Create)
	}

	if config.Read != 1*time.Minute {
		t.Errorf("Expected Read timeout 1m, got %v", config.Read)
	}

	if config.Update != 20*time.Minute {
		t.Errorf("Expected Update timeout 20m, got %v", config.Update)
	}

	if config.Delete != 5*time.Minute {
		t.Errorf("Expected Delete timeout 5m, got %v", config.Delete)
	}
}

func TestCalculatePollInterval(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{
			name:     "short timeout uses minimum",
			timeout:  10 * time.Second,
			expected: MinPollInterval,
		},
		{
			name:     "medium timeout calculates interval",
			timeout:  2 * time.Minute,
			expected: 6 * time.Second,
		},
		{
			name:     "default timeout calculates interval",
			timeout:  DefaultTimeout,
			expected: 30 * time.Second,
		},
		{
			name:     "long timeout uses maximum",
			timeout:  20 * time.Minute,
			expected: MaxPollInterval,
		},
		{
			name:     "very long timeout uses maximum",
			timeout:  1 * time.Hour,
			expected: MaxPollInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePollInterval(tt.timeout)
			if result != tt.expected {
				t.Errorf("Expected poll interval %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculatePollInterval_Bounds(t *testing.T) {
	// Test minimum bound
	veryShortTimeout := 5 * time.Second
	interval := CalculatePollInterval(veryShortTimeout)
	if interval < MinPollInterval {
		t.Errorf("Poll interval %v should not be less than minimum %v", interval, MinPollInterval)
	}

	// Test maximum bound
	veryLongTimeout := 2 * time.Hour
	interval = CalculatePollInterval(veryLongTimeout)
	if interval > MaxPollInterval {
		t.Errorf("Poll interval %v should not be greater than maximum %v", interval, MaxPollInterval)
	}
}
