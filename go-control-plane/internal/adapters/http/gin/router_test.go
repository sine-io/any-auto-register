package gingateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	viperconfig "go-control-plane/internal/adapters/config/viper"
	zerologadapter "go-control-plane/internal/adapters/log/zerolog"
	accountcommand "go-control-plane/internal/application/command/account"
	actioncommand "go-control-plane/internal/application/command/action"
	taskcommand "go-control-plane/internal/application/command/task"
	accountquery "go-control-plane/internal/application/query/account"
	platformquery "go-control-plane/internal/application/query/platform"
	taskquery "go-control-plane/internal/application/query/task"
)

type fakeTaskQueryHandler struct{}

func (fakeTaskQueryHandler) Handle(context.Context, taskquery.Query) (taskquery.Result, error) {
	return taskquery.Result{
		Total: 1,
		Page:  1,
		Items: []taskquery.TaskItem{{ID: "task_1", Platform: "dummy", Status: "done", Progress: "1/1", Success: 1}},
	}, nil
}

type fakePlatformQueryHandler struct{}

func (fakePlatformQueryHandler) Handle(context.Context) (platformquery.Result, error) {
	return platformquery.Result{
		Items: []platformquery.PlatformItem{
			{Name: "trae", DisplayName: "Trae.ai", SupportedExecutors: []string{"protocol", "headed"}},
		},
	}, nil
}

type fakeGetTaskHandler struct{}

func (fakeGetTaskHandler) Handle(context.Context, taskquery.GetTaskQuery) (taskquery.TaskItem, error) {
	return taskquery.TaskItem{
		ID:          "task_1",
		Platform:    "dummy",
		Status:      "done",
		Progress:    "1/1",
		Success:     1,
		CashierURLs: []string{"https://example.com/pay"},
	}, nil
}

type fakeListTaskLogsHandler struct{}

func (fakeListTaskLogsHandler) Handle(context.Context, taskquery.ListLogsQuery) (taskquery.ListLogsResult, error) {
	return taskquery.ListLogsResult{
		Total: 1,
		Page:  1,
		Items: []taskquery.TaskLogItem{{ID: 1, Platform: "dummy", Email: "user@example.com", Status: "success", CreatedAt: time.Unix(1, 0).UTC()}},
	}, nil
}

type fakeListTaskEventsHandler struct{}

func (fakeListTaskEventsHandler) Handle(context.Context, taskquery.ListEventsQuery) (taskquery.ListEventsResult, error) {
	return taskquery.ListEventsResult{
		Items: []taskquery.TaskEventItem{{ID: 1, Message: "line1"}},
	}, nil
}

type fakeListAccountsHandler struct{}

func (fakeListAccountsHandler) Handle(context.Context, accountquery.ListAccountsQuery) (accountquery.ListAccountsResult, error) {
	return accountquery.ListAccountsResult{
		Total: 1,
		Page:  1,
		Items: []accountquery.AccountItem{{ID: 1, Platform: "dummy", Email: "user@example.com", Status: "registered"}},
	}, nil
}

type fakeDashboardStatsHandler struct{}

func (fakeDashboardStatsHandler) Handle(context.Context) (accountquery.DashboardStatsResult, error) {
	return accountquery.DashboardStatsResult{
		Total:      1,
		ByPlatform: map[string]int64{"dummy": 1},
		ByStatus:   map[string]int64{"registered": 1},
	}, nil
}

type fakeGetConfigHandler struct{}

func (fakeGetConfigHandler) Handle(context.Context) (map[string]string, error) {
	return map[string]string{"mail_provider": "moemail", "yescaptcha_key": ""}, nil
}

type fakeCreateTaskHandler struct{}

func (fakeCreateTaskHandler) Handle(context.Context, taskcommand.Command) (taskcommand.Result, error) {
	return taskcommand.Result{TaskID: "task_123"}, nil
}

type fakeCheckAccountHandler struct{}

func (fakeCheckAccountHandler) Handle(context.Context, accountcommand.CheckAccountCommand) (accountcommand.CheckAccountResult, error) {
	return accountcommand.CheckAccountResult{OK: true, Valid: true}, nil
}

type fakeExecuteActionHandler struct{}

func (fakeExecuteActionHandler) Handle(context.Context, actioncommand.ExecutePlatformActionCommand) (actioncommand.ExecutePlatformActionResult, error) {
	return actioncommand.ExecutePlatformActionResult{OK: true, Data: map[string]any{"message": "done"}}, nil
}

type fakeApplyWorkerEventHandler struct {
	calls []taskcommand.ApplyWorkerEventCommand
}

func (f *fakeApplyWorkerEventHandler) Handle(_ context.Context, cmd taskcommand.ApplyWorkerEventCommand) error {
	f.calls = append(f.calls, cmd)
	return nil
}

func TestNewRouterExposesHealthEndpoint(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	logger := zerologadapter.New("info")
	router := NewRouter(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}

	if payload["status"] != "ok" {
		t.Fatalf("expected status=ok, got %#v", payload["status"])
	}
}

func TestNewRouterExposesTaskAndPlatformEndpoints(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	logger := zerologadapter.New("info")
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		ListTasks:         fakeTaskQueryHandler{},
		ListPlatforms:     fakePlatformQueryHandler{},
		ListAccounts:      fakeListAccountsHandler{},
		GetDashboardStats: fakeDashboardStatsHandler{},
		GetConfig:         fakeGetConfigHandler{},
		GetTask:           fakeGetTaskHandler{},
		ListTaskLogs:      fakeListTaskLogsHandler{},
		ListTaskEvents:    fakeListTaskEventsHandler{},
		CreateTask:        fakeCreateTaskHandler{},
	})

	taskReq := httptest.NewRequest(http.MethodGet, "/tasks?page=1&page_size=10", nil)
	taskRec := httptest.NewRecorder()
	router.ServeHTTP(taskRec, taskReq)

	if taskRec.Code != http.StatusOK {
		t.Fatalf("expected /tasks to return 200, got %d", taskRec.Code)
	}

	platformReq := httptest.NewRequest(http.MethodGet, "/platforms", nil)
	platformRec := httptest.NewRecorder()
	router.ServeHTTP(platformRec, platformReq)

	if platformRec.Code != http.StatusOK {
		t.Fatalf("expected /platforms to return 200, got %d", platformRec.Code)
	}
	var platformPayload []map[string]any
	if err := json.Unmarshal(platformRec.Body.Bytes(), &platformPayload); err != nil {
		t.Fatalf("expected platform array response, got %v", err)
	}
	if len(platformPayload) != 1 || platformPayload[0]["name"] != "trae" {
		t.Fatalf("unexpected platform payload: %#v", platformPayload)
	}

	apiPlatformReq := httptest.NewRequest(http.MethodGet, "/api/platforms", nil)
	apiPlatformRec := httptest.NewRecorder()
	router.ServeHTTP(apiPlatformRec, apiPlatformReq)
	if apiPlatformRec.Code != http.StatusOK {
		t.Fatalf("expected /api/platforms to return 200, got %d", apiPlatformRec.Code)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/api/accounts/stats", nil)
	statsRec := httptest.NewRecorder()
	router.ServeHTTP(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("expected /api/accounts/stats to return 200, got %d", statsRec.Code)
	}

	configReq := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	configRec := httptest.NewRecorder()
	router.ServeHTTP(configRec, configReq)
	if configRec.Code != http.StatusOK {
		t.Fatalf("expected /api/config to return 200, got %d", configRec.Code)
	}

	getTaskReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task_1", nil)
	getTaskRec := httptest.NewRecorder()
	router.ServeHTTP(getTaskRec, getTaskReq)
	if getTaskRec.Code != http.StatusOK {
		t.Fatalf("expected /api/tasks/:id to return 200, got %d", getTaskRec.Code)
	}
	var taskPayload map[string]any
	if err := json.Unmarshal(getTaskRec.Body.Bytes(), &taskPayload); err != nil {
		t.Fatalf("expected task json, got %v", err)
	}
	if taskPayload["success"] != float64(1) {
		t.Fatalf("expected task payload success field, got %#v", taskPayload)
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/api/tasks/logs?page=1&page_size=10", nil)
	logsRec := httptest.NewRecorder()
	router.ServeHTTP(logsRec, logsReq)
	if logsRec.Code != http.StatusOK {
		t.Fatalf("expected /api/tasks/logs to return 200, got %d", logsRec.Code)
	}

	streamReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task_1/logs/stream", nil)
	streamRec := httptest.NewRecorder()
	router.ServeHTTP(streamRec, streamReq)
	if streamRec.Code != http.StatusOK {
		t.Fatalf("expected /api/tasks/:id/logs/stream to return 200, got %d", streamRec.Code)
	}
	if !strings.Contains(streamRec.Body.String(), "\"line\":\"line1\"") {
		t.Fatalf("expected stream payload to include line event, got %s", streamRec.Body.String())
	}
}

func TestNewRouterExposesRegisterEndpoint(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	logger := zerologadapter.New("info")
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		CreateTask: fakeCreateTaskHandler{},
	})

	req := httptest.NewRequest(http.MethodPost, "/tasks/register", strings.NewReader(`{"platform":"dummy","count":1}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /tasks/register to return 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if payload["task_id"] != "task_123" {
		t.Fatalf("unexpected task response: %#v", payload)
	}
}

func TestNewRouterExposesCheckAndActionEndpoints(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	logger := zerologadapter.New("info")
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		CheckAccount:  fakeCheckAccountHandler{},
		ExecuteAction: fakeExecuteActionHandler{},
	})

	checkReq := httptest.NewRequest(http.MethodPost, "/accounts/1/check", nil)
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)
	if checkRec.Code != http.StatusOK {
		t.Fatalf("expected /accounts/:id/check to return 200, got %d", checkRec.Code)
	}

	actionReq := httptest.NewRequest(http.MethodPost, "/actions/dummy/1/sync_external", strings.NewReader(`{"params":{}}`))
	actionReq.Header.Set("Content-Type", "application/json")
	actionRec := httptest.NewRecorder()
	router.ServeHTTP(actionRec, actionReq)
	if actionRec.Code != http.StatusOK {
		t.Fatalf("expected /actions route to return 200, got %d", actionRec.Code)
	}
}

func TestNewRouterExposesInternalWorkerEventEndpoints(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	logger := zerologadapter.New("info")
	applyHandler := &fakeApplyWorkerEventHandler{}
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		ApplyWorkerEvent: applyHandler,
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/worker/tasks/task_1/log", strings.NewReader(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected internal worker endpoint to return 200, got %d", rec.Code)
	}
	if len(applyHandler.calls) != 1 {
		t.Fatalf("expected one callback event, got %#v", applyHandler.calls)
	}
	if applyHandler.calls[0].TaskID != "task_1" || applyHandler.calls[0].Message != "hello" {
		t.Fatalf("unexpected callback payload: %#v", applyHandler.calls[0])
	}
}
