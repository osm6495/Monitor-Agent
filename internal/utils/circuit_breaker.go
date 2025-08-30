package utils

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitBreaker states
const (
	StateClosed   = "closed"   // Normal operation
	StateOpen     = "open"     // Circuit is open, requests fail fast
	StateHalfOpen = "halfopen" // Testing if service is back
)

// CircuitBreakerError represents a circuit breaker error
var CircuitBreakerError = errors.New("circuit breaker is open")

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold   int           // Number of failures before opening circuit
	RecoveryTimeout    time.Duration // Time to wait before attempting recovery
	SuccessThreshold   int           // Number of successes needed to close circuit
	Timeout            time.Duration // Timeout for individual requests
	MaxConcurrentCalls int           // Maximum concurrent calls when half-open
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config    CircuitBreakerConfig
	state     string
	failures  int
	successes int
	lastError time.Time
	mutex     sync.RWMutex
	callCount int
	lastCall  time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.canExecute() {
		return CircuitBreakerError
	}

	cb.mutex.Lock()
	cb.callCount++
	cb.lastCall = time.Now()
	cb.mutex.Unlock()

	// Execute the function with timeout
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		cb.recordResult(err)
		return err
	case <-ctx.Done():
		cb.recordResult(ctx.Err())
		return ctx.Err()
	case <-time.After(cb.config.Timeout):
		cb.recordResult(errors.New("timeout"))
		return errors.New("timeout")
	}
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.lastError) >= cb.config.RecoveryTimeout {
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.mutex.Unlock()
			cb.mutex.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		// Limit concurrent calls in half-open state
		return cb.callCount < cb.config.MaxConcurrentCalls
	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		cb.failures++
		cb.lastError = time.Now()
		cb.successes = 0

		// Open circuit if failure threshold reached
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	} else {
		cb.successes++
		cb.failures = 0

		// Close circuit if success threshold reached
		if cb.state == StateHalfOpen && cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() string {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"state":     cb.state,
		"failures":  cb.failures,
		"successes": cb.successes,
		"lastError": cb.lastError,
		"callCount": cb.callCount,
		"lastCall":  cb.lastCall,
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.callCount = 0
}
