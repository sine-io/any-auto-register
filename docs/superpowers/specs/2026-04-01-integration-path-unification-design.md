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
- 本轮只覆盖当前已经存在的 endpoint：
  - `/refresh-token`
  - `/payment-link`
  - `/subscription`
  - `/upload-cpa`
- 本轮**不新增** `/upload-tm` API route；`upload_tm` 仍仅作为插件 action 能力存在

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
  - 额外提供可复用的原始订阅状态查询入口，供 `/subscription` route 使用
  - 保留当前每次请求可覆盖的 `proxy` 语义
- `ChatGPTBillingService`
  - 支付链接生成
  - 提供可复用的原始链接生成入口，保留 Team 路径需要的
    - `proxy`
    - `workspace_name`
    - `seat_quantity`
    - `price_interval`
- `ChatGPTExternalSyncService`
  - CPA / Team Manager 同步
  - 提供插件 action 包装入口，以及可供 API 复用的原始同步入口

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
  -> ChatGPTTokenService.refresh_account_raw / get_subscription_status_raw
  -> ChatGPTBillingService.generate_payment_link_raw
  -> ChatGPTExternalSyncService.upload_cpa_raw
```

### Grok auto sync path

```text
services.external_sync.sync_account
  -> ensure_grok2api_ready
  -> GrokSyncService.upload_grok2api_raw
```

### Grok backfill path

```text
api/integrations.py
  -> ensure_grok2api_ready
  -> GrokSyncService.upload_grok2api_raw
```

## Service Reuse Rule For Side Paths

为了避免让 API / backfill / auto-sync 调用方去手动拆插件 action envelope，本轮统一采用这条规则：

- **插件 action 路径**
  - service 暴露 `{"ok","data","error"}` 形态的方法
- **旁路调用链**
  - service 额外暴露“raw”入口，返回更贴近调用方当前契约的结果

例如：

- `ChatGPTTokenService`
  - `refresh_token(account) -> dict` 给插件 action 用
  - `refresh_account_raw(account, proxy=None) -> TokenRefreshResult` 给 API 用
  - `get_subscription_status_raw(account, proxy=None) -> str` 给 `/subscription` 用
- `ChatGPTBillingService`
  - `payment_link(...) -> dict` 给插件 action 用
  - `generate_payment_link_raw(...) -> str` 给 API 用
- `ChatGPTExternalSyncService`
  - `upload_cpa(...) -> dict` 给插件 action 用
  - `upload_cpa_raw(...) -> tuple[bool, str]` 给 API 用
- `GrokSyncService`
  - `upload_grok2api(account) -> dict` 给插件 action 用
  - `upload_grok2api_raw(account, api_url=None, app_key=None) -> tuple[bool, str]` 给 auto-sync / backfill 用

## Fallback Ownership Rule

对于 Grok 的旁路调用链，本轮明确采用：

- runtime readiness 仍由调用方负责：
  - `services.external_sync.py`
  - `api/integrations.py`
  继续先调用 `ensure_grok2api_ready()`
- `api_url / app_key` 的 fallback/default 决策继续保留在调用方现有位置
- `GrokSyncService.upload_grok2api_raw(...)` 只负责：
  - 调用 `upload_to_grok2api`
  - 统一返回 `(ok, msg)`

这样可以避免本轮把 “平台 service 收口” 扩大成 “runtime 配置策略统一”。 

进一步明确为：

### Grok auto-sync path

- `services.external_sync.py` 继续保持当前行为
- 它只负责：
  - 判断当前是否需要做 Grok 自动同步
  - 调用 `ensure_grok2api_ready()`
  - 再调用 `GrokSyncService.upload_grok2api_raw(account)`
- 在这条路径上：
  - **不显式传入** `api_url / app_key`
  - 继续依赖 `platforms.grok.grok2api_upload.upload_to_grok2api(...)` 的既有 config fallback

### Grok backfill path

- `api/integrations.py` 继续保持当前行为
- 它仍然负责：
  - 显式计算 `api_url / app_key` 的默认值
  - 再把这些值传给 `GrokSyncService.upload_grok2api_raw(account, api_url=..., app_key=...)`

这样做的原因是：

- auto-sync 当前已经依赖下层 fallback，改动它的配置决策层级收益很低
- backfill 当前已经在 API 层显式决定默认值，保留这一点最稳妥
- 两条路径都统一到了同一个 `GrokSyncService.upload_grok2api_raw(...)` 入口，但不强行统一它们原本不同的配置来源方式

## Raw Billing API Shape

为了避免 `ChatGPT` 旁路调用链实现时再发生分歧，本轮明确：

- `ChatGPTBillingService` 提供单一 raw 方法：

```python
generate_payment_link_raw(
    account,
    plan: str,
    country: str,
    proxy: str | None = None,
    workspace_name: str = "MyTeam",
    seat_quantity: int = 5,
    price_interval: str = "month",
) -> str
```

- 当 `plan == "plus"` 时，只使用：
  - `account`
  - `country`
  - `proxy`
- 当 `plan == "team"` 时，继续使用：
  - `proxy`
  - `workspace_name`
  - `seat_quantity`
  - `price_interval`

插件 action 路径仍然可以保留更薄的包装入口，但旁路 API 复用统一走这个 raw 方法。

## Raw Token API Shape

为了保留当前 `/refresh-token` 与 `/subscription` 的 per-request proxy 语义，本轮明确：

- `ChatGPTTokenService.refresh_account_raw(account, proxy=None) -> TokenRefreshResult`
- `ChatGPTTokenService.get_subscription_status_raw(account, proxy=None) -> str`

插件 action 路径继续保留较薄的包装入口：

- `refresh_token(account) -> dict`
- `check_valid(account) -> bool`

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
