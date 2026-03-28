package task

import (
	"fmt"
	"time"
)

type TaskRun struct {
	ID              string
	Platform        string
	Status          string
	ProgressCurrent int
	ProgressTotal   int
	SuccessCount    int
	ErrorCount      int
	ErrorSummary    string
	Errors          []string
	CashierURLs     []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (t TaskRun) Progress() string {
	return formatProgress(t.ProgressCurrent, t.ProgressTotal)
}

func formatProgress(current, total int) string {
	return fmt.Sprintf("%d/%d", current, total)
}
