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
