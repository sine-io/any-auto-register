package task

import "time"

type WorkerEventType string

const (
	WorkerEventStarted   WorkerEventType = "started"
	WorkerEventProgress  WorkerEventType = "progress"
	WorkerEventLog       WorkerEventType = "log"
	WorkerEventSucceeded WorkerEventType = "succeeded"
	WorkerEventFailed    WorkerEventType = "failed"
)

type WorkerEvent struct {
	TaskID          string
	Type            WorkerEventType
	Message         string
	ProgressCurrent int
	ProgressTotal   int
	SuccessCount    int
	ErrorCount      int
	ErrorSummary    string
	Errors          []string
	CashierURLs     []string
	OccurredAt      time.Time
}
