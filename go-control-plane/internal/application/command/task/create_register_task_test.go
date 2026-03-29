package taskcommand

import (
	"context"
	"sync"
	"testing"
	"time"

	domaintask "go-control-plane/internal/domain/task"
	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeCommandRepository struct {
	mu      sync.Mutex
	created []domaintask.TaskRun
	events  []string
	updates []UpdateResult
}

func (f *fakeCommandRepository) Create(_ context.Context, run domaintask.TaskRun, _ string, _ string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, run)
	return nil
}

func (f *fakeCommandRepository) AppendEvents(_ context.Context, _ string, messages []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, messages...)
	return nil
}

func (f *fakeCommandRepository) UpdateResult(_ context.Context, result UpdateResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, result)
	return nil
}

type fakeWorkerClient struct {
	response workerport.RegisterResponse
	requests []workerport.RegisterRequest
}

func (f fakeWorkerClient) Register(_ context.Context, req workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return f.response, nil
}

func (f fakeWorkerClient) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}

func (f fakeWorkerClient) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (f fakeWorkerClient) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (f fakeWorkerClient) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeWorkerClient) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f fakeWorkerClient) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f fakeWorkerClient) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f fakeWorkerClient) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeWorkerClient) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeWorkerClient) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeWorkerClient) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestHandleCreatesAndCompletesTaskUsingWorkerResponse(t *testing.T) {
	repo := &fakeCommandRepository{}
	handler := NewHandler(repo, fakeWorkerClient{
		response: workerport.RegisterResponse{
			OK:           true,
			SuccessCount: 1,
			Logs:         []string{"started", "done"},
			CashierURLs:  []string{"https://example.com/pay"},
		},
	}, func() string { return "task_123" }, func() time.Time { return time.Unix(10, 0).UTC() }, "", "")

	result, err := handler.Handle(context.Background(), Command{Platform: "dummy", Count: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.TaskID != "task_123" {
		t.Fatalf("expected task id task_123, got %s", result.TaskID)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected task create, got %#v", repo.created)
	}
	if len(repo.events) != 2 {
		t.Fatalf("expected worker logs to be persisted, got %#v", repo.events)
	}
	if len(repo.updates) != 1 || repo.updates[0].Status != "done" {
		t.Fatalf("expected done update, got %#v", repo.updates)
	}
}

func TestHandlePassesCallbackMetadataToWorker(t *testing.T) {
	repo := &fakeCommandRepository{}
	worker := &capturingWorkerClient{
		response: workerport.RegisterResponse{
			OK: true,
		},
		called: make(chan struct{}),
	}
	handler := NewHandler(repo, worker, func() string { return "task_999" }, func() time.Time { return time.Unix(10, 0).UTC() }, "http://127.0.0.1:8080", "secret-token")

	_, err := handler.Handle(context.Background(), Command{Platform: "dummy", Count: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	select {
	case <-worker.called:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected worker to be invoked")
	}
	if worker.request.TaskID != "task_999" {
		t.Fatalf("expected task id to be forwarded, got %#v", worker.request)
	}
	if worker.request.CallbackBaseURL != "http://127.0.0.1:8080" {
		t.Fatalf("expected callback base url to be forwarded, got %#v", worker.request)
	}
	if worker.request.CallbackToken != "secret-token" {
		t.Fatalf("expected callback token to be forwarded, got %#v", worker.request)
	}
}

func TestHandleReturnsImmediatelyWhenCallbackModeEnabled(t *testing.T) {
	repo := &fakeCommandRepository{}
	worker := &blockingWorkerClient{
		release:  make(chan struct{}),
		response: workerport.RegisterResponse{OK: true, SuccessCount: 1},
	}
	handler := NewHandler(repo, worker, func() string { return "task_async" }, func() time.Time { return time.Unix(10, 0).UTC() }, "http://127.0.0.1:8080", "secret-token")

	done := make(chan Result, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := handler.Handle(context.Background(), Command{Platform: "dummy", Count: 1})
		if err != nil {
			errCh <- err
			return
		}
		done <- result
	}()

	select {
	case err := <-errCh:
		t.Fatalf("expected no error, got %v", err)
	case result := <-done:
		if result.TaskID != "task_async" {
			t.Fatalf("unexpected result: %#v", result)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected Handle to return before worker completes")
	}

	close(worker.release)
	time.Sleep(50 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.created) != 1 {
		t.Fatalf("expected task create, got %#v", repo.created)
	}
	if len(repo.updates) != 1 || repo.updates[0].Status != "done" {
		t.Fatalf("expected async worker completion update, got %#v", repo.updates)
	}
}

type capturingWorkerClient struct {
	request  workerport.RegisterRequest
	response workerport.RegisterResponse
	called   chan struct{}
}

func (f *capturingWorkerClient) Register(_ context.Context, req workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	f.request = req
	if f.called != nil {
		close(f.called)
	}
	return f.response, nil
}

func (f *capturingWorkerClient) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}

func (f *capturingWorkerClient) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (f *capturingWorkerClient) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (f *capturingWorkerClient) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *capturingWorkerClient) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f *capturingWorkerClient) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f *capturingWorkerClient) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f *capturingWorkerClient) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *capturingWorkerClient) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *capturingWorkerClient) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *capturingWorkerClient) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

type blockingWorkerClient struct {
	release  chan struct{}
	response workerport.RegisterResponse
}

func (f *blockingWorkerClient) Register(_ context.Context, req workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	<-f.release
	return f.response, nil
}

func (f *blockingWorkerClient) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}

func (f *blockingWorkerClient) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (f *blockingWorkerClient) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (f *blockingWorkerClient) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *blockingWorkerClient) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f *blockingWorkerClient) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f *blockingWorkerClient) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f *blockingWorkerClient) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *blockingWorkerClient) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *blockingWorkerClient) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *blockingWorkerClient) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}
