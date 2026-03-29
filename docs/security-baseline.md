# Security Baseline

当前仓库的最小安全基线聚焦在四件事：

1. `Go control plane` 的内部 Worker 回调接口不能再匿名调用。
2. Python 主服务的 CORS 不能固定为全开放，必须由环境变量控制。
3. Go 控制面的关键写操作需要留下审计日志，便于排查配置变更和任务触发来源。
4. 配置读取接口不能再把 secret 明文返回给前端。

## Internal Callback Auth

- Go 内部回调入口：`/internal/worker/tasks/:taskID/*`
- 认证头：`X-AAR-Internal-Callback-Token`
- Go 环境变量：`AAR_INTERNAL_CALLBACK_TOKEN`
- Python Worker 会在回调 `started / progress / log / succeeded / failed` 时自动附带该 header

如果 `AAR_INTERNAL_CALLBACK_TOKEN` 为空，Go 侧会保持兼容模式，不强制校验。

## CORS Allowlist

- Python 环境变量：`APP_CORS_ALLOW_ORIGINS`
- 格式：逗号分隔

示例：

```text
APP_CORS_ALLOW_ORIGINS=http://localhost:5173,https://app.example.com
```

如果未设置，则默认回退到：

```text
*
```

这用于保持当前本地开发兼容；正式环境应显式配置允许来源。

## Audit Logging

Go 控制面当前已对这些关键写操作输出 `kind=audit` 的结构化日志：

- `task.register`
- `config.update`
- `proxy.add`
- `proxy.bulk_add`
- `proxy.delete`
- `proxy.toggle`
- `proxy.check`
- `solver.restart`
- `integration.start_all`
- `integration.stop_all`
- `integration.start`
- `integration.install`
- `integration.stop`
- `integration.backfill`
- `account.check`
- `action.execute`

日志只记录动作名、对象 ID、平台、变更 key、结果状态等安全元数据，不记录配置值本身。

## Secret Config Masking

- 配置读接口会对 secret 字段返回统一占位符：`********`
- 当前 Go 与 Python 两侧都会对敏感配置做掩码
- 前端 Settings 页保存时只会提交真实改动，不会把 `********` 再写回数据库

这意味着：

- 未修改 secret 时，点击保存不会覆盖原值
- 手动输入新 secret 时，会更新为新值
- 将 secret 输入框清空并保存时，会把该值更新为空字符串
