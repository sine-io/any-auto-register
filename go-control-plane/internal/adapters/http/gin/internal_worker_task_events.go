package gingateway

import (
	"net/http"

	securitycommand "go-control-plane/internal/application/command/security"
	taskcommand "go-control-plane/internal/application/command/task"
	domaintask "go-control-plane/internal/domain/task"

	"github.com/gin-gonic/gin"
)

type workerProgressRequest struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

type workerLogRequest struct {
	Message string `json:"message"`
}

type workerResultRequest struct {
	SuccessCount int      `json:"success_count"`
	ErrorCount   int      `json:"error_count"`
	Error        string   `json:"error"`
	Errors       []string `json:"errors"`
	CashierURLs  []string `json:"cashier_urls"`
}

func registerInternalWorkerRoutes(router *gin.Engine, deps Dependencies, expectedToken string) {
	group := router.Group("/internal/worker/tasks/:taskID")
	group.Use(func(c *gin.Context) {
		if err := securitycommand.ValidateInternalCallbackToken(
			expectedToken,
			c.GetHeader(securitycommand.InternalCallbackTokenHeader),
		); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Next()
	})

	group.POST("/started", func(c *gin.Context) {
		handleWorkerEvent(c, deps, taskcommand.ApplyWorkerEventCommand{
			TaskID: c.Param("taskID"),
			Type:   domaintask.WorkerEventStarted,
		})
	})

	group.POST("/progress", func(c *gin.Context) {
		var req workerProgressRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		handleWorkerEvent(c, deps, taskcommand.ApplyWorkerEventCommand{
			TaskID:          c.Param("taskID"),
			Type:            domaintask.WorkerEventProgress,
			ProgressCurrent: req.Current,
			ProgressTotal:   req.Total,
		})
	})

	group.POST("/log", func(c *gin.Context) {
		var req workerLogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		handleWorkerEvent(c, deps, taskcommand.ApplyWorkerEventCommand{
			TaskID:  c.Param("taskID"),
			Type:    domaintask.WorkerEventLog,
			Message: req.Message,
		})
	})

	group.POST("/succeeded", func(c *gin.Context) {
		var req workerResultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		handleWorkerEvent(c, deps, taskcommand.ApplyWorkerEventCommand{
			TaskID:       c.Param("taskID"),
			Type:         domaintask.WorkerEventSucceeded,
			SuccessCount: req.SuccessCount,
			ErrorCount:   req.ErrorCount,
			ErrorSummary: req.Error,
			Errors:       req.Errors,
			CashierURLs:  req.CashierURLs,
		})
	})

	group.POST("/failed", func(c *gin.Context) {
		var req workerResultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		handleWorkerEvent(c, deps, taskcommand.ApplyWorkerEventCommand{
			TaskID:       c.Param("taskID"),
			Type:         domaintask.WorkerEventFailed,
			SuccessCount: req.SuccessCount,
			ErrorCount:   req.ErrorCount,
			ErrorSummary: req.Error,
			Errors:       req.Errors,
			CashierURLs:  req.CashierURLs,
		})
	})
}

func handleWorkerEvent(c *gin.Context, deps Dependencies, cmd taskcommand.ApplyWorkerEventCommand) {
	if deps.ApplyWorkerEvent == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "worker event handler not configured"})
		return
	}
	if err := deps.ApplyWorkerEvent.Handle(c.Request.Context(), cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
