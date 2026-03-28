# Go Control Plane Cutover Checklist

本清单用于验证：

- 前端读查询接口是否已成功切到 Go 控制面
- Python Worker 是否仍能承接平台注册与动作执行
- 双后端部署是否具备可切换条件

## 1. 本地开发联调

### 1.1 启动 Python Worker

```bash
source .venv/bin/activate
python main.py
```

确认：

- `http://127.0.0.1:8000/api/platforms` 可访问
- `http://127.0.0.1:8000/api/worker/register` 可被 Go 调用

### 1.2 启动 Go 控制面

```bash
cd go-control-plane
AAR_SERVER_PORT=8080 \
AAR_SERVER_PUBLIC_BASE_URL=http://127.0.0.1:8080 \
AAR_WORKER_BASE_URL=http://127.0.0.1:8000 \
AAR_DATABASE_URL=../account_manager.db \
go run ./cmd/server server
```

确认：

- `http://127.0.0.1:8080/health`
- `http://127.0.0.1:8080/api/platforms`
- `http://127.0.0.1:8080/api/config`

### 1.3 启动前端

```bash
cd frontend
npm run dev
```

确认：

- `http://127.0.0.1:5173` 可访问
- `frontend/.env.development` 中：
  - `VITE_PY_API_BASE=/api`
  - `VITE_GO_API_BASE=/api-go`
  - `VITE_PY_PROXY_TARGET=http://127.0.0.1:8000`
  - `VITE_GO_PROXY_TARGET=http://127.0.0.1:8080`

## 2. 前端查询切换验证

打开浏览器网络面板，确认以下 GET 请求命中 `/api-go`：

- 平台列表：`/api-go/platforms`
- 配置读取：`/api-go/config`
- 账号列表：`/api-go/accounts`
- 仪表盘统计：`/api-go/accounts/stats`
- 任务历史：`/api-go/tasks/logs`
- 任务详情：`/api-go/tasks/{taskId}`
- 任务日志流：`/api-go/tasks/{taskId}/logs/stream`

确认以下写请求仍然正常完成：

- `POST /api-go/tasks/register`
- `POST /api/tasks/register` 不应再被前端使用

## 3. 功能冒烟

### 3.1 创建注册任务

步骤：

1. 进入“注册任务”
2. 创建一个最小单账号任务
3. 观察请求返回是否快速拿到 `task_id`
4. 观察详情页状态从 `pending/running` 更新到 `done/failed`

确认：

- 任务详情来自 Go
- worker 回调能更新 Go 的状态
- 日志流中能看到至少一条执行日志

### 3.2 账号校验

步骤：

1. 选一个已有账号
2. 执行“检查账号”动作

确认：

- 请求走 Go 控制面
- Go 再调 Python Worker
- 结果能正常返回

### 3.3 平台动作

步骤：

1. 选一个支持动作的平台账号
2. 执行动作

确认：

- 动作可用性元数据正常显示
- 不可用动作仍保持禁用
- 可用动作可以由 Go 转发到 Python Worker 执行

## 4. 数据一致性

检查 SQLite 中这些表：

- `task_runs`
- `task_events`
- `accounts`
- `configs`

确认：

- 新任务写入 `task_runs`
- worker 回调写入 `task_events`
- 查询接口与数据库内容一致

## 5. 双后端 Compose 验收

环境具备 Docker 后执行：

```bash
docker compose -f docker-compose.control-plane.yml up --build
```

确认：

- `gateway` 暴露 `8000`
- `go-control-plane` 可访问 `8080`
- `python-worker` 可访问 `8000`（容器内）
- 打开 `http://localhost:8000`
- 前端页面能正常加载与操作

## 6. 切换判定

满足以下条件可视为“查询与任务链路已切换到 Go”：

- 前端关键 GET 查询全部命中 Go
- 任务创建由 Go 发起并持久化
- worker 回调能更新 Go 的任务状态和事件
- 账号校验/动作执行由 Go 转发且结果正确
- 现有平台注册功能未回退

## 7. 未完成项

当前即使本清单通过，仍不代表全迁完成。剩余工作包括：

- `Settings / Proxies / Integrations` 全量迁到 Go
- 生产环境反向代理与鉴权强化
- Worker 回调鉴权
- 容器级监控与健康检查
