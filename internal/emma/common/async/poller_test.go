package async

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPoller_Poll_Success(t *testing.T) {
	callCount := 0
	statusChecker := func(ctx context.Context) (string, error) {
		callCount++
		if callCount >= 3 {
			return "completed", nil
		}
		return "in_progress", nil
	}

	config := PollerConfig{
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed", "success"},
		FailureStates: []string{"failed", "error"},
	}

	poller := NewPoller(config)
	ctx := context.Background()

	err := poller.Poll(ctx)
	if err != nil {
		t.Errorf("Expected successful poll, got error: %v", err)
	}

	if callCount < 3 {
		t.Errorf("Expected at least 3 status checks, got %d", callCount)
	}
}

func TestPoller_Poll_Timeout(t *testing.T) {
	statusChecker := func(ctx context.Context) (string, error) {
		return "in_progress", nil
	}

	config := PollerConfig{
		Timeout:       200 * time.Millisecond,
		PollInterval:  50 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed"},
		FailureStates: []string{"failed"},
	}

	poller := NewPoller(config)
	ctx := context.Background()

	err := poller.Poll(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if err != nil && err.Error() != "timeout waiting for operation to complete" {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestPoller_Poll_FailureState(t *testing.T) {
	callCount := 0
	statusChecker := func(ctx context.Context) (string, error) {
		callCount++
		if callCount >= 2 {
			return "failed", nil
		}
		return "in_progress", nil
	}

	config := PollerConfig{
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed"},
		FailureStates: []string{"failed", "error"},
	}

	poller := NewPoller(config)
	ctx := context.Background()

	err := poller.Poll(ctx)
	if err == nil {
		t.Error("Expected failure state error, got nil")
	}

	if err != nil && err.Error() != "operation failed with status: failed" {
		t.Errorf("Expected failure state error, got: %v", err)
	}
}

func TestPoller_Poll_Cancellation(t *testing.T) {
	statusChecker := func(ctx context.Context) (string, error) {
		return "in_progress", nil
	}

	config := PollerConfig{
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed"},
		FailureStates: []string{"failed"},
	}

	poller := NewPoller(config)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	err := poller.Poll(ctx)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestPoller_Poll_StatusCheckerError(t *testing.T) {
	expectedErr := errors.New("status check failed")
	statusChecker := func(ctx context.Context) (string, error) {
		return "", expectedErr
	}

	config := PollerConfig{
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed"},
		FailureStates: []string{"failed"},
	}

	poller := NewPoller(config)
	ctx := context.Background()

	err := poller.Poll(ctx)
	if err == nil {
		t.Error("Expected status checker error, got nil")
	}

	if err != nil && !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to wrap status checker error, got: %v", err)
	}
}

func TestPoller_Poll_MultipleTargetStates(t *testing.T) {
	callCount := 0
	statusChecker := func(ctx context.Context) (string, error) {
		callCount++
		if callCount >= 2 {
			return "success", nil
		}
		return "pending", nil
	}

	config := PollerConfig{
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed", "success", "done"},
		FailureStates: []string{"failed", "error"},
	}

	poller := NewPoller(config)
	ctx := context.Background()

	err := poller.Poll(ctx)
	if err != nil {
		t.Errorf("Expected successful poll, got error: %v", err)
	}
}

func TestPoller_Poll_ImmediateSuccess(t *testing.T) {
	statusChecker := func(ctx context.Context) (string, error) {
		return "completed", nil
	}

	config := PollerConfig{
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
		StatusChecker: statusChecker,
		TargetStates:  []string{"completed"},
		FailureStates: []string{"failed"},
	}

	poller := NewPoller(config)
	ctx := context.Background()

	start := time.Now()
	err := poller.Poll(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected successful poll, got error: %v", err)
	}

	// Should complete quickly (within first poll interval)
	if duration > 500*time.Millisecond {
		t.Errorf("Expected quick completion, took %v", duration)
	}
}
