# Multi-User Architecture

本文档描述如果项目未来进入“多人共享 / 长期在线部署”阶段，整体架构应该如何演进。

## Current Reality

当前系统本质仍是：

- 单库
- 单控制面
- 单机优先
- Python Worker 承担大量本地自动化执行

这意味着它已经具备“可以被多人使用”的雏形，但还不具备真正多用户系统的隔离能力。

## Current Constraints

### 1. Data Is Shared By Default

当前这些对象没有用户边界：

- `accounts`
- `task_runs`
- `task_events`
- `task_logs`
- `configs`
- `proxies`

如果直接多人共用：

- 谁都能看到谁的账号
- 谁都能删别人的任务历史
- 全局配置互相污染

### 2. Control Plane Has No User Context

Go 控制面现在的路由基本都是“系统级”：

- 没有认证用户
- 没有租户上下文
- 没有请求归属

### 3. Worker Side Effects Are Global

Python Worker 当前很多动作是全局副作用：

- 浏览器环境
- Solver
- 桌面切换
- 某些平台的本地客户端重启

这天然不适合多个用户同时操作同一节点。

## Recommended Multi-User Model

### Stage 1: Single Tenant, Multi Operator

最现实的第一阶段不是多租户，而是：

- 单租户
- 多操作员
- 有账号归属和审计

目标：

- 区分“谁做了什么”
- 限制谁能改哪些资源
- 保持现有部署复杂度可控

### Stage 2: Team / Workspace Model

下一阶段建议引入：

- `workspace`
- `user`
- `membership`

然后让资源都挂在 `workspace_id` 下：

- accounts
- tasks
- logs
- configs
- proxies

### Stage 3: True Multi-Tenant Isolation

只有在产品目标明确需要时，再考虑：

- 数据租户隔离
- Worker 能力池隔离
- 配置隔离
- 计费与配额

## Proposed Domain Boundaries

### User

代表登录主体。

核心字段：

- `id`
- `email`
- `display_name`
- `status`

### Workspace

代表资源隔离域。

核心字段：

- `id`
- `name`
- `status`

### Membership

表示用户与 workspace 的关系。

核心字段：

- `user_id`
- `workspace_id`
- `role`

## Resource Ownership

如果未来进入多人模式，这些表建议全部增加 `workspace_id`：

- `accounts`
- `task_runs`
- `task_events`
- `task_logs`
- `configs`
- `proxies`

如果要做更强审计，还可以增加：

- `created_by`
- `updated_by`

## API Evolution

建议不是一次性推翻现有 API，而是按这条路径演进：

1. 先增加认证
2. 再增加 request user context
3. 再给资源加 `workspace_id`
4. 最后让 handler 强制按 workspace 过滤

## Worker Implications

多用户不是只改控制面。

Worker 也要考虑：

- 哪些任务可共享执行池
- 哪些桌面操作必须单租户独占
- 哪些平台动作需要资源锁

## Recommendation

当前建议：

- 先不要把系统定位成“正式多用户 SaaS”
- 只把未来边界写清楚
- 真要进入多人化，优先做：
  - 用户认证
  - workspace 边界
  - 资源归属
