package agentruntime

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const DefaultMaxConcurrency = 4

// Coordinator executes single, parallel, and chained agent workflows.
type Coordinator struct {
	Runtime        Runtime
	MaxConcurrency int
}

// NewCoordinator creates a coordinator around a runtime.
func NewCoordinator(runtime Runtime) *Coordinator {
	return &Coordinator{Runtime: runtime, MaxConcurrency: DefaultMaxConcurrency}
}

// Invoke delegates to the underlying runtime.
func (c *Coordinator) Invoke(ctx context.Context, request Request) (*Result, error) {
	return c.runtime().Invoke(ctx, request)
}

// InvokeParallel runs requests concurrently with a bounded worker pool.
func (c *Coordinator) InvokeParallel(ctx context.Context, requests []Request) ([]*Result, error) {
	if len(requests) == 0 {
		return []*Result{}, nil
	}
	limit := c.MaxConcurrency
	if limit <= 0 {
		limit = DefaultMaxConcurrency
	}
	if limit > len(requests) {
		limit = len(requests)
	}
	results := make([]*Result, len(requests))
	var firstErr error
	var mu sync.Mutex
	next := 0
	worker := func() {
		for {
			mu.Lock()
			idx := next
			next++
			mu.Unlock()
			if idx >= len(requests) {
				return
			}
			result, err := c.runtime().Invoke(ctx, requests[idx])
			mu.Lock()
			results[idx] = result
			if err != nil && firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}
	}
	var wg sync.WaitGroup
	wg.Add(limit)
	for range limit {
		go func() { defer wg.Done(); worker() }()
	}
	wg.Wait()
	return results, firstErr
}

// InvokeChain runs requests sequentially. Each request task may contain {previous}.
func (c *Coordinator) InvokeChain(ctx context.Context, requests []Request) ([]*Result, error) {
	results := make([]*Result, 0, len(requests))
	previous := ""
	for i, request := range requests {
		request.Task = strings.ReplaceAll(request.Task, "{previous}", previous)
		result, err := c.runtime().Invoke(ctx, request)
		if result != nil {
			results = append(results, result)
		}
		if err != nil {
			return results, err
		}
		if result == nil {
			return results, fmt.Errorf("chain step %d returned nil result", i+1)
		}
		if result.Status != StatusSuccess {
			return results, nil
		}
		previous = result.Output
	}
	return results, nil
}

func (c *Coordinator) runtime() Runtime {
	if c.Runtime != nil {
		return c.Runtime
	}
	return NewPiWorker()
}
