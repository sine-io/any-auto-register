package gingateway

import (
	"bytes"
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
	configcommand "go-control-plane/internal/application/command/config"
	proxycommand "go-control-plane/internal/application/command/proxy"
	taskcommand "go-control-plane/internal/application/command/task"
	accountquery "go-control-plane/internal/application/query/account"
	actionquery "go-control-plane/internal/application/query/action"
	platformquery "go-control-plane/internal/application/query/platform"
	proxyquery "go-control-plane/internal/application/query/proxy"
	systemquery "go-control-plane/internal/application/query/system"
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

type fakeListProxiesHandler struct{}

func (fakeListProxiesHandler) Handle(context.Context) ([]proxyquery.ProxyItem, error) {
	return []proxyquery.ProxyItem{{ID: 1, URL: "http://1.1.1.1:8080", Region: "US", IsActive: true}}, nil
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

type fakeUpdateConfigHandler struct{}

func (fakeUpdateConfigHandler) Handle(context.Context, configcommand.UpdateConfigCommand) (configcommand.UpdateConfigResult, error) {
	return configcommand.UpdateConfigResult{OK: true, Updated: []string{"mail_provider"}}, nil
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

type fakeListActionsHandler struct{}

func (fakeListActionsHandler) Handle(context.Context, string) (actionquery.ListActionsResult, error) {
	return actionquery.ListActionsResult{
		Actions: []map[string]any{{"id": "sync_external", "available": true}},
	}, nil
}

type fakeProxyCommandHandler struct{}

func (fakeProxyCommandHandler) Add(context.Context, proxycommand.AddProxyCommand) (map[string]any, error) {
	return map[string]any{"id": 1}, nil
}

func (fakeProxyCommandHandler) BulkAdd(context.Context, proxycommand.BulkAddProxiesCommand) (map[string]any, error) {
	return map[string]any{"added": 2}, nil
}

func (fakeProxyCommandHandler) Toggle(context.Context, proxycommand.ToggleProxyCommand) (map[string]any, error) {
	return map[string]any{"is_active": false}, nil
}

func (fakeProxyCommandHandler) Delete(context.Context, proxycommand.DeleteProxyCommand) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

func (fakeProxyCommandHandler) Check(context.Context, proxycommand.CheckProxiesCommand) (map[string]any, error) {
	return map[string]any{"message": "检测任务已启动"}, nil
}

type fakeSolverStatusHandler struct{}

func (fakeSolverStatusHandler) Handle(context.Context) (systemquery.SolverStatusResult, error) {
	return systemquery.SolverStatusResult{Running: true}, nil
}

type fakeRestartSolverHandler struct{}

func (fakeRestartSolverHandler) Handle(context.Context) (map[string]any, error) {
	return map[string]any{"message": "重启中"}, nil
}

type fakeListIntegrationServicesHandler struct{}

func (fakeListIntegrationServicesHandler) Handle(context.Context) (map[string]any, error) {
	return map[string]any{"items": []map[string]any{{"name": "grok2api", "running": true}}}, nil
}

type fakeIntegrationCommandHandler struct{}

func (fakeIntegrationCommandHandler) StartAll(context.Context) (map[string]any, error) {
	return map[string]any{"items": []map[string]any{}}, nil
}
func (fakeIntegrationCommandHandler) StopAll(context.Context) (map[string]any, error) {
	return map[string]any{"items": []map[string]any{}}, nil
}
func (fakeIntegrationCommandHandler) Start(context.Context, string) (map[string]any, error) {
	return map[string]any{"name": "grok2api"}, nil
}
func (fakeIntegrationCommandHandler) Install(context.Context, string) (map[string]any, error) {
	return map[string]any{"name": "grok2api"}, nil
}
func (fakeIntegrationCommandHandler) Stop(context.Context, string) (map[string]any, error) {
	return map[string]any{"name": "grok2api"}, nil
}
func (fakeIntegrationCommandHandler) Backfill(context.Context, []string) (map[string]any, error) {
	return map[string]any{"total": 1, "success": 1}, nil
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
		ListTasks:               fakeTaskQueryHandler{},
		ListPlatforms:           fakePlatformQueryHandler{},
		ListProxies:             fakeListProxiesHandler{},
		ListAccounts:            fakeListAccountsHandler{},
		GetDashboardStats:       fakeDashboardStatsHandler{},
		GetConfig:               fakeGetConfigHandler{},
		UpdateConfig:            fakeUpdateConfigHandler{},
		GetTask:                 fakeGetTaskHandler{},
		ListTaskLogs:            fakeListTaskLogsHandler{},
		ListTaskEvents:          fakeListTaskEventsHandler{},
		GetSolverStatus:         fakeSolverStatusHandler{},
		RestartSolver:           fakeRestartSolverHandler{},
		ListIntegrationServices: fakeListIntegrationServicesHandler{},
		CreateTask:              fakeCreateTaskHandler{},
		ProxyCommands:           fakeProxyCommandHandler{},
		IntegrationCommands:     fakeIntegrationCommandHandler{},
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

	configPutReq := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(`{"data":{"mail_provider":"moemail"}}`))
	configPutReq.Header.Set("Content-Type", "application/json")
	configPutRec := httptest.NewRecorder()
	router.ServeHTTP(configPutRec, configPutReq)
	if configPutRec.Code != http.StatusOK {
		t.Fatalf("expected PUT /api/config to return 200, got %d", configPutRec.Code)
	}

	proxiesReq := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	proxiesRec := httptest.NewRecorder()
	router.ServeHTTP(proxiesRec, proxiesReq)
	if proxiesRec.Code != http.StatusOK {
		t.Fatalf("expected /api/proxies to return 200, got %d", proxiesRec.Code)
	}

	solverReq := httptest.NewRequest(http.MethodGet, "/api/solver/status", nil)
	solverRec := httptest.NewRecorder()
	router.ServeHTTP(solverRec, solverReq)
	if solverRec.Code != http.StatusOK {
		t.Fatalf("expected /api/solver/status to return 200, got %d", solverRec.Code)
	}

	solverRestartReq := httptest.NewRequest(http.MethodPost, "/api/solver/restart", strings.NewReader(`{}`))
	solverRestartReq.Header.Set("Content-Type", "application/json")
	solverRestartRec := httptest.NewRecorder()
	router.ServeHTTP(solverRestartRec, solverRestartReq)
	if solverRestartRec.Code != http.StatusOK {
		t.Fatalf("expected /api/solver/restart to return 200, got %d", solverRestartRec.Code)
	}

	integrationReq := httptest.NewRequest(http.MethodGet, "/api/integrations/services", nil)
	integrationRec := httptest.NewRecorder()
	router.ServeHTTP(integrationRec, integrationReq)
	if integrationRec.Code != http.StatusOK {
		t.Fatalf("expected /api/integrations/services to return 200, got %d", integrationRec.Code)
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

	proxyAddReq := httptest.NewRequest(http.MethodPost, "/api/proxies", strings.NewReader(`{"url":"http://1.1.1.1:8080","region":"US"}`))
	proxyAddReq.Header.Set("Content-Type", "application/json")
	proxyAddRec := httptest.NewRecorder()
	router.ServeHTTP(proxyAddRec, proxyAddReq)
	if proxyAddRec.Code != http.StatusOK {
		t.Fatalf("expected POST /api/proxies to return 200, got %d", proxyAddRec.Code)
	}

	startAllReq := httptest.NewRequest(http.MethodPost, "/api/integrations/services/start-all", strings.NewReader(`{}`))
	startAllReq.Header.Set("Content-Type", "application/json")
	startAllRec := httptest.NewRecorder()
	router.ServeHTTP(startAllRec, startAllReq)
	if startAllRec.Code != http.StatusOK {
		t.Fatalf("expected POST /api/integrations/services/start-all to return 200, got %d", startAllRec.Code)
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

func TestNewRouterWritesAuditLogsForWriteEndpoints(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	var buf bytes.Buffer
	logger := zerologadapter.NewWithWriter("info", &buf)
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		CreateTask:   fakeCreateTaskHandler{},
		UpdateConfig: fakeUpdateConfigHandler{},
	})

	taskReq := httptest.NewRequest(http.MethodPost, "/tasks/register", strings.NewReader(`{"platform":"dummy","count":1}`))
	taskReq.Header.Set("Content-Type", "application/json")
	taskRec := httptest.NewRecorder()
	router.ServeHTTP(taskRec, taskReq)
	if taskRec.Code != http.StatusOK {
		t.Fatalf("expected /tasks/register to return 200, got %d", taskRec.Code)
	}

	configReq := httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(`{"data":{"mail_provider":"moemail"}}`))
	configReq.Header.Set("Content-Type", "application/json")
	configRec := httptest.NewRecorder()
	router.ServeHTTP(configRec, configReq)
	if configRec.Code != http.StatusOK {
		t.Fatalf("expected /config to return 200, got %d", configRec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"kind":"audit"`) {
		t.Fatalf("expected audit log marker, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"action":"task.register"`) {
		t.Fatalf("expected task register audit log, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"task_id":"task_123"`) {
		t.Fatalf("expected task id in audit log, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"action":"config.update"`) {
		t.Fatalf("expected config update audit log, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"updated_keys":["mail_provider"]`) {
		t.Fatalf("expected updated keys in audit log, got %s", logOutput)
	}
}

func TestNewRouterExposesCheckAndActionEndpoints(t *testing.T) {
	cfg, err := viperconfig.Load("")
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	logger := zerologadapter.New("info")
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		ListActions:   fakeListActionsHandler{},
		CheckAccount:  fakeCheckAccountHandler{},
		ExecuteAction: fakeExecuteActionHandler{},
	})

	actionsReq := httptest.NewRequest(http.MethodGet, "/actions/dummy", nil)
	actionsRec := httptest.NewRecorder()
	router.ServeHTTP(actionsRec, actionsReq)
	if actionsRec.Code != http.StatusOK {
		t.Fatalf("expected /actions/:platform to return 200, got %d", actionsRec.Code)
	}

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
	cfg.Internal.CallbackToken = "secret-token"

	logger := zerologadapter.New("info")
	applyHandler := &fakeApplyWorkerEventHandler{}
	router := NewRouterWithDependencies(cfg, logger, Dependencies{
		ApplyWorkerEvent: applyHandler,
	})

	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/internal/worker/tasks/task_1/log", strings.NewReader(`{"message":"hello"}`))
	unauthorizedReq.Header.Set("Content-Type", "application/json")
	unauthorizedRec := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedRec, unauthorizedReq)

	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to return 401, got %d", unauthorizedRec.Code)
	}
	if len(applyHandler.calls) != 0 {
		t.Fatalf("expected unauthorized request to be ignored, got %#v", applyHandler.calls)
	}

	req := httptest.NewRequest(http.MethodPost, "/internal/worker/tasks/task_1/log", strings.NewReader(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AAR-Internal-Callback-Token", "secret-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected valid token request to return 200, got %d", rec.Code)
	}
	if len(applyHandler.calls) != 1 {
		t.Fatalf("expected one callback event, got %#v", applyHandler.calls)
	}
	if applyHandler.calls[0].TaskID != "task_1" || applyHandler.calls[0].Message != "hello" {
		t.Fatalf("unexpected callback payload: %#v", applyHandler.calls[0])
	}
}
