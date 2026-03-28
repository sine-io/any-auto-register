# Go Control Plane Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不重写 Python 执行层的前提下，引入基于 Go 的控制面，逐步接管 API、任务编排、查询和配置管理。

**Architecture:** 新建 Go 控制面项目，按 `DDD Lite + CQRS + Clean Arch + DIP` 组织代码；Python 收缩为 worker service，通过 HTTP/JSON 协议与 Go 通信；Go 成为任务状态和查询的真相源。

**Tech Stack:** Go, gin, zerolog, cobra, viper, SQLite, Python worker, HTTP/JSON

---

### Task 1: 初始化 Go 控制面项目

**Files:**
- Create: `go-control-plane/go.mod`
- Create: `go-control-plane/cmd/server/main.go`
- Create: `go-control-plane/internal/adapters/http/gin/router.go`
- Create: `go-control-plane/internal/adapters/log/zerolog/logger.go`
- Create: `go-control-plane/internal/adapters/config/viper/loader.go`

- [ ] Step 1: 初始化 `go mod`
- [ ] Step 2: 引入 `gin`、`zerolog`、`cobra`、`viper`
- [ ] Step 3: 写最小 `cobra` server 命令
- [ ] Step 4: 提供 `/health` 路由
- [ ] Step 5: 运行并确认本地服务可启动

### Task 2: 建立领域与应用层骨架

**Files:**
- Create: `go-control-plane/internal/domain/task/entity.go`
- Create: `go-control-plane/internal/domain/task/repository.go`
- Create: `go-control-plane/internal/domain/account/entity.go`
- Create: `go-control-plane/internal/domain/platform/entity.go`
- Create: `go-control-plane/internal/application/query/task/list_tasks.go`
- Create: `go-control-plane/internal/application/query/platform/list_platforms.go`

- [ ] Step 1: 定义 `TaskRun` 聚合
- [ ] Step 2: 定义 repository 接口
- [ ] Step 3: 建立 query use case 结构
- [ ] Step 4: 保证 domain/application 不依赖适配器实现

### Task 3: 实现 SQLite 只读查询链路

**Files:**
- Create: `go-control-plane/internal/adapters/persistence/sqlite/db.go`
- Create: `go-control-plane/internal/adapters/persistence/sqlite/task_repository.go`
- Create: `go-control-plane/internal/adapters/persistence/sqlite/platform_repository.go`
- Modify: `go-control-plane/internal/application/query/task/list_tasks.go`
- Modify: `go-control-plane/internal/application/query/platform/list_platforms.go`

- [ ] Step 1: 接入 SQLite
- [ ] Step 2: 实现任务分页查询
- [ ] Step 3: 实现平台元数据查询
- [ ] Step 4: 暴露 `/tasks`、`/platforms`
- [ ] Step 5: 用当前 SQLite 数据库验证接口返回

### Task 4: 给 Python 增加 worker 专用协议

**Files:**
- Create: `api/worker.py`
- Modify: `main.py`
- Create: `docs/worker-protocol.md`

- [ ] Step 1: 定义 `/worker/register`
- [ ] Step 2: 定义 `/worker/check-account`
- [ ] Step 3: 定义 `/worker/execute-action`
- [ ] Step 4: 将 worker 协议文档化

### Task 5: 实现 Go -> Python worker client

**Files:**
- Create: `go-control-plane/internal/ports/outbound/worker/client.go`
- Create: `go-control-plane/internal/adapters/worker/http/client.go`
- Create: `go-control-plane/internal/application/command/task/create_register_task.go`

- [ ] Step 1: 定义 worker client 接口
- [ ] Step 2: 实现 HTTP worker client
- [ ] Step 3: 实现 `CreateRegisterTask` command handler
- [ ] Step 4: 先以同步假实现打通调用

### Task 6: 建立任务事件回调接口

**Files:**
- Create: `go-control-plane/internal/adapters/http/gin/handlers/internal_worker_task_events.go`
- Create: `go-control-plane/internal/application/command/task/apply_worker_event.go`
- Create: `go-control-plane/internal/domain/task/events.go`

- [ ] Step 1: 定义 started/progress/log/succeeded/failed 事件 DTO
- [ ] Step 2: Go 接收回调并更新 `task_runs` / `task_events`
- [ ] Step 3: 保证 Go 是任务状态真相源

### Task 7: 迁移前端 API 指向 Go

**Files:**
- Modify: `frontend/src/lib/utils.ts`
- Modify: Go control plane 路由
- Modify: Python worker 暴露范围

- [ ] Step 1: 前端查询接口改读 Go
- [ ] Step 2: 注册任务创建改走 Go
- [ ] Step 3: Python worker 仅保留执行相关接口

### Task 8: 配置与部署整合

**Files:**
- Create: `go-control-plane/configs/app.yaml`
- Modify: `docker-compose.yml`
- Modify: `README.md`

- [ ] Step 1: 用 `viper` 建立配置加载
- [ ] Step 2: 让 compose 同时启动 Go 和 Python worker
- [ ] Step 3: 更新 README 启动说明

### Task 9: 验证与切换

**Files:**
- No code changes required

- [ ] Step 1: 验证 Go `/health`
- [ ] Step 2: 验证 Go `/tasks`、`/platforms`
- [ ] Step 3: 验证 Go 创建任务 -> Python 执行 -> Go 收到回调
- [ ] Step 4: 验证前端正常读取 Go 控制面
- [ ] Step 5: 记录仍未迁移的 Python API 并冻结边界
