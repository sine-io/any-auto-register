# Go Control Plane Suggested Commit Plan

当前改动面已经比较大，建议不要压成一个无差别大提交。更稳的做法是按职责拆分。

## 推荐提交粒度

### Commit 1: Python 风险收口

建议包含：

- `core/db.py`
- `core/registry.py`
- `core/base_platform.py`
- `api/accounts.py`
- `api/tasks.py`
- `api/actions.py`
- `main.py`
- `tests/`

建议提交信息：

```text
feat(python): persist task state and harden platform metadata
```

### Commit 2: Frontend 元数据与任务链路适配

建议包含：

- `frontend/src/App.tsx`
- `frontend/src/lib/registerOptions.ts`
- `frontend/src/lib/utils.ts`
- `frontend/src/pages/Accounts.tsx`
- `frontend/src/pages/Register.tsx`
- `frontend/src/pages/TaskHistory.tsx`
- `frontend/vite.config.ts`
- `frontend/.env.development`

建议提交信息：

```text
feat(frontend): route query traffic to go control plane
```

### Commit 3: Go 控制面基础骨架

建议包含：

- `go-control-plane/go.mod`
- `go-control-plane/go.sum`
- `go-control-plane/cmd/server/`
- `go-control-plane/internal/adapters/config/`
- `go-control-plane/internal/adapters/log/`
- `go-control-plane/internal/adapters/http/gin/router.go`
- 对应 Go 测试

建议提交信息：

```text
feat(go): add control plane server skeleton with gin and cobra
```

### Commit 4: Go 查询侧

建议包含：

- `go-control-plane/internal/domain/*`
- `go-control-plane/internal/application/query/*`
- `go-control-plane/internal/adapters/persistence/sqlite/*`
- `go-control-plane/internal/adapters/http/gin/router.go` 中查询路由部分
- 对应 Go 测试

建议提交信息：

```text
feat(go): add sqlite-backed query handlers for control plane
```

### Commit 5: Worker 协议与任务回调

建议包含：

- `api/worker.py`
- `docs/worker-protocol.md`
- `go-control-plane/internal/ports/outbound/worker/`
- `go-control-plane/internal/adapters/worker/http/`
- `go-control-plane/internal/application/command/task/*`
- `go-control-plane/internal/adapters/http/gin/internal_worker_task_events.go`
- `go-control-plane/internal/adapters/persistence/sqlite/task_command_repository.go`

建议提交信息：

```text
feat(worker): connect go task commands to python worker callbacks
```

### Commit 6: Go 命令转发

建议包含：

- `go-control-plane/internal/application/command/account/`
- `go-control-plane/internal/application/command/action/`
- `go-control-plane/internal/adapters/persistence/sqlite/account_repository.go`
- `go-control-plane/internal/adapters/http/gin/router.go` 中 check/action 路由

建议提交信息：

```text
feat(go): proxy account checks and platform actions through worker
```

### Commit 7: 部署与切换文档

建议包含：

- `Dockerfile`
- `go-control-plane/Dockerfile`
- `docker-compose.control-plane.yml`
- `deploy/Caddyfile`
- `README.md`
- `docs/go-control-plane-cutover-checklist.md`
- `docs/go-control-plane-commit-checklist.md`
- `docs/go-control-plane-commit-plan.md`
- `scripts/run_go_control_plane_dev.sh`

建议提交信息：

```text
docs(deploy): add go control plane dev and cutover workflow
```

## 如果你只想做一次提交

可以，但建议至少保证提交信息足够明确，例如：

```text
feat: introduce go control plane with python worker bridge
```

这种做法的缺点：

- 回滚困难
- 审查困难
- 后续定位回归成本高

## 提交前命令

建议在最终提交前统一执行：

```bash
source .venv/bin/activate
pytest tests/test_risk_hardening.py -q
python -m compileall main.py api core platforms services

cd go-control-plane
go test ./...

cd ../frontend
npm run build
```
