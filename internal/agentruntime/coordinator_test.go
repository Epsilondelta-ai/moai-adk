package agentruntime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

type fakeRuntime struct {
	mu       sync.Mutex
	requests []Request
}

func (f *fakeRuntime) Invoke(_ context.Context, request Request) (*Result, error) {
	f.mu.Lock()
	f.requests = append(f.requests, request)
	f.mu.Unlock()
	return &Result{Agent: request.Agent, Status: StatusSuccess, Output: "out:" + request.Task}, nil
}

func TestCoordinatorInvokeChainReplacesPrevious(t *testing.T) {
	rt := &fakeRuntime{}
	coord := NewCoordinator(rt)
	results, err := coord.InvokeChain(context.Background(), []Request{
		{Agent: "a", Task: "first"},
		{Agent: "b", Task: "second {previous}"},
	})
	if err != nil {
		t.Fatalf("InvokeChain() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results len = %d", len(results))
	}
	if !strings.Contains(rt.requests[1].Task, "out:first") {
		t.Fatalf("previous not injected: %#v", rt.requests[1])
	}
}

func TestCoordinatorInvokeParallelKeepsOrder(t *testing.T) {
	rt := &fakeRuntime{}
	coord := NewCoordinator(rt)
	coord.MaxConcurrency = 2
	requests := []Request{{Agent: "a", Task: "1"}, {Agent: "b", Task: "2"}, {Agent: "c", Task: "3"}}
	results, err := coord.InvokeParallel(context.Background(), requests)
	if err != nil {
		t.Fatalf("InvokeParallel() error = %v", err)
	}
	for i, result := range results {
		want := fmt.Sprintf("out:%d", i+1)
		if result.Output != want {
			t.Fatalf("result[%d].Output = %q, want %q", i, result.Output, want)
		}
	}
}
