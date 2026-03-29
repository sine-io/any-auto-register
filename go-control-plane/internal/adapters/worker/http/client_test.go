package workerhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

func TestClientRegisterParsesWorkerResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/worker/register" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req workerport.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("expected request json, got %v", err)
		}
		_ = json.NewEncoder(w).Encode(workerport.RegisterResponse{
			OK:           true,
			SuccessCount: 1,
			CashierURLs:  []string{"https://example.com/pay"},
			Logs:         []string{"started", "done"},
		})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.Register(context.Background(), workerport.RegisterRequest{Platform: "dummy", Count: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK || result.SuccessCount != 1 {
		t.Fatalf("unexpected worker response: %#v", result)
	}
	if len(result.Logs) != 2 {
		t.Fatalf("expected logs from worker, got %#v", result.Logs)
	}
}

func TestClientCheckAccountParsesWorkerResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/worker/check-account" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(workerport.CheckAccountResponse{
			OK:    true,
			Valid: true,
		})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.CheckAccount(context.Background(), workerport.CheckAccountRequest{Platform: "dummy", AccountID: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK || !result.Valid {
		t.Fatalf("unexpected check response: %#v", result)
	}
}

func TestClientExecuteActionParsesWorkerResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/worker/execute-action" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(workerport.ExecuteActionResponse{
			OK: true,
			Data: map[string]any{
				"message": "done",
			},
		})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.ExecuteAction(context.Background(), workerport.ExecuteActionRequest{
		Platform: "dummy", AccountID: 1, ActionID: "sync_external", Params: map[string]any{},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK || result.Data["message"] != "done" {
		t.Fatalf("unexpected action response: %#v", result)
	}
}

func TestClientGetsSolverStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/solver/status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(workerport.SolverStatusResponse{Running: true})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.GetSolverStatus(context.Background())
	if err != nil || !result.Running {
		t.Fatalf("unexpected solver status: %#v err=%v", result, err)
	}
}

func TestClientListsIntegrationServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/integrations/services" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(workerport.IntegrationServicesResponse{
			Items: []map[string]any{{"name": "grok2api", "running": true}},
		})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.ListIntegrationServices(context.Background())
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("unexpected integration services: %#v err=%v", result, err)
	}
}

func TestClientListsActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/worker/actions/dummy" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(workerport.ListActionsResponse{
			Actions: []map[string]any{{"id": "sync_external", "available": true}},
		})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.ListActions(context.Background(), "dummy")
	if err != nil || len(result.Actions) != 1 {
		t.Fatalf("unexpected actions result: %#v err=%v", result, err)
	}
}
