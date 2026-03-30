# PostgreSQL Migration Evaluation

本文档评估当前项目是否值得从 SQLite 迁移到 PostgreSQL，以及迁移的触发条件、收益和代价。

## Current State

当前项目主要使用两套持久化入口：

- Python 侧：`core/db.py` + SQLModel
- Go 控制面：`go-control-plane/internal/adapters/persistence/sqlite/*.go`

当前数据表重点包括：

- `accounts`
- `task_logs`
- `task_runs`
- `task_events`
- `configs`
- `proxies`

## Why SQLite Still Works Today

在当前项目形态下，SQLite 仍然合理：

- 主要部署模式仍是单机
- 主数据库访问仍集中在单节点
- Go 控制面和 Python Worker 当前默认共享本地 DB 文件
- 日志和任务事件虽然频繁写，但总体仍是“本地工具级别”负载

## Current Bottlenecks

### 1. Task Event Writes

虽然已经做了批量刷盘，但 `task_events` 依然是高频写热点。

问题不在 SQL 复杂度，而在：

- 单文件数据库写锁
- Python 与 Go 共用同一 SQLite 文件
- Docker 环境下 volume I/O 波动

### 2. Shared Access From Two Runtimes

当前 Python 和 Go 都会直接打 SQLite。

这在单机场景能跑，但会放大这些问题：

- 写锁竞争
- 事务等待不可见
- 一端卡住时另一端更难定位

### 3. Batch Deletion / Import / Task History Growth

随着 `accounts`、`task_logs`、`task_events` 继续增长：

- 导入/批量删除会拉高写压力
- 历史任务查询和清理会越来越依赖表扫描与索引质量

## What PostgreSQL Would Improve

如果迁移到 PostgreSQL，最直接的收益是：

- 更稳的并发写
- 更明确的事务和锁行为
- 更适合 Go / Python 同时访问
- 更容易接入远程部署、多实例和后续多用户模型

## What PostgreSQL Would Not Automatically Solve

迁移 PostgreSQL 并不会自动解决：

- 平台插件层的 IO/业务耦合
- Solver / 浏览器自动化本身的资源波动
- Worker 回调时序问题
- E2E/CI 稳定性问题

也就是说，数据库迁移不是当前最优先的性能按钮。

## Migration Cost

### Python Side

需要处理：

- `core/db.py` 的 engine 初始化
- SQLModel/Session 配置
- 可能的 Alembic 或迁移工具引入

### Go Side

需要处理：

- SQLite 适配器替换
- SQL 方言差异
- `datetime(...)` 排序等 SQLite 专用语法
- 驱动切换和 DSN 管理

### Deployment Side

需要新增：

- PostgreSQL 服务
- 备份与恢复策略
- 凭据管理
- 连接池配置

## Trigger Conditions

建议只有满足以下条件之一，再推进 PostgreSQL：

1. Go 控制面与 Python Worker 明显出现 DB 锁竞争
2. `task_events` 写入量让 SQLite 成为真实瓶颈
3. 要做远程 Worker / 多节点控制面
4. 要进入多用户/权限模型阶段

## Suggested Path

推荐顺序不是“立刻迁”，而是：

1. 先继续保持 SQLite
2. 记录真实瓶颈数据
3. 先补 schema migration 机制
4. 再做 PostgreSQL 分支验证

## Recommendation

当前建议：

- **短期：继续用 SQLite**
- **中期：引入迁移机制**
- **长期：在多实例 / 多用户 / 高并发写成为真实需求时，再切 PostgreSQL**
