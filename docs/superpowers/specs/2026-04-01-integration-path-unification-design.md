# Integration Path Unification Design

**Goal**

把当前已经 service 化的平台主路径继续扩展到关键旁路调用链，让 `ChatGPT` 与 `Grok` 的非插件入口路径也尽量复用已有 service，而不是继续直接调用旧底层模块。

具体目标：

1. 收口 `ChatGPT` 的关键 API 旁路调用链
2. 收口 `Grok` 的自动同步 / backfill 旁路调用链
3. 让旁路路径和主插件路径共享同一套平台 service 语义
4. 在不扩大 scope 的前提下，减少主线与旁路的实现分裂

## Scope

本轮仅覆盖：

- `api/chatgpt.py`
- `services/external_sync.py`
- `api/integrations.py`
- `platforms/chatgpt/services/*`
- `platforms/grok/services/*`
- 相关测试与文档

本轮**不覆盖**：

- `api/actions.py`
- `api/tasks.py` 的主注册链路
- `platforms/*/core.py`
- `platforms/*/switch.py`
- `services/grok2api_runtime.py` 的内部实现
- 多用户 / RBAC / PostgreSQL / Worker 扩展

## Current Problems

### 1. ChatGPT 的旁路路径还在直连旧模块

当前主插件路径已经统一到：

- `ChatGPTRegistrationService`
- `ChatGPTTokenService`
- `ChatGPTBillingService`
- `ChatGPTExternalSyncService`

但 `api/chatgpt.py` 仍然直接调用：

- `platforms.chatgpt.token_refresh`
- `platforms.chatgpt.payment`
- `platforms.chatgpt.cpa_upload`

这意味着：
- 插件主路径和 API 旁路路径仍是两套调用面
- `services/token.py` / `services/billing.py` / `services/external_sync.py` 的治理收益没有完全释放

### 2. Grok 的旁路同步链仍绕开 `GrokSyncService`

当前主插件路径已经统一到：
- `GrokSyncService.upload_grok2api`

但旁路路径仍然直接走：

- `services.external_sync.py`
- `api/integrations.py`
- `platforms.grok.grok2api_upload.upload_to_grok2api`

这导致：
- 插件 action 和自动同步 / backfill 仍有潜在漂移空间
- 后续如果 Grok sync 行为变化，要改不止一个地方

### 3. runtime 检查层和平台同步层边界还不清楚

以 Grok 为例：
- `services.grok2api_runtime.py::ensure_grok2api_ready` 是运行环境检查 / 启动辅助
- `platforms.grok.grok2api_upload.upload_to_grok2api` 是具体同步动作
- `GrokSyncService` 是插件 action service

现在的缺口不是“要不要保留 runtime 层”，而是：
- 旁路路径到底应该在哪一层停住
- 哪些责任属于 runtime 层
- 哪些责任属于平台 sync service

## Recommended Approach

采用“中等收口”方案：

- 不深改底层模块
- 不试图统一所有 API
- 只收口关键旁路调用链，使其尽量经过平台 service

## Target Structure

### ChatGPT

当前已有：
- `platforms/chatgpt/services/token.py`
- `platforms/chatgpt/services/billing.py`
- `platforms/chatgpt/services/external_sync.py`

目标：
- `api/chatgpt.py` 改为复用这些 service，而不是直接导入 legacy 模块

### Grok

当前已有：
- `platforms/grok/services/sync.py`

目标：
- `services.external_sync.py` 的 Grok 分支复用 `GrokSyncService`
- `api/integrations.py` 的 Grok backfill 分支复用 `GrokSyncService`
- `services.grok2api_runtime.py` 保留为 runtime 检查层，不并入平台 service

## Responsibility Split

### ChatGPT side

- `api/chatgpt.py`
  - 只负责 API 参数解析、DB 更新、HTTP 返回
  - 业务动作通过 service 调用
- `ChatGPTTokenService`
  - token 有效性和刷新
- `ChatGPTBillingService`
  - 支付链接生成
- `ChatGPTExternalSyncService`
  - CPA / Team Manager 同步

### Grok side

- `services.external_sync.py`
  - 仍负责“按平台路由自动同步”
  - 但 Grok 分支不再直接调用 `upload_to_grok2api`
- `api/integrations.py`
  - 仍负责 backfill API 层
  - 但 Grok 分支不再直接调用 `upload_to_grok2api`
- `services.grok2api_runtime.py`
  - 仍负责 runtime readiness / 自动启动检查
- `GrokSyncService`
  - 统一 Grok 的插件 action 路径与旁路同步动作包装

## Data Flow After Refactor

### ChatGPT API path

```text
api/chatgpt.py
  -> ChatGPTTokenService.check_valid / refresh_token
  -> ChatGPTBillingService.payment_link
  -> ChatGPTExternalSyncService.upload_cpa / upload_tm
```

### Grok auto sync path

```text
services.external_sync.sync_account
  -> ensure_grok2api_ready
  -> GrokSyncService.upload_grok2api
```

### Grok backfill path

```text
api/integrations.py
  -> ensure_grok2api_ready
  -> GrokSyncService.upload_grok2api
```

## What Stays Unchanged

### ChatGPT

本轮不改：
- `register_v2.py`
- `token_refresh.py`
- `payment.py`
- `cpa_upload.py`

### Grok

本轮不改：
- `platforms/grok/core.py`
- `services/grok2api_runtime.py` 内部实现
- grok2api runtime 的启动逻辑

## Testing Strategy

### ChatGPT
新增或补充：
- `api/chatgpt.py` 旁路调用 service 的测试
- 失败路径与 DB 更新测试

### Grok
新增或补充：
- `services.external_sync.py` 对 Grok 走 `GrokSyncService` 的测试
- `api/integrations.py` 对 Grok backfill 走 `GrokSyncService` 的测试

### Keep Existing
继续保持：
- `tests/platforms/test_grok_services.py`
- `tests/platforms/test_chatgpt_services.py`
- `tests/platforms/test_platform_contracts.py`

## Success Criteria

完成后，应满足：

- `ChatGPT` 主路径与关键 API 旁路共享同一 service 语义
- `Grok` 插件 action 路径、自动同步路径、backfill 路径共享同一 sync service 语义
- runtime readiness 层与平台 sync service 边界清晰
- 不改变现有对外 API 协议
- 不引入新的平台行为漂移

## Non-Goals

这轮不做：

- 深拆任何 `core.py / switch.py`
- 改动 Go 控制面 API 协议
- 收口所有平台所有旁路，只聚焦 `ChatGPT` 和 `Grok`
