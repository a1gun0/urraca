package engine

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestQueueEnqueueAndPop(t *testing.T) {
	cfg := Config{MaxJobs: 3}
	eng := &Engine{cfg: cfg, queue: []Job{}}

	jobs := []Job{
		{Stage: "low", Priority: 10},
		{Stage: "high", Priority: 100},
		{Stage: "low", Priority: 10}, // duplicate
	}
	for i, j := range jobs {
		ok := eng.enqueue(j)
		if i == 2 && ok {
			t.Errorf("expected duplicate to be rejected")
		}
	}

	if len(eng.queue) != 2 {
		t.Fatalf("unexpected queue length: %d", len(eng.queue))
	}

	first, ok := eng.popJob()
	if !ok || first.Stage != "high" {
		t.Fatalf("expected high priority job first, got %v", first)
	}
}

func TestJobTimeoutsAreSkipped(t *testing.T) {
	eng := &Engine{cfg: Config{}, queue: []Job{}}
	old := Job{Stage: "old", CreatedAt: time.Now().Add(-10 * time.Second), Timeout: 1 * time.Second}
	eng.enqueue(old)
	j, ok := eng.popJob()
	if ok {
		t.Errorf("expected expired job to be dropped, got %v", j)
	}
}

func TestSnapshotCopySafety(t *testing.T) {
	eng := &Engine{}
	eng.pushEvent(Event{Kind: EventLog, Message: "x", CreatedAt: time.Now()})
	eng.pushFinding(Finding{ID: "f"})
	eng.enqueue(Job{Stage: "s"})

	stage, finds, queue, evs := eng.Snapshot()
	if stage != "" || len(finds) != 1 || len(queue) != 1 || len(evs) != 1 {
		t.Fatalf("unexpected snapshot values")
	}

	// mutate originals and ensure snapshot stayed the same
	eng.events = nil
	if len(evs) != 1 {
		t.Errorf("snapshot aliased events slice")
	}
}

func TestEngineStartStopsWhenContextCancelled(t *testing.T) {
	cfg := DefaultConfig("http://example.com")
	eng := New(cfg, Definition{})
	ctx, cancel := context.WithCancel(context.Background())
	emitted := make([]Event, 0, 10)
	emit := func(ev Event) {
		emitted = append(emitted, ev)
	}
	go eng.Start(ctx, emit)
	// let it run a bit and then cancel
	time.Sleep(10 * time.Millisecond)
	cancel()
	// give goroutine time to notice
	time.Sleep(10 * time.Millisecond)
	if len(emitted) == 0 || !strings.Contains(emitted[0].Message, "pipeline") {
		t.Errorf("expected pipeline log event, got %v", emitted)
	}
}
