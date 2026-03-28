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
	taskcommand "go-control-plane/internal/application/command/task"
	accountquery "go-control-plane/internal/application/query/account"
	configquery "go-control-plane/internal/application/query/config"
	platformquery "go-control-plane/internal/application/query/platform"
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

type ListAccountsHandler interface {
	Handle(context.Context, accountquery.ListAccountsQuery) (accountquery.ListAccountsResult, error)
}

type DashboardStatsHandler interface {
	Handle(context.Context) (accountquery.DashboardStatsResult, error)
}

type GetConfigHandler interface {
	Handle(context.Context) (map[string]string, error)
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

type Dependencies struct {
	ListTasks         TaskQueryHandler
	ListPlatforms     PlatformQueryHandler
	ListAccounts      ListAccountsHandler
	GetDashboardStats DashboardStatsHandler
	GetConfig         GetConfigHandler
	GetTask           GetTaskHandler
	ListTaskLogs      ListTaskLogsHandler
	ListTaskEvents    ListTaskEventsHandler
	CreateTask        CreateTaskHandler
	ApplyWorkerEvent  ApplyWorkerEventHandler
	CheckAccount      CheckAccountHandler
	ExecuteAction     ExecuteActionHandler
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
	platformRepo := sqliteadapter.NewPlatformRepository(db, cfg.Platforms)
	workerClient := workerhttp.New(cfg.Worker.BaseURL)
	applyWorkerEvent := taskcommand.NewApplyWorkerEventHandler(taskCommandRepo)

	return Dependencies{
		ListTasks:         taskquery.NewHandler(taskRepo),
		ListPlatforms:     platformquery.NewHandler(platformRepo),
		ListAccounts:      accountquery.NewListAccountsHandler(accountRepo),
		GetDashboardStats: accountquery.NewDashboardStatsHandler(accountRepo),
		GetConfig:         configquery.NewGetConfigHandler(configRepo),
		GetTask:           taskquery.NewGetTaskHandler(taskRepo),
		ListTaskLogs:      taskquery.NewListLogsHandler(taskRepo),
		ListTaskEvents:    taskquery.NewListEventsHandler(taskRepo),
		CreateTask:        taskcommand.NewHandler(taskCommandRepo, workerClient, nil, nil, cfg.Server.PublicBaseURL),
		ApplyWorkerEvent:  applyWorkerEvent,
		CheckAccount:      accountcommand.NewCheckAccountHandler(accountRepo, workerClient),
		ExecuteAction:     actioncommand.NewExecutePlatformActionHandler(accountRepo, workerClient),
	}
}

func NewRouterWithDependencies(cfg viperconfig.AppConfig, logger zerolog.Logger, deps Dependencies) *gin.Engine {
	_ = cfg
	_ = logger

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	registerPublicRoutes(router, deps)
	registerPublicRoutes(router.Group("/api"), deps)
	registerInternalWorkerRoutes(router, deps)

	return router
}

type routeRegistrar interface {
	GET(string, ...gin.HandlerFunc) gin.IRoutes
	POST(string, ...gin.HandlerFunc) gin.IRoutes
}

func registerPublicRoutes(router routeRegistrar, deps Dependencies) {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.CreateTask.Handle(c.Request.Context(), cmd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

	router.POST("/accounts/:accountID/check", func(c *gin.Context) {
		if deps.CheckAccount == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "check account handler not configured"})
			return
		}
		accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}
		result, err := deps.CheckAccount.Handle(c.Request.Context(), accountcommand.CheckAccountCommand{AccountID: accountID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/actions/:platform/:accountID/:actionID", func(c *gin.Context) {
		if deps.ExecuteAction == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "execute action handler not configured"})
			return
		}
		accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}
		var body struct {
			Params map[string]any `json:"params"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := deps.ExecuteAction.Handle(c.Request.Context(), actioncommand.ExecutePlatformActionCommand{
			Platform:  c.Param("platform"),
			AccountID: accountID,
			ActionID:  c.Param("actionID"),
			Params:    body.Params,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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
