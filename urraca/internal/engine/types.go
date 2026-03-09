package engine

import "time"

type Finding struct {
	ID         string
	Target     string
	Module     string
	Category   string
	Subtype    string
	Value      string
	URL        string
	Status     int
	Confidence int
	Evidence   string
	Timestamp  time.Time
	Severity   string
}

type Job struct {
	ID        string
	Stage     string
	Target    string
	Priority  int
	CreatedAt time.Time
	Timeout   time.Duration
	Input     map[string]string
}

type EventKind string

const (
	EventLog     EventKind = "log"
	EventFinding EventKind = "finding"
	EventJob     EventKind = "job"
	EventStage   EventKind = "stage"
)

type Event struct {
	Kind      EventKind
	Message   string
	Finding   *Finding
	Job       *Job
	CreatedAt time.Time
}

type Config struct {
	Target         string
	Timeout        time.Duration
	StageDelay     time.Duration
	MaxDepth       int
	MaxJobs        int
	FollowRedirect bool
}
