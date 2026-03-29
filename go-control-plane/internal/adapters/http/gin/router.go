package gingateway

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	viperconfig "go-control-plane/internal/adapters/config/viper"
	sqliteadapter "go-control-plane/internal/adapters/persistence/sqlite"
	workerhttp "go-control-plane/internal/adapters/worker/http"
	accountcommand "go-control-plane/internal/application/command/account"
	actioncommand "go-control-plane/internal/application/command/action"
	configcommand "go-control-plane/internal/application/command/config"
	integrationcommand "go-control-plane/internal/application/command/integration"
	proxycommand "go-control-plane/internal/application/command/proxy"
	systemcommand "go-control-plane/internal/application/command/system"
	taskcommand "go-control-plane/internal/application/command/task"
	accountquery "go-control-plane/internal/application/query/account"
	actionquery "go-control-plane/internal/application/query/action"
	configquery "go-control-plane/internal/application/query/config"
	integrationquery "go-control-plane/internal/application/query/integration"
	platformquery "go-control-plane/internal/application/query/platform"
	proxyquery "go-control-plane/internal/application/query/proxy"
	systemquery "go-control-plane/internal/application/query/system"
	taskquery "go-control-plane/internal/application/query/task"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type TaskQueryHandler interface {
	Handle(context.Context, taskquery.Query) (taskquery.Result, error)
}

type PlatformQueryHandler interface {
	Handle(context.Context) (platformquery.Result, error)
}

type ActionListHandler interface {
	Handle(context.Context, string) (actionquery.ListActionsResult, error)
}

type ListProxiesHandler interface {
	Handle(context.Context) ([]proxyquery.ProxyItem, error)
}

type ListAccountsHandler interface {
	Handle(context.Context, accountquery.ListAccountsQuery) (accountquery.ListAccountsResult, error)
}

type DashboardStatsHandler interface {
	Handle(context.Context) (accountquery.DashboardStatsResult, error)
}

type GetConfigHandler interface {
	Handle(context.Context) (map[string]string, error)
}

type UpdateConfigHandler interface {
	Handle(context.Context, configcommand.UpdateConfigCommand) (configcommand.UpdateConfigResult, error)
}

type CreateAccountHandler interface {
	Handle(context.Context, accountcommand.CreateAccountCommand) (accountcommand.AccountMutationResult, error)
}

type UpdateAccountHandler interface {
	Handle(context.Context, accountcommand.UpdateAccountCommand) (accountcommand.AccountMutationResult, error)
}

type DeleteAccountHandler interface {
	Handle(context.Context, accountcommand.DeleteAccountCommand) (map[string]any, error)
}

type BatchDeleteAccountsHandler interface {
	Handle(context.Context, accountcommand.BatchDeleteAccountsCommand) (accountcommand.BatchDeleteAccountsResult, error)
}

type ImportAccountsHandler interface {
	Handle(context.Context, accountcommand.ImportAccountsCommand) (accountcommand.ImportAccountsResult, error)
}

type GetTaskHandler interface {
	Handle(context.Context, taskquery.GetTaskQuery) (taskquery.TaskItem, error)
}

type ListTaskLogsHandler interface {
	Handle(context.Context, taskquery.ListLogsQuery) (taskquery.ListLogsResult, error)
}

type ListTaskEventsHandler interface {
	Handle(context.Context, taskquery.ListEventsQuery) (taskquery.ListEventsResult, error)
}

type DeleteTaskLogsHandler interface {
	Handle(context.Context, taskcommand.DeleteTaskLogsCommand) (taskcommand.DeleteTaskLogsResult, error)
}

type SolverStatusHandler interface {
	Handle(context.Context) (systemquery.SolverStatusResult, error)
}

type RestartSolverHandler interface {
	Handle(context.Context) (map[string]any, error)
}

type ListIntegrationServicesHandler interface {
	Handle(context.Context) (map[string]any, error)
}

type IntegrationCommandHandler interface {
	StartAll(context.Context) (map[string]any, error)
	StopAll(context.Context) (map[string]any, error)
	Start(context.Context, string) (map[string]any, error)
	Install(context.Context, string) (map[string]any, error)
	Stop(context.Context, string) (map[string]any, error)
	Backfill(context.Context, []string) (map[string]any, error)
}

type Dependencies struct {
	ListTasks               TaskQueryHandler
	ListPlatforms           PlatformQueryHandler
	ListActions             ActionListHandler
	ListProxies             ListProxiesHandler
	ListAccounts            ListAccountsHandler
	GetDashboardStats       DashboardStatsHandler
	GetConfig               GetConfigHandler
	UpdateConfig            UpdateConfigHandler
	CreateAccount           CreateAccountHandler
	UpdateAccount           UpdateAccountHandler
	DeleteAccount           DeleteAccountHandler
	BatchDeleteAccounts     BatchDeleteAccountsHandler
	ImportAccounts          ImportAccountsHandler
	GetTask                 GetTaskHandler
	ListTaskLogs            ListTaskLogsHandler
	ListTaskEvents          ListTaskEventsHandler
	DeleteTaskLogs          DeleteTaskLogsHandler
	GetSolverStatus         SolverStatusHandler
	RestartSolver           RestartSolverHandler
	ListIntegrationServices ListIntegrationServicesHandler
	CreateTask              CreateTaskHandler
	ApplyWorkerEvent        ApplyWorkerEventHandler
	CheckAccount            CheckAccountHandler
	ExecuteAction           ExecuteActionHandler
	ProxyCommands           ProxyCommandHandler
	IntegrationCommands     IntegrationCommandHandler
}

type CreateTaskHandler interface {
	Handle(context.Context, taskcommand.Command) (taskcommand.Result, error)
}

type ApplyWorkerEventHandler interface {
	Handle(context.Context, taskcommand.ApplyWorkerEventCommand) error
}

type CheckAccountHandler interface {
	Handle(context.Context, accountcommand.CheckAccountCommand) (accountcommand.CheckAccountResult, error)
}

type ExecuteActionHandler interface {
	Handle(context.Context, actioncommand.ExecutePlatformActionCommand) (actioncommand.ExecutePlatformActionResult, error)
}

type ProxyCommandHandler interface {
	Add(context.Context, proxycommand.AddProxyCommand) (map[string]any, error)
	BulkAdd(context.Context, proxycommand.BulkAddProxiesCommand) (map[string]any, error)
	Toggle(context.Context, proxycommand.ToggleProxyCommand) (map[string]any, error)
	Delete(context.Context, proxycommand.DeleteProxyCommand) (map[string]any, error)
	Check(context.Context, proxycommand.CheckProxiesCommand) (map[string]any, error)
}

func NewRouter(cfg viperconfig.AppConfig, logger zerolog.Logger) *gin.Engine {
	db, err := sqliteadapter.Open(cfg.Database.URL)
	if err != nil {
		panic(err)
	}

	deps := buildDependencies(db, cfg)
	return NewRouterWithDependencies(cfg, logger, deps)
}

func buildDependencies(db *sql.DB, cfg viperconfig.AppConfig) Dependencies {
	taskRepo := sqliteadapter.NewTaskRepository(db)
	taskCommandRepo := sqliteadapter.NewTaskCommandRepository(db)
	accountRepo := sqliteadapter.NewAccountRepository(db)
	configRepo := sqliteadapter.NewConfigRepository(db)
	proxyRepo := sqliteadapter.NewProxyRepository(db)
	platformRepo := sqliteadapter.NewPlatformRepository(db, cfg.Platforms)
	workerClient := workerhttp.New(cfg.Worker.BaseURL)
	applyWorkerEvent := taskcommand.NewApplyWorkerEventHandler(taskCommandRepo)
	callbackBaseURL := cfg.Server.CallbackBaseURL
	if callbackBaseURL == "" {
		callbackBaseURL = cfg.Server.PublicBaseURL
	}
	integrationHandler := integrationcommand.NewHandler(workerClient)

	return Dependencies{
		ListTasks:               taskquery.NewHandler(taskRepo),
		ListPlatforms:           platformquery.NewHandler(platformRepo),
		ListActions:             actionquery.NewListActionsHandler(workerClient),
		ListProxies:             proxyquery.NewListProxiesHandler(proxyRepo),
		ListAccounts:            accountquery.NewListAccountsHandler(accountRepo),
		GetDashboardStats:       accountquery.NewDashboardStatsHandler(accountRepo),
		GetConfig:               configquery.NewGetConfigHandler(configRepo),
		UpdateConfig:            configcommand.NewUpdateConfigHandler(configRepo),
		CreateAccount:           accountcommand.NewCreateAccountHandler(accountRepo),
		UpdateAccount:           accountcommand.NewUpdateAccountHandler(accountRepo),
		DeleteAccount:           accountcommand.NewDeleteAccountHandler(accountRepo),
		BatchDeleteAccounts:     accountcommand.NewBatchDeleteAccountsHandler(accountRepo),
		ImportAccounts:          accountcommand.NewImportAccountsHandler(accountRepo),
		GetTask:                 taskquery.NewGetTaskHandler(taskRepo),
		ListTaskLogs:            taskquery.NewListLogsHandler(taskRepo),
		ListTaskEvents:          taskquery.NewListEventsHandler(taskRepo),
		DeleteTaskLogs:          taskcommand.NewDeleteTaskLogsHandler(taskRepo),
		GetSolverStatus:         systemquery.NewSolverStatusHandler(workerClient),
		RestartSolver:           systemcommand.NewRestartSolverHandler(workerClient),
		ListIntegrationServices: integrationquery.NewListServicesHandler(workerClient),
		CreateTask:              taskcommand.NewHandler(taskCommandRepo, workerClient, nil, nil, callbackBaseURL, cfg.Internal.CallbackToken),
		ApplyWorkerEvent:        applyWorkerEvent,
		CheckAccount:            accountcommand.NewCheckAccountHandler(accountRepo, workerClient),
		ExecuteAction:           actioncommand.NewExecutePlatformActionHandler(accountRepo, workerClient),
		ProxyCommands:           proxycommand.NewProxyCommandHandler(proxyRepo),
		IntegrationCommands:     integrationHandler,
	}
}

func NewRouterWithDependencies(cfg viperconfig.AppConfig, logger zerolog.Logger, deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	registerPublicRoutes(router, deps, logger)
	registerPublicRoutes(router.Group("/api"), deps, logger)
	registerInternalWorkerRoutes(router, deps, cfg.Internal.CallbackToken)

	return router
}

type routeRegistrar interface {
	GET(string, ...gin.HandlerFunc) gin.IRoutes
	POST(string, ...gin.HandlerFunc) gin.IRoutes
	PUT(string, ...gin.HandlerFunc) gin.IRoutes
	PATCH(string, ...gin.HandlerFunc) gin.IRoutes
	DELETE(string, ...gin.HandlerFunc) gin.IRoutes
}

func registerPublicRoutes(router routeRegistrar, deps Dependencies, logger zerolog.Logger) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "go-control-plane",
		})
	})

	router.GET("/tasks", func(c *gin.Context) {
		if deps.ListTasks == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "tasks query handler not configured"})
			return
		}
		result, err := deps.ListTasks.Handle(c.Request.Context(), taskquery.Query{
			Page:     parseInt(c.DefaultQuery("page", "1"), 1),
			PageSize: parseInt(c.DefaultQuery("page_size", "50"), 50),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.GET("/tasks/logs", func(c *gin.Context) {
		if deps.ListTaskLogs == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "task logs query handler not configured"})
			return
		}
		result, err := deps.ListTaskLogs.Handle(c.Request.Context(), taskquery.ListLogsQuery{
			Platform: c.DefaultQuery("platform", ""),
			Page:     parseInt(c.DefaultQuery("page", "1"), 1),
			PageSize: parseInt(c.DefaultQuery("page_size", "50"), 50),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/tasks/logs/batch-delete", func(c *gin.Context) {
		if deps.DeleteTaskLogs == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "task log delete handler not configured"})
			return
		}
		var cmd taskcommand.DeleteTaskLogsCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "task_logs.batch_delete", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.DeleteTaskLogs.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "task_logs.batch_delete", "error", map[string]any{"count": len(cmd.IDs)}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "task_logs.batch_delete", "ok", map[string]any{
			"count":   len(cmd.IDs),
			"deleted": result.Deleted,
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.GET("/tasks/:taskID", func(c *gin.Context) {
		if deps.GetTask == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "task detail handler not configured"})
			return
		}
		result, err := deps.GetTask.Handle(c.Request.Context(), taskquery.GetTaskQuery{TaskID: c.Param("taskID")})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.GET("/tasks/:taskID/logs/stream", func(c *gin.Context) {
		if deps.ListTaskEvents == nil || deps.GetTask == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "task event handlers not configured"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}

		sinceID, _ := strconv.ParseInt(c.DefaultQuery("since", "0"), 10, 64)
		taskID := c.Param("taskID")
		for {
			select {
			case <-c.Request.Context().Done():
				return
			default:
			}

			result, err := deps.ListTaskEvents.Handle(c.Request.Context(), taskquery.ListEventsQuery{
				TaskID:  taskID,
				SinceID: sinceID,
			})
			if err != nil {
				return
			}
			for _, item := range result.Items {
				sinceID = item.ID
				_, _ = fmt.Fprintf(c.Writer, "data: {\"line\":%q,\"event_id\":%d}\n\n", item.Message, item.ID)
				flusher.Flush()
			}

			task, err := deps.GetTask.Handle(c.Request.Context(), taskquery.GetTaskQuery{TaskID: taskID})
			if err == nil && (task.Status == "done" || task.Status == "failed") {
				_, _ = fmt.Fprintf(c.Writer, "data: {\"done\":true,\"status\":%q}\n\n", task.Status)
				flusher.Flush()
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	})

	router.POST("/tasks/register", func(c *gin.Context) {
		if deps.CreateTask == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "task command handler not configured"})
			return
		}
		var cmd taskcommand.Command
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "task.register", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.CreateTask.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "task.register", "error", map[string]any{
				"platform": cmd.Platform,
				"count":    cmd.Count,
			}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "task.register", "ok", map[string]any{
			"platform": cmd.Platform,
			"count":    cmd.Count,
			"task_id":  result.TaskID,
		}, nil)
		c.JSON(http.StatusOK, gin.H{"task_id": result.TaskID})
	})

	router.GET("/platforms", func(c *gin.Context) {
		if deps.ListPlatforms == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "platforms query handler not configured"})
			return
		}
		result, err := deps.ListPlatforms.Handle(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result.Items)
	})

	router.GET("/actions/:platform", func(c *gin.Context) {
		if deps.ListActions == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "actions query handler not configured"})
			return
		}
		result, err := deps.ListActions.Handle(c.Request.Context(), c.Param("platform"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.GET("/accounts", func(c *gin.Context) {
		if deps.ListAccounts == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "accounts query handler not configured"})
			return
		}
		result, err := deps.ListAccounts.Handle(c.Request.Context(), accountquery.ListAccountsQuery{
			Platform: c.DefaultQuery("platform", ""),
			Status:   c.DefaultQuery("status", ""),
			Email:    c.DefaultQuery("email", ""),
			Page:     parseInt(c.DefaultQuery("page", "1"), 1),
			PageSize: parseInt(c.DefaultQuery("page_size", "20"), 20),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.GET("/accounts/stats", func(c *gin.Context) {
		if deps.GetDashboardStats == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "dashboard stats handler not configured"})
			return
		}
		result, err := deps.GetDashboardStats.Handle(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/accounts", func(c *gin.Context) {
		if deps.CreateAccount == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "account create handler not configured"})
			return
		}
		var cmd accountcommand.CreateAccountCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "account.create", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.CreateAccount.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "account.create", "error", map[string]any{"platform": cmd.Platform}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "account.create", "ok", map[string]any{
			"account_id": result.ID,
			"platform":   result.Platform,
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/accounts/import", func(c *gin.Context) {
		if deps.ImportAccounts == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "account import handler not configured"})
			return
		}
		var cmd accountcommand.ImportAccountsCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "account.import", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.ImportAccounts.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "account.import", "error", map[string]any{"platform": cmd.Platform, "count": len(cmd.Lines)}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "account.import", "ok", map[string]any{
			"platform": cmd.Platform,
			"count":    len(cmd.Lines),
			"created":  result.Created,
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/accounts/batch-delete", func(c *gin.Context) {
		if deps.BatchDeleteAccounts == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "account batch delete handler not configured"})
			return
		}
		var cmd accountcommand.BatchDeleteAccountsCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "account.batch_delete", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.BatchDeleteAccounts.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "account.batch_delete", "error", map[string]any{"count": len(cmd.IDs)}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "account.batch_delete", "ok", map[string]any{
			"count":   len(cmd.IDs),
			"deleted": result.Deleted,
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.PATCH("/accounts/:accountID", func(c *gin.Context) {
		if deps.UpdateAccount == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "account update handler not configured"})
			return
		}
		accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
		if err != nil {
			auditLog(logger, "account.update", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}
		var cmd accountcommand.UpdateAccountCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "account.update", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cmd.AccountID = accountID
		result, err := deps.UpdateAccount.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "account.update", "error", map[string]any{"account_id": accountID}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "account.update", "ok", map[string]any{"account_id": accountID}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.DELETE("/accounts/:accountID", func(c *gin.Context) {
		if deps.DeleteAccount == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "account delete handler not configured"})
			return
		}
		accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
		if err != nil {
			auditLog(logger, "account.delete", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}
		result, err := deps.DeleteAccount.Handle(c.Request.Context(), accountcommand.DeleteAccountCommand{AccountID: accountID})
		if err != nil {
			auditLog(logger, "account.delete", "error", map[string]any{"account_id": accountID}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "account.delete", "ok", map[string]any{"account_id": accountID}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.GET("/config", func(c *gin.Context) {
		if deps.GetConfig == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "config handler not configured"})
			return
		}
		result, err := deps.GetConfig.Handle(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.PUT("/config", func(c *gin.Context) {
		if deps.UpdateConfig == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "config update handler not configured"})
			return
		}
		var cmd configcommand.UpdateConfigCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "config.update", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.UpdateConfig.Handle(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "config.update", "error", nil, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "config.update", "ok", map[string]any{"updated_keys": result.Updated}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.GET("/proxies", func(c *gin.Context) {
		if deps.ListProxies == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "proxies query handler not configured"})
			return
		}
		result, err := deps.ListProxies.Handle(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/proxies", func(c *gin.Context) {
		if deps.ProxyCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "proxy command handler not configured"})
			return
		}
		var cmd proxycommand.AddProxyCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "proxy.add", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.ProxyCommands.Add(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "proxy.add", "error", map[string]any{"region": cmd.Region}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "proxy.add", "ok", map[string]any{
			"region":   cmd.Region,
			"proxy_id": result["id"],
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/proxies/bulk", func(c *gin.Context) {
		if deps.ProxyCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "proxy command handler not configured"})
			return
		}
		var cmd proxycommand.BulkAddProxiesCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			auditLog(logger, "proxy.bulk_add", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.ProxyCommands.BulkAdd(c.Request.Context(), cmd)
		if err != nil {
			auditLog(logger, "proxy.bulk_add", "error", map[string]any{
				"region": cmd.Region,
				"count":  len(cmd.Proxies),
			}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "proxy.bulk_add", "ok", map[string]any{
			"region": cmd.Region,
			"count":  len(cmd.Proxies),
			"added":  result["added"],
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.DELETE("/proxies/:proxyID", func(c *gin.Context) {
		if deps.ProxyCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "proxy command handler not configured"})
			return
		}
		proxyID, err := strconv.ParseInt(c.Param("proxyID"), 10, 64)
		if err != nil {
			auditLog(logger, "proxy.delete", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proxy id"})
			return
		}
		result, err := deps.ProxyCommands.Delete(c.Request.Context(), proxycommand.DeleteProxyCommand{ProxyID: proxyID})
		if err != nil {
			auditLog(logger, "proxy.delete", "error", map[string]any{"proxy_id": proxyID}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "proxy.delete", "ok", map[string]any{"proxy_id": proxyID}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.PATCH("/proxies/:proxyID/toggle", func(c *gin.Context) {
		if deps.ProxyCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "proxy command handler not configured"})
			return
		}
		proxyID, err := strconv.ParseInt(c.Param("proxyID"), 10, 64)
		if err != nil {
			auditLog(logger, "proxy.toggle", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proxy id"})
			return
		}
		result, err := deps.ProxyCommands.Toggle(c.Request.Context(), proxycommand.ToggleProxyCommand{ProxyID: proxyID})
		if err != nil {
			auditLog(logger, "proxy.toggle", "error", map[string]any{"proxy_id": proxyID}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "proxy.toggle", "ok", map[string]any{
			"proxy_id":  proxyID,
			"is_active": result["is_active"],
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/proxies/check", func(c *gin.Context) {
		if deps.ProxyCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "proxy command handler not configured"})
			return
		}
		result, err := deps.ProxyCommands.Check(c.Request.Context(), proxycommand.CheckProxiesCommand{})
		if err != nil {
			auditLog(logger, "proxy.check", "error", nil, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "proxy.check", "ok", nil, nil)
		c.JSON(http.StatusOK, result)
	})

	router.GET("/solver/status", func(c *gin.Context) {
		if deps.GetSolverStatus == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "solver status handler not configured"})
			return
		}
		result, err := deps.GetSolverStatus.Handle(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/solver/restart", func(c *gin.Context) {
		if deps.RestartSolver == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "solver restart handler not configured"})
			return
		}
		result, err := deps.RestartSolver.Handle(c.Request.Context())
		if err != nil {
			auditLog(logger, "solver.restart", "error", nil, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "solver.restart", "ok", nil, nil)
		c.JSON(http.StatusOK, result)
	})

	router.GET("/integrations/services", func(c *gin.Context) {
		if deps.ListIntegrationServices == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration services handler not configured"})
			return
		}
		result, err := deps.ListIntegrationServices.Handle(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/integrations/services/start-all", func(c *gin.Context) {
		if deps.IntegrationCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration command handler not configured"})
			return
		}
		result, err := deps.IntegrationCommands.StartAll(c.Request.Context())
		if err != nil {
			auditLog(logger, "integration.start_all", "error", nil, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "integration.start_all", "ok", nil, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/integrations/services/stop-all", func(c *gin.Context) {
		if deps.IntegrationCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration command handler not configured"})
			return
		}
		result, err := deps.IntegrationCommands.StopAll(c.Request.Context())
		if err != nil {
			auditLog(logger, "integration.stop_all", "error", nil, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "integration.stop_all", "ok", nil, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/integrations/services/:name/start", func(c *gin.Context) {
		if deps.IntegrationCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration command handler not configured"})
			return
		}
		name := c.Param("name")
		result, err := deps.IntegrationCommands.Start(c.Request.Context(), name)
		if err != nil {
			auditLog(logger, "integration.start", "error", map[string]any{"name": name}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "integration.start", "ok", map[string]any{"name": name}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/integrations/services/:name/install", func(c *gin.Context) {
		if deps.IntegrationCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration command handler not configured"})
			return
		}
		name := c.Param("name")
		result, err := deps.IntegrationCommands.Install(c.Request.Context(), name)
		if err != nil {
			auditLog(logger, "integration.install", "error", map[string]any{"name": name}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "integration.install", "ok", map[string]any{"name": name}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/integrations/services/:name/stop", func(c *gin.Context) {
		if deps.IntegrationCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration command handler not configured"})
			return
		}
		name := c.Param("name")
		result, err := deps.IntegrationCommands.Stop(c.Request.Context(), name)
		if err != nil {
			auditLog(logger, "integration.stop", "error", map[string]any{"name": name}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "integration.stop", "ok", map[string]any{"name": name}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/integrations/backfill", func(c *gin.Context) {
		if deps.IntegrationCommands == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "integration command handler not configured"})
			return
		}
		var body struct {
			Platforms []string `json:"platforms"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			auditLog(logger, "integration.backfill", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.IntegrationCommands.Backfill(c.Request.Context(), body.Platforms)
		if err != nil {
			auditLog(logger, "integration.backfill", "error", map[string]any{"platforms": body.Platforms}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "integration.backfill", "ok", map[string]any{"platforms": body.Platforms}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/accounts/:accountID/check", func(c *gin.Context) {
		if deps.CheckAccount == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "check account handler not configured"})
			return
		}
		accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
		if err != nil {
			auditLog(logger, "account.check", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}
		result, err := deps.CheckAccount.Handle(c.Request.Context(), accountcommand.CheckAccountCommand{AccountID: accountID})
		if err != nil {
			auditLog(logger, "account.check", "error", map[string]any{"account_id": accountID}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "account.check", "ok", map[string]any{
			"account_id": accountID,
			"valid":      result.Valid,
		}, nil)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/actions/:platform/:accountID/:actionID", func(c *gin.Context) {
		if deps.ExecuteAction == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "execute action handler not configured"})
			return
		}
		accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
		if err != nil {
			auditLog(logger, "action.execute", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}
		var body struct {
			Params map[string]any `json:"params"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			auditLog(logger, "action.execute", "invalid_request", nil, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		platform := c.Param("platform")
		actionID := c.Param("actionID")
		result, err := deps.ExecuteAction.Handle(c.Request.Context(), actioncommand.ExecutePlatformActionCommand{
			Platform:  platform,
			AccountID: accountID,
			ActionID:  actionID,
			Params:    body.Params,
		})
		if err != nil {
			auditLog(logger, "action.execute", "error", map[string]any{
				"platform":   platform,
				"account_id": accountID,
				"action_id":  actionID,
			}, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		auditLog(logger, "action.execute", "ok", map[string]any{
			"platform":   platform,
			"account_id": accountID,
			"action_id":  actionID,
			"ok":         result.OK,
		}, nil)
		c.JSON(http.StatusOK, result)
	})
}

func parseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func auditLog(logger zerolog.Logger, action string, result string, fields map[string]any, err error) {
	event := logger.Info().
		Str("kind", "audit").
		Str("action", action).
		Str("result", result)
	if err != nil {
		event = event.Str("error", err.Error())
	}
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg("audit")
}
