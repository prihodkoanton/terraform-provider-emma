package async

import (
	"context"
	"fmt"
	"time"
)

// PollerConfig configures async operation polling
type PollerConfig struct {
	Timeout       time.Duration
	PollInterval  time.Duration
	StatusChecker func(ctx context.Context) (string, error)
	TargetStates  []string
	FailureStates []string
}

// Poller handles async operation polling
type Poller struct {
	config PollerConfig
}

// NewPoller creates a new Poller with the given configuration
func NewPoller(config PollerConfig) *Poller {
	return &Poller{config: config}
}

// Poll waits for operation to reach target state
func (p *Poller) Poll(ctx context.Context) error {
	deadline := time.Now().Add(p.config.Timeout)
	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for operation to complete")
			}

			status, err := p.config.StatusChecker(ctx)
			if err != nil {
				return fmt.Errorf("error checking status: %w", err)
			}

			// Check if reached target state
			for _, target := range p.config.TargetStates {
				if status == target {
					return nil
				}
			}

			// Check if reached failure state
			for _, failure := range p.config.FailureStates {
				if status == failure {
					return fmt.Errorf("operation failed with status: %s", status)
				}
			}
		}
	}
}
