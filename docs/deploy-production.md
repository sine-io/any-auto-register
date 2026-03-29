# Production Deployment

本文档描述 `Go control plane + Python worker + Caddy gateway` 的生产部署基线。

## 1. 目标拓扑

- `gateway`
  - 对外暴露 HTTP 入口
  - 将 `/api-go/*` 转发到 Go 控制面
  - 其余请求转发到 Python Worker
- `go-control-plane`
  - 负责管理面、查询面、任务编排和 worker 回调
- `python-worker`
  - 负责平台注册执行、浏览器自动化、邮箱与验证码链路

## 2. 目录准备

建议在服务器上保留这些目录：

```text
deploy/
data/
logs/
```

- `data/` 用于 SQLite、运行时数据
- `logs/` 用于容器日志或宿主聚合日志

## 3. 环境变量

从模板开始：

```bash
cp deploy/.env.example .env
```

最关键的变量：

- `GATEWAY_PORT`
- `PYTHON_VNC_PORT`
- `APP_DB_URL`
- `AAR_DATABASE_URL`
- `APP_CORS_ALLOW_ORIGINS`
- `AAR_SERVER_PUBLIC_BASE_URL`
- `AAR_SERVER_CALLBACK_BASE_URL`
- `AAR_WORKER_BASE_URL`
- `AAR_INTERNAL_CALLBACK_TOKEN`

生产环境至少要调整：

- `APP_CORS_ALLOW_ORIGINS`
  - 不要保留 `*`
- `AAR_SERVER_PUBLIC_BASE_URL`
  - 改成真实外部访问地址
- `AAR_INTERNAL_CALLBACK_TOKEN`
  - 设置成强随机值

## 4. 启动

```bash
docker compose --env-file .env -f docker-compose.control-plane.yml up -d --build
```

首次构建会更慢，尤其 `PREFETCH_CAMOUFOX=1` 时。

## 5. 最小验收

优先执行：

```bash
bash scripts/smoke_control_plane.sh
```

如果生产端口不是默认值，先导出：

```bash
export GATEWAY_PORT=8000
export PYTHON_VNC_PORT=6080
```

然后检查：

- `/api-go/health`
- `/api-go/platforms`
- `/api-go/config`
- `/api-go/solver/status`
- `tasks/register -> tasks/{id} -> logs/stream`

## 6. 反向代理与域名

当前仓库内置的是单 Caddy 网关方案。

如果你要挂域名：

- 让公网域名指向 `gateway`
- `AAR_SERVER_PUBLIC_BASE_URL` 与域名保持一致
- `AAR_SERVER_CALLBACK_BASE_URL` 仍保持容器内可达地址，不要改成公网地址

## 7. 日志与诊断

优先使用：

```bash
docker compose -f docker-compose.control-plane.yml logs --tail=200
```

重点看：

- `go-control-plane`
- `python-worker`
- `gateway`

## 8. 已知边界

- 当前主数据库仍是 SQLite
- 当前没有正式监控/告警接入
- `python-worker` 依然承担自动化执行重任，因此宿主资源波动会直接影响注册稳定性
