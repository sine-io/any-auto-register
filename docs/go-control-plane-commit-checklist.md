# Go Control Plane Commit Checklist

本清单用于当前这轮双后端迁移提交前的自检。

## 1. Python 验证

在仓库根目录执行：

```bash
source .venv/bin/activate
python -m compileall main.py api core platforms services
pytest tests/test_risk_hardening.py -q
```

通过条件：

- `compileall` 无错误
- `pytest` 全绿

## 2. Go 验证

在 `go-control-plane/` 目录执行：

```bash
go test ./...
go build -o /tmp/go-control-plane-server ./cmd/server
```

通过条件：

- `go test` 全绿
- `go build` 成功

## 3. Frontend 验证

在 `frontend/` 目录执行：

```bash
npm run build
```

通过条件：

- TypeScript 构建通过
- Vite 构建通过

## 4. 本地联调验证

优先执行：

```bash
bash scripts/smoke_python_worker.sh
```

以及：

```bash
bash scripts/smoke_control_plane.sh
```

如果自动化脚本失败，再按下面的分步命令排查。

### 启动 Python Worker

```bash
source .venv/bin/activate
PORT=8000 python main.py
```

### 启动 Go 控制面

```bash
cd go-control-plane
AAR_SERVER_PORT=8080 \
AAR_SERVER_PUBLIC_BASE_URL=http://127.0.0.1:8080 \
AAR_WORKER_BASE_URL=http://127.0.0.1:8000 \
AAR_DATABASE_URL=../account_manager.db \
go run ./cmd/server server
```

### 验收点

- `GET http://127.0.0.1:8080/health`
- `GET http://127.0.0.1:8080/api/platforms`
- `GET http://127.0.0.1:8080/api/config`
- `GET http://127.0.0.1:8080/api/accounts/stats`
- `GET http://127.0.0.1:8080/api/tasks/logs?page=1&page_size=10`

## 5. 任务链路验收

至少验证一次：

```bash
curl -X POST http://127.0.0.1:8080/api/tasks/register \
  -H 'Content-Type: application/json' \
  -d '{"platform":"dummy","count":1}'
```

然后确认：

- 请求快速返回 `task_id`
- `GET /api/tasks` 中出现新任务
- 状态从 `pending/running` 变为 `done/failed`
- `GET /api/tasks/{id}/logs/stream` 有输出

## 6. Go -> Python 命令转发验收

至少验证：

- `POST /api/accounts/{id}/check`
- `POST /api/actions/{platform}/{id}/{actionId}`

通过条件：

- Go 能正常转发到 Python Worker
- 结果与 Python 当前行为一致

## 7. Frontend 环境确认

检查以下文件存在并符合当前约定：

- [frontend/.env.development](/root/any-auto-register/frontend/.env.development)
- [frontend/vite.config.ts](/root/any-auto-register/frontend/vite.config.ts)

确认：

- `VITE_PY_API_BASE=/api`
- `VITE_GO_API_BASE=/api-go`
- `/api` 代理到 Python
- `/api-go` 代理到 Go

## 8. Docker 冒烟确认

优先执行：

```bash
bash scripts/smoke_control_plane.sh
```

通过条件：

- Go `/health` 可访问
- `tasks/register -> tasks/:id -> logs/stream` 路径可达
- `solver/status`、`config`、`platforms` 可访问

## 9. 部署文件确认

检查这些文件：

- [docker-compose.yml](/root/any-auto-register/docker-compose.yml)
- [docker-compose.control-plane.yml](/root/any-auto-register/docker-compose.control-plane.yml)
- [deploy/Caddyfile](/root/any-auto-register/deploy/Caddyfile)

确认：

- 旧单 Python 部署仍可保留
- 双后端部署文件齐全
- Caddy 路由规则与前端 `VITE_*_API_BASE` 对应

## 10. 文档确认

检查这些文档：

- [docs/worker-protocol.md](/root/any-auto-register/docs/worker-protocol.md)
- [docs/go-control-plane-cutover-checklist.md](/root/any-auto-register/docs/go-control-plane-cutover-checklist.md)
- [docs/superpowers/specs/2026-03-28-go-control-plane-design.md](/root/any-auto-register/docs/superpowers/specs/2026-03-28-go-control-plane-design.md)
- [docs/superpowers/plans/2026-03-28-go-control-plane-migration.md](/root/any-auto-register/docs/superpowers/plans/2026-03-28-go-control-plane-migration.md)

## 11. 提交判定

满足以下条件再提交：

- Python / Go / Frontend 三侧构建测试全绿
- 本地联调至少做过一轮
- 文档与实际行为一致
- 没有遗留临时调试文件、临时数据库、临时二进制
