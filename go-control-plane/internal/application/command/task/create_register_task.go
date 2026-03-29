package taskcommand

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domaintask "go-control-plane/internal/domain/task"
	workerport "go-control-plane/internal/ports/outbound/worker"
)

type Command struct {
	Platform             string         `json:"platform"`
	Email                string         `json:"email,omitempty"`
	Password             string         `json:"password,omitempty"`
	Count                int            `json:"count"`
	Concurrency          int            `json:"concurrency,omitempty"`
	RegisterDelaySeconds float64        `json:"register_delay_seconds,omitempty"`
	Proxy                string         `json:"proxy,omitempty"`
	ExecutorType         string         `json:"executor_type,omitempty"`
	CaptchaSolver        string         `json:"captcha_solver,omitempty"`
	Extra                map[string]any `json:"extra,omitempty"`
}

type Result struct {
	TaskID string `json:"task_id"`
}

type UpdateResult struct {
	TaskID          string
	Status          string
	ProgressCurrent int
	ProgressTotal   int
	SuccessCount    int
	ErrorCount      int
	ErrorSummary    string
	ErrorsJSON      string
	CashierURLsJSON string
	UpdatedAt       time.Time
}

type Repository interface {
	Create(ctx context.Context, run domaintask.TaskRun, requestJSON string, errorsJSON string, cashierURLsJSON string) error
	AppendEvents(ctx context.Context, taskID string, messages []string) error
	UpdateResult(ctx context.Context, result UpdateResult) error
}

type Handler struct {
	repo            Repository
	worker          workerport.Client
	newID           func() string
	now             func() time.Time
	callbackBaseURL string
	callbackToken   string
}

func NewHandler(repo Repository, worker workerport.Client, newID func() string, now func() time.Time, callbackBaseURL string, callbackToken string) Handler {
	if newID == nil {
		newID = func() string {
			return fmt.Sprintf("task_%d", time.Now().UnixMilli())
		}
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return Handler{
		repo:            repo,
		worker:          worker,
		newID:           newID,
		now:             now,
		callbackBaseURL: callbackBaseURL,
		callbackToken:   callbackToken,
	}
}

func (h Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	taskID := h.newID()
	if taskID == "" {
		taskID = fmt.Sprintf("task_%d", time.Now().UnixMilli())
	}
	now := h.now()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if cmd.Count <= 0 {
		cmd.Count = 1
	}

	requestJSON, _ := json.Marshal(cmd)
	run := domaintask.TaskRun{
		ID:              taskID,
		Platform:        cmd.Platform,
		Status:          "pending",
		ProgressCurrent: 0,
		ProgressTotal:   cmd.Count,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := h.repo.Create(ctx, run, string(requestJSON), "[]", "[]"); err != nil {
		return Result{}, err
	}

	if h.callbackBaseURL != "" {
		go h.runWorkerRegister(context.Background(), taskID, cmd)
		return Result{TaskID: taskID}, nil
	}

	if err := h.runWorkerRegister(ctx, taskID, cmd); err != nil {
		return Result{}, err
	}
	return Result{TaskID: taskID}, nil
}

func (h Handler) runWorkerRegister(ctx context.Context, taskID string, cmd Command) error {
	resp, err := h.worker.Register(ctx, workerport.RegisterRequest{
		TaskID:               taskID,
		CallbackBaseURL:      h.callbackBaseURL,
		CallbackToken:        h.callbackToken,
		Platform:             cmd.Platform,
		Email:                cmd.Email,
		Password:             cmd.Password,
		Count:                cmd.Count,
		Concurrency:          cmd.Concurrency,
		RegisterDelaySeconds: cmd.RegisterDelaySeconds,
		Proxy:                cmd.Proxy,
		ExecutorType:         cmd.ExecutorType,
		CaptchaSolver:        cmd.CaptchaSolver,
		Extra:                cmd.Extra,
	})
	if len(resp.Logs) > 0 && h.callbackBaseURL == "" {
		_ = h.repo.AppendEvents(ctx, taskID, resp.Logs)
	}

	update := UpdateResult{
		TaskID:          taskID,
		Status:          "done",
		ProgressCurrent: cmd.Count,
		ProgressTotal:   cmd.Count,
		SuccessCount:    resp.SuccessCount,
		ErrorCount:      resp.ErrorCount,
		ErrorSummary:    resp.Error,
		ErrorsJSON:      mustJSON(resp.Errors),
		CashierURLsJSON: mustJSON(resp.CashierURLs),
		UpdatedAt:       h.now(),
	}
	if err != nil || !resp.OK {
		update.Status = "failed"
		if err != nil {
			update.ErrorSummary = err.Error()
			update.ErrorsJSON = mustJSON([]string{err.Error()})
		} else if resp.Error != "" {
			update.ErrorSummary = resp.Error
		}
	}
	if err := h.repo.UpdateResult(ctx, update); err != nil {
		return err
	}
	return nil
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}
