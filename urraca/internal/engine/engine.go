package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/urraca/internal/pipeline"
)

type Engine struct {
	cfg       Config
	pipeline  pipeline.Definition
	mu        sync.Mutex
	events    []Event
	findings  []Finding
	queue     []Job
	running   bool
	stageName string // protected by mu
}

func DefaultConfig(target string) Config {
	return Config{
		Target:         target,
		Timeout:        8 * time.Second,
		StageDelay:     600 * time.Millisecond,
		MaxDepth:       3,
		MaxJobs:        256,
		FollowRedirect: true,
	}
}

func New(cfg Config, def pipeline.Definition) *Engine {
	if def.Stages == nil {
		def = pipeline.Default()
	}
	return &Engine{
		cfg:      cfg,
		pipeline: def,
		queue: []Job{
			{
				ID:        "seed-bootstrap",
				Stage:     "bootstrap",
				Target:    cfg.Target,
				Priority:  100,
				CreatedAt: time.Now(),
				Timeout:   cfg.Timeout,
				Input:     map[string]string{"target": cfg.Target},
			},
		},
		events:   make([]Event, 0, 64),
		findings: make([]Finding, 0, 32),
	}
}

func (e *Engine) Start(ctx context.Context, emit func(Event)) {
	// make sure we don't start twice
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			emit(Event{Kind: EventLog, Message: fmt.Sprintf("panic en el engine: %v", r), CreatedAt: time.Now()})
		}
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			emit(Event{Kind: EventLog, Message: "pipeline detenido", CreatedAt: time.Now()})
			return
		default:
		}

		job, ok := e.popJob()
		if !ok {
			emit(Event{Kind: EventLog, Message: "pipeline completo", CreatedAt: time.Now()})
			return
		}

		// drop job if it has expired
		if job.Timeout > 0 && time.Since(job.CreatedAt) > job.Timeout {
			emit(Event{Kind: EventLog, Message: "job descartado por timeout: " + job.Stage, Job: &job, CreatedAt: time.Now()})
			continue
		}

		// record current stage under lock
		e.mu.Lock()
		e.stageName = job.Stage
		// keep track of stage until next iteration
		e.mu.Unlock()

		emit(Event{Kind: EventStage, Message: fmt.Sprintf("stage: %s", job.Stage), Job: &job, CreatedAt: time.Now()})

		handler, found := e.pipeline.Stages[job.Stage]
		if !found {
			emit(Event{Kind: EventLog, Message: "stage no definido: " + job.Stage, CreatedAt: time.Now()})
			continue
		}

		results := handler(job, e.cfg)

		for _, ev := range results.Events {
			e.pushEvent(ev)
			emit(ev)
		}
		for _, f := range results.Findings {
			e.pushFinding(f)
			ev := Event{Kind: EventFinding, Message: f.Value, Finding: &f, CreatedAt: time.Now()}
			e.pushEvent(ev)
			emit(ev)
		}
		for _, next := range results.NextJobs {
			if e.enqueue(next) {
				ev := Event{Kind: EventJob, Message: next.Stage, Job: &next, CreatedAt: time.Now()}
				e.pushEvent(ev)
				emit(ev)
			}
		}

		time.Sleep(e.cfg.StageDelay)
	}
}

func (e *Engine) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *Engine) Snapshot() (stage string, findings []Finding, queue []Job, events []Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	fc := append([]Finding(nil), e.findings...)
	qc := append([]Job(nil), e.queue...)
	ec := append([]Event(nil), e.events...)
	return e.stageName, fc, qc, ec
}

func (e *Engine) popJob() (Job, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.queue) == 0 {
		return Job{}, false
	}
	// pick job with highest priority (larger number means higher priority)
	bestIdx := 0
	for i := 1; i < len(e.queue); i++ {
		if e.queue[i].Priority > e.queue[bestIdx].Priority {
			bestIdx = i
		}
	}
	j := e.queue[bestIdx]
	// remove from slice
	e.queue = append(e.queue[:bestIdx], e.queue[bestIdx+1:]...)
	return j, true
}

func (e *Engine) enqueue(job Job) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if job.Input == nil {
		job.Input = make(map[string]string)
	}

	for _, existing := range e.queue {
		if existing.Stage == job.Stage && existing.Target == job.Target {
			if existing.Input["url"] == job.Input["url"] && existing.Input["path"] == job.Input["path"] {
				return false // duplicate
			}
		}
	}
	if len(e.queue) >= e.cfg.MaxJobs {
		// queue full, drop silently
		return false
	}
	e.queue = append(e.queue, job)
	return true
}

func (e *Engine) pushEvent(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
	if len(e.events) > 500 {
		e.events = e.events[len(e.events)-500:]
	}
}

func (e *Engine) pushFinding(f Finding) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.findings = append(e.findings, f)
	if len(e.findings) > 1000 {
		e.findings = e.findings[len(e.findings)-1000:]
	}
}
