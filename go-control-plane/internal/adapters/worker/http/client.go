package workerhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) Client {
	return Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (c Client) Register(ctx context.Context, req workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	var payload workerport.RegisterResponse
	if err := c.post(ctx, "/api/worker/register", req, &payload); err != nil {
		return workerport.RegisterResponse{}, err
	}
	return payload, nil
}

func (c Client) CheckAccount(ctx context.Context, req workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	var payload workerport.CheckAccountResponse
	if err := c.post(ctx, "/api/worker/check-account", req, &payload); err != nil {
		return workerport.CheckAccountResponse{}, err
	}
	return payload, nil
}

func (c Client) ExecuteAction(ctx context.Context, req workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	var payload workerport.ExecuteActionResponse
	if err := c.post(ctx, "/api/worker/execute-action", req, &payload); err != nil {
		return workerport.ExecuteActionResponse{}, err
	}
	return payload, nil
}

func (c Client) GetSolverStatus(ctx context.Context) (workerport.SolverStatusResponse, error) {
	var payload workerport.SolverStatusResponse
	if err := c.get(ctx, "/api/solver/status", &payload); err != nil {
		return workerport.SolverStatusResponse{}, err
	}
	return payload, nil
}

func (c Client) RestartSolver(ctx context.Context) (map[string]any, error) {
	var payload map[string]any
	if err := c.post(ctx, "/api/solver/restart", map[string]any{}, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c Client) ListIntegrationServices(ctx context.Context) (workerport.IntegrationServicesResponse, error) {
	var payload workerport.IntegrationServicesResponse
	if err := c.get(ctx, "/api/integrations/services", &payload); err != nil {
		return workerport.IntegrationServicesResponse{}, err
	}
	return payload, nil
}

func (c Client) StartAllIntegrationServices(ctx context.Context) (workerport.IntegrationServicesResponse, error) {
	var payload workerport.IntegrationServicesResponse
	if err := c.post(ctx, "/api/integrations/services/start-all", map[string]any{}, &payload); err != nil {
		return workerport.IntegrationServicesResponse{}, err
	}
	return payload, nil
}

func (c Client) StopAllIntegrationServices(ctx context.Context) (workerport.IntegrationServicesResponse, error) {
	var payload workerport.IntegrationServicesResponse
	if err := c.post(ctx, "/api/integrations/services/stop-all", map[string]any{}, &payload); err != nil {
		return workerport.IntegrationServicesResponse{}, err
	}
	return payload, nil
}

func (c Client) StartIntegrationService(ctx context.Context, name string) (map[string]any, error) {
	var payload map[string]any
	if err := c.post(ctx, "/api/integrations/services/"+name+"/start", map[string]any{}, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c Client) InstallIntegrationService(ctx context.Context, name string) (map[string]any, error) {
	var payload map[string]any
	if err := c.post(ctx, "/api/integrations/services/"+name+"/install", map[string]any{}, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c Client) StopIntegrationService(ctx context.Context, name string) (map[string]any, error) {
	var payload map[string]any
	if err := c.post(ctx, "/api/integrations/services/"+name+"/stop", map[string]any{}, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c Client) BackfillIntegrations(ctx context.Context, platforms []string) (map[string]any, error) {
	var payload map[string]any
	if err := c.post(ctx, "/api/integrations/backfill", map[string]any{"platforms": platforms}, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c Client) post(ctx context.Context, path string, reqBody any, out any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		switch v := out.(type) {
		case *workerport.RegisterResponse:
			if v.Error != "" {
				return errors.New(v.Error)
			}
		case *workerport.CheckAccountResponse:
			if v.Error != "" {
				return errors.New(v.Error)
			}
		case *workerport.ExecuteActionResponse:
			if v.Error != "" {
				return errors.New(v.Error)
			}
		}
		return errors.New(resp.Status)
	}
	return nil
}

func (c Client) get(ctx context.Context, path string, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New(resp.Status)
	}
	return nil
}
