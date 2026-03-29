# Post-Cutover Milestone Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在双后端已经跑通的基础上，按优先级完成安全、CI、接口收口、生产化和长期演进任务。

**Architecture:** 继续保持 `Go control plane + Python worker` 的边界不变。Go 侧负责管理面、查询面、编排面和安全边界，Python 侧继续承担平台注册、浏览器自动化、验证码与邮箱链路。后续任务优先收口风险和上线能力，而不是继续扩展功能。

**Tech Stack:** Go, gin, zerolog, cobra, viper, Python, FastAPI, SQLite, Docker Compose, Caddy, React, Vite, pytest, Go test

---

## Priority Overview

| Priority | Milestone | Outcome |
|---|---|---|
| P0 | 安全基线 | 控制面对外、Worker 回调、配置返回面达到最小可上线标准 |
| P0 | CI 与切换自动化 | 每次提交都自动验证 Python、Go、前端和 Docker 冒烟 |
| P0 | 旧接口收口 | 前端和运维路径只认 Go 控制面，Python 收缩为纯 Worker |
| P1 | Solver 稳定性 | Solver 状态、重启、日志噪音和健康检查变得可预期 |
| P1 | 生产部署基线 | 形成可复制的生产部署、回滚、配置模板 |
| P1 | 端到端回归 | 核心链路具备 E2E 验证，不再只靠手工验收 |
| P2 | 平台插件治理 | 降低 `platforms/` 目录的维护波动和隐式耦合 |
| P2 | 数据与并发演进 | 为 PostgreSQL、多 Worker、远程执行做边界准备 |
| P3 | 多用户与权限模型 | 为多人共享和长期在线部署建立用户/权限体系 |

---

### Task 1: P0 安全基线

**Files:**
- Modify: `main.py`
- Modify: `api/worker.py`
- Modify: `go-control-plane/internal/adapters/http/gin/router.go`
- Modify: `go-control-plane/internal/adapters/config/viper/loader.go`
- Modify: `go-control-plane/internal/adapters/worker/http/client.go`
- Modify: `frontend/src/lib/utils.ts`
- Create: `go-control-plane/internal/application/command/security/validate_internal_token.go`
- Create: `go-control-plane/internal/application/command/security/validate_internal_token_test.go`
- Create: `docs/security-baseline.md`
- Test: `tests/test_risk_hardening.py`
- Test: `go-control-plane/internal/adapters/http/gin/router_test.go`

- [ ] Step 1: 先补一个失败测试，覆盖“未携带内部 token 时，Go 内部回调接口拒绝请求”。
- [ ] Step 2: 在 Go 配置中新增 `internal.callback_token`，并在 `/internal/worker/tasks/*` 上强制校验。
- [ ] Step 3: 在 Python worker 的回调请求里附带 token，并补失败日志，避免 silent failure。
- [ ] Step 4: 在 `main.py` 中把 `allow_origins=["*"]` 改为环境变量驱动白名单，默认保持本地开发兼容。
- [ ] Step 5: 审查 `api/config.py`、`api/actions.py`、`api/chatgpt.py`、Go 查询接口，列出并收紧敏感字段回传面。
- [ ] Step 6: 给关键写操作补最小审计日志，至少覆盖配置更新、代理增删改、任务创建、动作执行、服务启停。
- [ ] Step 7: 运行 Python、Go、前端回归，并手工验证 token 错误时回调被拒绝。
- [ ] Step 8: 提交一次独立 commit，例如 `feat: add internal callback auth and security baseline`。

### Task 2: P0 CI 与切换自动化

**Files:**
- Create: `.github/workflows/control-plane-ci.yml`
- Create: `.github/workflows/docker-smoke.yml`
- Create: `scripts/smoke_control_plane.sh`
- Create: `scripts/smoke_python_worker.sh`
- Modify: `docs/go-control-plane-cutover-checklist.md`
- Modify: `docs/go-control-plane-commit-checklist.md`
- Modify: `README.md`

- [ ] Step 1: 写一个最小 CI workflow，顺序执行 `pytest tests/test_risk_hardening.py -q`、`python -m compileall`、`go test ./...`、`npm run build`。
- [ ] Step 2: 把已经做过的本地 Docker 冒烟操作整理成 `scripts/smoke_control_plane.sh`。
- [ ] Step 3: 写一个独立 workflow，在 CI 中用 `docker compose -f docker-compose.control-plane.yml up --build -d` 跑最小容器级检查。
- [ ] Step 4: 让脚本验证这些端点：`/api-go/health`、`/api-go/tasks/register`、`/api-go/tasks/:id`、`/api-go/tasks/:id/logs/stream`、`/api-go/solver/status`。
- [ ] Step 5: 将切换清单与提交清单收敛到自动化脚本，减少手工步骤。
- [ ] Step 6: 在 README 中明确 CI 是发布前硬门槛。
- [ ] Step 7: 提交一次独立 commit，例如 `ci: add control plane verification workflows`。

### Task 3: P0 旧 Python 管理面接口收口

**Files:**
- Modify: `main.py`
- Modify: `api/config.py`
- Modify: `api/proxies.py`
- Modify: `api/integrations.py`
- Modify: `api/accounts.py`
- Modify: `api/tasks.py`
- Modify: `frontend/src/lib/utils.ts`
- Modify: `frontend/src/pages/Settings.tsx`
- Modify: `frontend/src/pages/Proxies.tsx`
- Modify: `frontend/src/pages/Accounts.tsx`
- Modify: `frontend/src/pages/Register.tsx`
- Modify: `frontend/src/pages/TaskHistory.tsx`
- Modify: `docs/worker-protocol.md`
- Modify: `README.md`

- [ ] Step 1: 列一张接口矩阵，标记哪些 `/api/*` 已经由 Go 接管，哪些仍属于 Python Worker。
- [ ] Step 2: 对已迁移的管理接口，在 Python 侧标注为兼容层或内部接口，不再作为默认公开入口。
- [ ] Step 3: 确认前端所有读接口、任务接口、设置接口、代理接口都默认走 Go 控制面。
- [ ] Step 4: 把 Python worker 的说明文档改成“执行面 API”，避免继续被当成主 API 使用。
- [ ] Step 5: 手工验证前端在 `VITE_GO_API_BASE` 配置下，不再依赖 Python 管理面接口。
- [ ] Step 6: 补一条回归测试，验证 Go `/api` 前缀和 Python `/api/worker/*` 的边界不重叠。
- [ ] Step 7: 提交一次独立 commit，例如 `refactor: shrink python backend to worker-facing surface`。

### Task 4: P1 Solver 稳定性与健康检查

**Files:**
- Modify: `services/solver_manager.py`
- Modify: `services/turnstile_solver/start.py`
- Modify: `services/turnstile_solver/api_solver.py`
- Modify: `main.py`
- Modify: `go-control-plane/internal/application/query/system/get_solver_status.go`
- Modify: `go-control-plane/internal/application/command/system/restart_solver.go`
- Modify: `go-control-plane/internal/adapters/http/gin/router.go`
- Modify: `tests/test_risk_hardening.py`
- Test: `go-control-plane/internal/application/query/system/get_solver_status_test.go`
- Test: `go-control-plane/internal/application/command/system/restart_solver_test.go`

- [ ] Step 1: 先写失败测试，覆盖 `solver_manager.stop()` 在子进程停不下时不应直接抛出 500。
- [ ] Step 2: 把 `stop()` 改成更稳的终止流程：`terminate -> wait -> kill`，并确保日志文件句柄总能关闭。
- [ ] Step 3: 给 Solver 增加更明确的 ready 判定，区分“进程存活”和“浏览器池初始化完成”。
- [ ] Step 4: 如果需要，改 `services/turnstile_solver/start.py` 为直接使用 Hypercorn 配置，而不是依赖 `Quart.run()` 默认行为。
- [ ] Step 5: 处理 restart 后日志里的 `BaseSubprocessTransport.__del__` 噪音，至少将其控制在可接受范围内。
- [ ] Step 6: 让 Go 的 `/api/solver/status` 可以区分 `starting / running / failed`，而不只是布尔值。
- [ ] Step 7: 在 Docker 环境实测 `restart -> false -> true` 的完整恢复过程。
- [ ] Step 8: 提交一次独立 commit，例如 `fix: harden solver lifecycle and readiness`。

### Task 5: P1 生产部署基线

**Files:**
- Modify: `docker-compose.control-plane.yml`
- Modify: `deploy/Caddyfile`
- Create: `deploy/.env.example`
- Create: `docs/deploy-production.md`
- Create: `docs/rollback-plan.md`
- Modify: `README.md`

- [ ] Step 1: 把当前 compose 中的关键环境变量整理成 `.env.example`。
- [ ] Step 2: 在 `docs/deploy-production.md` 中明确公网域名、反向代理、数据卷、日志目录和端口策略。
- [ ] Step 3: 增加回滚文档，说明如何回退到单 Python 方案和如何回退到前一版本镜像。
- [ ] Step 4: 检查 Caddy 配置是否需要增加压缩、超时、头透传和日志配置。
- [ ] Step 5: 验证生产文档能在一台干净机器上复现部署。
- [ ] Step 6: 提交一次独立 commit，例如 `docs: add production deployment baseline`。

### Task 6: P1 端到端回归覆盖

**Files:**
- Create: `tests/e2e/test_control_plane_smoke.py`
- Create: `tests/e2e/test_solver_flow.py`
- Create: `tests/e2e/conftest.py`
- Modify: `scripts/smoke_control_plane.sh`
- Modify: `.github/workflows/docker-smoke.yml`

- [ ] Step 1: 定义最小 E2E 目标，只覆盖已经验证过的高价值链路，不追求全量 UI 自动化。
- [ ] Step 2: 写一个失败测试，覆盖 `register -> task detail -> logs/stream`。
- [ ] Step 3: 写一个失败测试，覆盖 `solver/status -> solver/restart -> solver/status`。
- [ ] Step 4: 用最小实现把 E2E 挂到 compose 冒烟里，避免和现有单测重复。
- [ ] Step 5: 在 CI 中只保留“最快能发现回归”的核心 E2E，不把所有平台跑进去。
- [ ] Step 6: 提交一次独立 commit，例如 `test: add control plane e2e smoke coverage`。

### Task 7: P2 平台插件治理

**Files:**
- Modify: `core/base_platform.py`
- Modify: `core/registry.py`
- Modify: `platforms/*/plugin.py`
- Modify: `platforms/*/core.py`
- Create: `docs/platform-plugin-guidelines.md`
- Create: `tests/platforms/test_platform_contracts.py`

- [ ] Step 1: 先把平台插件的共性契约写清楚，包括 availability、actions、错误返回、token 更新规则。
- [ ] Step 2: 给 `platforms/*/plugin.py` 补最小契约测试，先不深入站点逻辑。
- [ ] Step 3: 统一高频平台的错误包装和日志格式，优先从 `cursor`、`trae`、`grok` 开始。
- [ ] Step 4: 把明显的 IO/业务混杂点记录出来，作为后续分解候选。
- [ ] Step 5: 提交一次独立 commit，例如 `refactor: standardize platform plugin contracts`。

### Task 8: P2 数据与并发演进

**Files:**
- Modify: `go-control-plane/internal/adapters/persistence/sqlite/*.go`
- Modify: `core/db.py`
- Create: `docs/postgres-migration-evaluation.md`
- Create: `docs/worker-scaling-evaluation.md`

- [ ] Step 1: 先写评估文档，不直接改数据库后端。
- [ ] Step 2: 记录当前 SQLite 在任务事件、日志流、并发写上的瓶颈点。
- [ ] Step 3: 评估 PostgreSQL 迁移路径，包括 schema、连接、事务和部署成本。
- [ ] Step 4: 评估远程 Worker、多 Worker、任务队列化是否值得进入下一轮里程碑。
- [ ] Step 5: 提交一次独立 commit，例如 `docs: evaluate database and worker scaling paths`。

### Task 9: P3 多用户与权限模型

**Files:**
- Create: `docs/multi-user-architecture.md`
- Create: `docs/rbac-model.md`
- Create: `docs/secrets-management.md`

- [ ] Step 1: 先给出设计文档，而不是直接改代码。
- [ ] Step 2: 明确用户模型、角色模型、接口权限边界。
- [ ] Step 3: 评估凭据加密、审计追踪、租户隔离的复杂度。
- [ ] Step 4: 基于产品目标决定是否真的进入多人系统阶段。
- [ ] Step 5: 提交一次独立 commit，例如 `docs: define multi-user and rbac roadmap`。

---

## Recommended Execution Order

1. Task 1: P0 安全基线
2. Task 2: P0 CI 与切换自动化
3. Task 3: P0 旧 Python 管理面接口收口
4. Task 4: P1 Solver 稳定性与健康检查
5. Task 5: P1 生产部署基线
6. Task 6: P1 端到端回归覆盖
7. Task 7: P2 平台插件治理
8. Task 8: P2 数据与并发演进
9. Task 9: P3 多用户与权限模型

## Exit Criteria

- P0 完成后：系统具备最小安全边界、自动化验证和单一控制面入口。
- P1 完成后：系统具备更稳定的 Solver 与生产部署说明，且核心链路有 E2E 保障。
- P2 完成后：平台扩展和数据演进进入可规划状态，而不是继续堆隐式复杂度。
- P3 完成前：不建议把当前项目宣称为正式的多用户 SaaS。
