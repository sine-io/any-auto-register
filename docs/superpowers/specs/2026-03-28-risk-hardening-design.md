# Risk Hardening Design

**Goal**

收掉当前项目里影响稳定性和可维护性的六类结构性风险，同时保持现有 Web/Electron 使用方式不被破坏。

**Scope**

- 修正 `trial_end_time` 未落库问题
- 修正 Docker 数据持久化路径错位
- 将任务状态和实时日志从进程内内存迁移到 SQLite
- 统一平台元数据，由后端驱动前端平台/执行器展示
- 将列表接口从全量拉取计数改为数据库计数
- 对平台相关的 Windows-only 能力暴露显式元数据，避免隐式运行时失败

**Non-goals**

- 不重写平台注册逻辑
- 不改造线程模型为队列/分布式任务系统
- 不重构外部插件安装/拉起机制的整体架构

## Current Problems

### 1. Trial expiration is not effective

`Account` 和 `AccountModel` 都有 `trial_end_time` 字段，但账号保存逻辑未写入这个字段。调度器依赖这个字段做过期判断，因此 `trial -> expired` 的自动迁移不可靠。

### 2. Docker persistence target is wrong

SQLite 默认写到仓库根目录 `account_manager.db`，而 compose 挂载的是 `/app/data`。容器重建后数据库不能按预期持久化。

### 3. Task state is memory-only

任务元信息、运行状态和 SSE 日志都依赖 `_tasks` 进程内字典。服务重启后，前端无法继续查看状态或追踪日志。

### 4. Platform metadata is split across frontend and backend

后端能动态发现平台，但前端仍硬编码平台列表、执行器支持和部分筛选项，导致插件化扩展没有真正打通。

### 5. Pagination uses full-table reads for counts

账号列表和任务历史都通过 `len(query.all())` 获取总数，会随着数据增长造成不必要的内存和查询开销。

### 6. Platform/OS constraints are implicit

部分能力只在 Windows 下工作，但 API 和前端没有清晰表达可用性，用户只能在运行时碰到失败。

## Proposed Architecture

## Data Model

保留现有表：

- `accounts`
- `task_logs`
- `proxies`
- `configs`

新增两张表：

- `task_runs`
  - `id`: 任务 ID，沿用字符串形式，如 `task_...`
  - `platform`
  - `status`: `pending | running | done | failed`
  - `progress_current`
  - `progress_total`
  - `success_count`
  - `error_count`
  - `error_summary`
  - `request_json`
  - `cashier_urls_json`
  - `created_at`
  - `updated_at`
- `task_events`
  - `id`
  - `task_id`
  - `seq`
  - `level`
  - `message`
  - `created_at`

`task_runs` 负责状态和聚合信息，`task_events` 负责实时日志与重放。

## Database Configuration

数据库路径改成环境变量优先：

- 新增 `APP_DB_URL`
- 默认仍为 `sqlite:///account_manager.db`

这样本地直接 `python main.py` 的行为保持不变；Docker/compose 显式将其设为 `sqlite:////app/data/account_manager.db`。

## Task Persistence Flow

### Creation

`/api/tasks/register` 在数据库创建一条 `task_runs` 记录，然后后台线程开始执行。

### Execution

后台执行函数更新：

- 任务状态
- 当前进度
- 成功/失败计数
- 错误摘要
- 升级链接

日志通过 `task_events` 追加写入。

### Read Path

- `/api/tasks/{id}` 从 `task_runs` 读取
- `/api/tasks` 从 `task_runs` 列表读取
- `/api/tasks/{id}/logs/stream` 从 `task_events` 增量读取

`_tasks` 不再作为真相源。可以完全移除，或仅保留为临时缓存；本次直接移除以避免双写风险。

## Platform Metadata Contract

后端 `list_platforms()` 返回扩展元数据：

- `name`
- `display_name`
- `version`
- `supported_executors`
- `available`
- `availability_reason`

其中：

- `supported_executors` 直接来自平台类
- `available/availability_reason` 用于表达系统平台限制

前端所有平台选项、执行器下拉、历史筛选都从该接口构建，不再手写平台名单。

## OS Availability Strategy

给 `BasePlatform` 增加可选可用性接口，默认全部可用：

- `is_available() -> bool`
- `get_unavailable_reason() -> str`

对于明确依赖 Windows 的平台：

- 在插件层返回不可用状态
- API 仍可列出该平台，但标明不可用原因
- 注册任务创建时做后端保护，避免启动后才失败

外部插件服务管理属于辅助能力，本次不整体重写，但平台可用性至少要在注册入口前收口。

## API Changes

### `/api/platforms`

返回完整平台元数据而非仅名称/展示名。

### `/api/tasks/*`

返回值改为数据库持久化后的结构，但保持主要字段兼容：

- `id`
- `status`
- `progress`
- `success`
- `errors`
- `cashier_urls`

### `/api/accounts`

总数改用 `COUNT(*)` 查询。

### `/api/tasks/logs`

总数改用 `COUNT(*)` 查询。

## Testing Strategy

新增 pytest 测试覆盖：

1. 账号保存时会持久化 `trial_end_time`
2. 任务创建后会写入 `task_runs`
3. 任务日志会写入 `task_events`
4. 平台列表返回 `supported_executors` 与可用性字段
5. 账号/任务历史接口的总数查询逻辑保持正确

测试使用临时 SQLite 文件并通过环境变量重定向数据库，避免污染默认数据库文件。

## Migration and Compatibility

- 仍使用 `create_all()`，不引入 Alembic
- 新表可直接创建，不影响旧数据
- 默认数据库路径不改，保证已有本地运行方式兼容
- Docker 环境通过环境变量切换到新持久化路径
- 前端消费平台元数据时保留兼容降级处理，避免后端旧响应导致空白页

## Risks and Mitigations

- 风险：任务持久化改动面较大
  - 处理：保留接口返回结构兼容，先用测试锁住行为
- 风险：平台可用性判断影响现有使用
  - 处理：只对明确 Windows-only 平台启用限制
- 风险：SSE 改为数据库轮询后延迟增加
  - 处理：保持 0.5s 轮询粒度，优先稳定性与可恢复性
