package worker

import "context"

type RegisterRequest struct {
	TaskID               string         `json:"task_id,omitempty"`
	CallbackBaseURL      string         `json:"callback_base_url,omitempty"`
	CallbackToken        string         `json:"callback_token,omitempty"`
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

type RegisterResponse struct {
	OK           bool     `json:"ok"`
	SuccessCount int      `json:"success_count"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors"`
	CashierURLs  []string `json:"cashier_urls"`
	Logs         []string `json:"logs"`
	Error        string   `json:"error"`
}

type CheckAccountRequest struct {
	Platform  string `json:"platform"`
	AccountID int64  `json:"account_id"`
}

type CheckAccountResponse struct {
	OK    bool   `json:"ok"`
	Valid bool   `json:"valid"`
	Error string `json:"error"`
}

type ExecuteActionRequest struct {
	Platform  string         `json:"platform"`
	AccountID int64          `json:"account_id"`
	ActionID  string         `json:"action_id"`
	Params    map[string]any `json:"params"`
}

type ExecuteActionResponse struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data"`
	Error string         `json:"error"`
}

type ListActionsResponse struct {
	Actions []map[string]any `json:"actions"`
}

type SolverStatusResponse struct {
	Running bool   `json:"running"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
}

type IntegrationServicesResponse struct {
	Items []map[string]any `json:"items"`
}

type Client interface {
	Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error)
	CheckAccount(ctx context.Context, req CheckAccountRequest) (CheckAccountResponse, error)
	ExecuteAction(ctx context.Context, req ExecuteActionRequest) (ExecuteActionResponse, error)
	ListActions(ctx context.Context, platform string) (ListActionsResponse, error)
	GetSolverStatus(ctx context.Context) (SolverStatusResponse, error)
	RestartSolver(ctx context.Context) (map[string]any, error)
	ListIntegrationServices(ctx context.Context) (IntegrationServicesResponse, error)
	StartAllIntegrationServices(ctx context.Context) (IntegrationServicesResponse, error)
	StopAllIntegrationServices(ctx context.Context) (IntegrationServicesResponse, error)
	StartIntegrationService(ctx context.Context, name string) (map[string]any, error)
	InstallIntegrationService(ctx context.Context, name string) (map[string]any, error)
	StopIntegrationService(ctx context.Context, name string) (map[string]any, error)
	BackfillIntegrations(ctx context.Context, platforms []string) (map[string]any, error)
}
