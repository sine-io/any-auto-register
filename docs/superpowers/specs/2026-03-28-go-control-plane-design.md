# Go Control Plane + Python Worker Design

**Goal**

将当前仓库演进为 `Go 控制面 + Python 执行面` 的双后端架构：

- Go 负责 API、任务编排、配置管理、状态持久化、查询接口和实时事件输出
- Python 保留平台注册、浏览器自动化、邮箱收码、验证码求解和站点对抗逻辑

目标优先级：

1. 单机可稳定运行
2. 未来可拆成远程 worker / 多人部署
3. 不重写已有 Python 平台资产

## Decision

不建议把整个后端一次性重写为 Go。

推荐方案：

- 保留当前 Python `platforms/`、`services/`、自动化执行能力
- 新建 Go 控制面服务，逐步接管 API 和状态管理
- 通过明确协议让 Go 调用 Python worker

## Constraints

### Go 依赖约束

Go 侧统一使用：

- HTTP: `gin`
- Logging: `zerolog`
- CLI: `cobra`
- Config: `viper`

### Architecture Constraints

Go 代码风格遵循：

- `DDD Lite`
- `CQRS`
- `Clean Architecture`
- `DIP`

这里的 `DDD Lite` 指：

- 保持聚合、实体、值对象和用例边界清晰
- 不引入过重的领域建模和事件风暴流程
- 优先服务于这个仓库的工程现实，而不是形式化 DDD 完整教条

## Why Not Full Go Rewrite

当前仓库最有价值、也是最难迁移的部分并不是 HTTP API，而是：

- `platforms/` 里的平台实现
- 浏览器自动化
- 邮箱服务抽象与适配
- 验证码求解与站点细节

这些能力深度依赖 Python 生态：

- `playwright` / `patchright` / `camoufox`
- `curl_cffi`
- 已存在的平台适配代码

如果全量重写为 Go：

- 普通 CRUD / REST 层会更强
- 但执行层会被整体重写
- 回归风险与迁移成本都会显著上升

因此，Go 应该接管“控制和管理”，而不是接管“站点自动化执行”。

## Target Architecture

```text
┌────────────────────────────────────────────────────┐
│                    Frontend                         │
│      React / Vite / Ant Design / SSE or WS        │
└───────────────────────┬────────────────────────────┘
                        │
                        ▼
┌────────────────────────────────────────────────────┐
│                 Go Control Plane                    │
│ gin + zerolog + cobra + viper                      │
│                                                    │
│ - Auth / API                                       │
│ - Task orchestration                               │
│ - Config management                                │
│ - Query endpoints                                  │
│ - Event stream                                     │
│ - Persistence truth source                         │
└───────────────────────┬────────────────────────────┘
                        │ HTTP/JSON
                        ▼
┌────────────────────────────────────────────────────┐
│                 Python Worker                       │
│ FastAPI or lightweight internal service            │
│                                                    │
│ - Platform register/check/action                   │
│ - Browser automation                               │
│ - Mailbox providers                                │
│ - Captcha solving                                  │
│ - External plugin integration                      │
└────────────────────────────────────────────────────┘
```

## Responsibility Split

### Go Control Plane

负责：

- 对外 HTTP API
- 用户/系统鉴权入口
- 任务创建、取消、查询
- 任务状态持久化
- 事件日志聚合和推送
- 配置管理
- 平台元数据查询
- 分页与统计查询
- 审计和稳定性控制

不负责：

- 站点注册细节
- 浏览器自动化
- 邮箱验证码获取
- 各平台注册协议

### Python Worker

负责：

- `register`
- `check_valid`
- `execute_action`
- 临时邮箱适配
- 本地 solver / 浏览器自动化
- 站点对抗、cookie/token 提取

不负责：

- 对外用户 API
- 主数据库真相源
- 控制面鉴权
- 跨任务查询接口

## Clean Architecture Layout for Go

推荐目录：

```text
go-control-plane/
  cmd/
    server/
      main.go
    migrate/
      main.go
  internal/
    domain/
      task/
        entity.go
        repository.go
        value_objects.go
      account/
        entity.go
        repository.go
      platform/
        entity.go
      config/
        entity.go
        repository.go
    application/
      command/
        task/
          create_register_task.go
          cancel_task.go
        action/
          execute_platform_action.go
        config/
          update_config.go
      query/
        task/
          list_tasks.go
          get_task.go
        account/
          list_accounts.go
          get_dashboard_stats.go
        platform/
          list_platforms.go
    ports/
      inbound/
        httpdto/
      outbound/
        worker/
          client.go
        persistence/
          tx.go
          repositories.go
        events/
          publisher.go
    adapters/
      http/
        gin/
          router.go
          middleware/
          handlers/
      persistence/
        sqlite/
          task_repository.go
          account_repository.go
          config_repository.go
      worker/
        http/
          client.go
      config/
        viper/
          loader.go
      log/
        zerolog/
          logger.go
```

## DDD Lite Aggregate Boundaries

### Task Domain

核心聚合：

- `TaskRun`

属性：

- ID
- Platform
- Status
- Progress
- SuccessCount
- ErrorCount
- ErrorSummary
- CreatedAt / UpdatedAt

行为：

- `Start()`
- `AdvanceProgress()`
- `MarkSucceeded()`
- `MarkFailed()`
- `AppendCashierURL()`

### Account Domain

核心实体：

- `Account`

重点是持久化与查询边界，不建议在 Go 里复刻 Python 的站点业务逻辑。

### Platform Domain

核心只保留元数据：

- Name
- DisplayName
- SupportedExecutors
- Availability
- WorkerCapabilities

## CQRS Split

### Commands

- `CreateRegisterTask`
- `CancelTask`
- `ExecutePlatformAction`
- `UpdateConfig`

特点：

- 会修改状态
- 通过 repository + worker client 完成副作用

### Queries

- `ListTasks`
- `GetTask`
- `ListTaskEvents`
- `ListAccounts`
- `GetDashboardStats`
- `ListPlatforms`

特点：

- 只读
- 允许为性能单独优化 SQL

## DIP Rules

应用层依赖接口，不依赖具体实现：

- Task repository 是接口
- Worker client 是接口
- Event publisher 是接口
- Config provider 是接口

这样单元测试时可直接替换 fake/stub，而不依赖 SQLite 或 Python 进程。

## Communication Protocol

初期推荐 `HTTP + JSON`，因为最容易接入现有 Python 代码。

### Go -> Python

- `POST /worker/register`
- `POST /worker/check-account`
- `POST /worker/execute-action`
- `GET /worker/platforms`

### Python -> Go Callback

推荐让 Go 做真相源，因此 Python 在执行过程中回调 Go：

- `POST /internal/worker/tasks/{taskId}/started`
- `POST /internal/worker/tasks/{taskId}/progress`
- `POST /internal/worker/tasks/{taskId}/log`
- `POST /internal/worker/tasks/{taskId}/succeeded`
- `POST /internal/worker/tasks/{taskId}/failed`

这样：

- Go 持有最终任务状态
- 前端只需要读 Go
- Python 不直接写 Go 的主库

## Persistence Strategy

短期继续 SQLite 即可。

Go 控制面管理这些表：

- `task_runs`
- `task_events`
- `accounts`
- `configs`
- `proxies`

后续如需多人在线部署，可平滑切 PostgreSQL，但领域层和应用层不应感知数据库类型。

## Python Worker Refactor Boundary

Python 不需要再承担完整 Web UI 后端职责，可逐步收缩为 worker service。

优先保留：

- `platforms/`
- `services/turnstile_solver/`
- `core/base_mailbox.py`
- `core/base_captcha.py`

逐步移出：

- 用户查询 API
- 配置查询 API
- 仪表盘统计 API
- 主状态持久化

## Security Direction

引入 Go 控制面后，安全边界更适合放在 Go：

- API token / session auth
- 限制跨域
- 操作级鉴权
- 内部 worker 回调鉴权
- 请求审计日志

Python worker 可以只暴露内网或本机接口。

## Migration Strategy

### Phase 1

新建 Go 控制面骨架，不接管生产流量：

- `cobra` 启动
- `gin` 路由
- `zerolog` 日志
- `viper` 配置
- `/health`
- `/platforms`
- `/tasks` 查询只读接口

### Phase 2

Python 保留现状，增加 worker 专用接口。

### Phase 3

Go 接管任务创建：

- Go 创建 `task_run`
- Go 调 Python worker
- Python 回调任务事件

### Phase 4

Go 接管：

- 配置管理
- 账号查询
- 仪表盘统计
- 任务历史

### Phase 5

Go 接管动作执行入口和插件管理入口。

## What Should Not Be Migrated Early

短期不要迁移：

- `platforms/grok/core.py`
- `services/turnstile_solver/`
- 大多数 `platforms/*`
- 邮箱与验证码适配细节

这些都属于 Python 的强项区域，应该最后评估，而不是第一阶段重写。

## Success Criteria

迁移成功的标准不是“Go 代码变多”，而是：

- 前端只依赖 Go 控制面
- 任务状态和事件以 Go 为真相源
- Python 只承担执行职责
- 新平台接入仍然主要在 Python 侧完成
- 控制面可以独立扩展与部署
