# ChatGPT Platform Refactor Design

> 状态（2026-03-31）：本设计已在 `feature/chatgpt-refactor-spec` 分支落地。本文保留原始设计意图，同时在文末追加“实际落地结果 / 实现偏差 / 试点验收结论”。

**Goal**

把 `ChatGPT` 平台从“插件入口同时承担注册编排、token 刷新、支付链接生成和外部同步动作”重构为“薄插件 + 明确 service 边界”的结构，并且保持对外行为不变。

具体目标：

1. 不改变当前 `ChatGPTPlatform` 的外部接口
2. 让注册编排、token 处理、支付链路、外部同步边界更清楚
3. 把 mailbox adapter 组装、action capability routing 从插件入口移走
4. 让 `ChatGPT` 成为继 `Cursor / Trae / Kiro / Grok` 之后的第五个参考实现

## Decision

采用“中等拆分”方案：

- 保留 `platforms/chatgpt/register_v2.py` 中的注册流程引擎实现
- 保留 `platforms/chatgpt/token_refresh.py` 中的 token 刷新实现
- 保留 `platforms/chatgpt/payment.py` 中的支付链接与订阅检查实现
- 保留 `platforms/chatgpt/cpa_upload.py` 中的 CPA / Team Manager 外部同步实现
- 将 `plugin.py` 收缩为薄入口
- 新增明确 service 层

不采用“轻量拆分”的原因：

- `ChatGPT` 当前最厚的不只是 `register()`，更是 `execute_action()` 里把 token / billing / external sync 全部堆在一起
- 如果只抽 `register()`，`plugin.py` 仍会继续承担大部分 capability routing 负担

不采用“深拆底层文件”的原因：

- `register_v2.py / payment.py / cpa_upload.py` 都已经是相对独立的底层实现文件
- 第五个试点更适合继续验证“薄插件 + services”模式，而不是进入更深层协议重构

## Current Problems

### 1. Plugin Entry Is Too Fat

`platforms/chatgpt/plugin.py` 当前同时负责：

- 解析 `RegisterConfig`
- 组装 mailbox adapter
- 读取 `register_max_retries`
- 调用 `RegistrationEngineV2`
- `check_valid`
- `refresh_token`
- `payment_link`
- `upload_cpa`
- `upload_tm`
- action 路由与结果包装

插件入口承担了过多能力。

### 2. Registration Orchestration Is Mixed With Mailbox Adaptation

`register()` 当前不仅负责编排注册，还在插件入口里直接构造：

- Generic mailbox adapter
- 默认临时邮箱 adapter
- retry 参数
- 注册入口参数回填

这条链已经足够独立，应该从插件入口中分离出去。

### 3. Token / Billing / External Sync Are Capability Families, Not Plugin Concerns

`execute_action()` 当前把三类不同性质的动作混在一起：

- token refresh
- payment link
- CPA / Team Manager 同步

这类 capability routing 更适合由 service 边界承接，而不是继续由插件入口 if/elif 维持。

### 4. check_valid Belongs With Token Logic

`check_valid()` 当前通过 `payment.check_subscription_status()` 做有效性判断。虽然逻辑不长，但它与 token / subscription 状态强相关，应该归在 token/service 边界下，而不是留在插件入口。

## Target Structure

建议目录演进为：

```text
platforms/chatgpt/
  plugin.py
  register_v2.py
  token_refresh.py
  payment.py
  cpa_upload.py
  services/
    __init__.py
    registration.py
    token.py
    billing.py
    external_sync.py
```

## Responsibility Split

### plugin.py

只负责：

- 平台元数据
- `BasePlatform` 入口实现
- service 调度
- 保持当前 action id 和返回形状不变

### services/registration.py

负责：

- mailbox adapter 组装
- `register_max_retries` 读取
- 调用 `RegistrationEngineV2`
- 返回 `Account`

### services/token.py

负责：

- `check_valid`
- `refresh_token`

它是 token / subscription 相关能力边界。

### services/billing.py

负责：

- `payment_link`
- country / plan 参数路由
- 调用 `payment.py` 中底层生成逻辑

### services/external_sync.py

负责：

- `upload_cpa`
- `upload_tm`

它是面向外部同步动作的 service 边界。

## Why Keep register_v2.py / token_refresh.py / payment.py / cpa_upload.py Intact

### Why keep register_v2.py intact

`platforms/chatgpt/register_v2.py` 已经承担：

- OAuth 注册推进
- 邮箱验证码处理
- Token 获取
- Workspace 信息提取

问题不在注册协议内部，而在插件入口外围的 adapter 与 orchestration。

### Why keep token_refresh.py intact

`platforms/chatgpt/token_refresh.py` 已经是相对独立的 token 刷新实现，本轮只把它挂到 token service 上层使用。

### Why keep payment.py intact

`platforms/chatgpt/payment.py` 已经是支付链路的独立底层实现，包括：

- 订阅状态检查
- Plus / Team 支付链接生成

本轮不深入重写。

### Why keep cpa_upload.py intact

`platforms/chatgpt/cpa_upload.py` 本身已经是外部同步底层实现，本轮只把它从 `plugin.py` 的 action 分支中抽到 `external_sync service` 上层消费。

## Data Flow After Refactor

### Register

```text
ChatGPTPlatform.register
  -> ChatGPTRegistrationService.register
    -> build mailbox adapter
    -> read retry config
    -> RegistrationEngineV2.run
    -> Account
```

### Check Valid

```text
ChatGPTPlatform.check_valid
  -> ChatGPTTokenService.check_valid
    -> payment.check_subscription_status
```

### Action: refresh_token

```text
ChatGPTPlatform.execute_action("refresh_token")
  -> ChatGPTTokenService.refresh_token
    -> TokenRefreshManager.refresh_account
```

### Action: payment_link

```text
ChatGPTPlatform.execute_action("payment_link")
  -> ChatGPTBillingService.payment_link
    -> generate_plus_link / generate_team_link
```

### Action: upload_cpa / upload_tm

```text
ChatGPTPlatform.execute_action("upload_cpa")
  -> ChatGPTExternalSyncService.upload_cpa
    -> generate_token_json
    -> upload_to_cpa

ChatGPTPlatform.execute_action("upload_tm")
  -> ChatGPTExternalSyncService.upload_tm
    -> upload_to_team_manager
```

## Error Handling

保持现有外部约定：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}`

并保持当前 `ChatGPT` 的几个关键错误语义不变：

- token 刷新失败
- 支付链接生成失败
- CPA 上传失败
- Team Manager 上传失败
- 注册失败

## Testing Strategy

### Contract Tests

继续依赖现有：

- `tests/platforms/test_platform_contracts.py`

### Service-Level Tests

新增：

- `tests/platforms/test_chatgpt_services.py`

优先覆盖：

- registration service 的 mailbox adapter 组装
- token service 的 `check_valid / refresh_token`
- billing service 的 plus / team 参数路由
- external sync service 的 `upload_cpa / upload_tm` 包装
- plugin-level delegation tests（register / execute_action）

## Migration Plan

### Step 1

新增：

- `services/registration.py`
- `services/token.py`
- `services/billing.py`
- `services/external_sync.py`

### Step 2

先让 services 承接当前 `plugin.py` 的行为，不改 `register_v2.py / token_refresh.py / payment.py / cpa_upload.py`。

### Step 3

让 `plugin.py` 改为调用这些 services，并保持 action id / 返回形状不变。

### Step 4

补 `ChatGPT` 专项测试，并确认不影响现有控制面和前端调用。

## Success Criteria

完成后，应满足：

- `ChatGPTPlatform` 对外接口不变
- `plugin.py` 明显变薄
- register / token / billing / external sync 各自有明确 service 边界
- 专项测试存在并通过
- `ChatGPT` 可以成为第五个参考实现

## Non-Goals

这轮不做：

- 深拆 `register_v2.py`
- 深拆 `token_refresh.py`
- 深拆 `payment.py`
- 深拆 `cpa_upload.py`
- 改动 Go worker 协议
- 改动前端

## Implementation Outcome

当前分支的实际落地结果是：

- 已新增 `platforms/chatgpt/services/`
  - `registration.py`
  - `token.py`
  - `billing.py`
  - `external_sync.py`
- `platforms/chatgpt/plugin.py` 已收缩为薄插件入口
  - `register()` 委托给 `ChatGPTRegistrationService`
  - `check_valid()` 委托给 `ChatGPTTokenService`
  - `execute_action("refresh_token" / "payment_link" / "upload_cpa" / "upload_tm")` 全部委托给对应 services
- `plugin.py` 的 helper factories 使用本地直接子模块导入
  - 避免在导入 `platforms.chatgpt.plugin` 时提前加载 `platforms.chatgpt.services`、`services.registration` 和 `register_v2.py`
- `ChatGPTPlatform` 的 action id、返回 envelope、默认参数路由保持不变
- 插件 action 路径之外的平行调用链保持原样
  - `api/chatgpt.py`
  - `services.external_sync.py`

## Behavior Parity Confirmed

本轮实现显式保留了以下关键语义：

- 自定义 mailbox 路径仍通过 generic mailbox adapter 组装注册流程
  - 固定 `email` 时继续复用同一个 mailbox account，不会偷偷切换邮箱
- 无 mailbox 时仍回退到 `TempMailLolMailbox`
  - `register_max_retries` 继续从 `config.extra` 读取，默认值仍为 `3`
- `check_valid()` 仍基于订阅状态判断有效性
  - `free / plus / team` 继续视为有效
  - `expired / invalid / banned / None` 继续视为无效
- `refresh_token / payment_link / upload_cpa / upload_tm` 继续暴露相同 action id，并返回统一 action envelope
- 插件导入阶段不再 eager 地拉起 services / registration path
  - 这是导入时行为收敛，不改变 `ChatGPTPlatform` 的外部接口

## Deviations Observed During Implementation

相对设计阶段与前四个参考实现，本轮记录到的偏差主要有三点：

- `platforms/chatgpt/token_refresh.py` 与 `platforms/chatgpt/payment.py` 这两个 preserved legacy 模块当前并非直接 import-safe
  - 它们仍保留对 legacy `Account` 形状的假设
  - 因此 `ChatGPTTokenService` / `ChatGPTBillingService` 通过 compatibility loader shim 注入适配后的 `Account`，而不是直接普通导入
- 为避免导入 `plugin.py` 时把注册链路整条提前拉起
  - `ChatGPTPlatform` 选择了本地 helper + 直接子模块导入
  - 而不是像 `Grok` 那样直接导入 `platforms.chatgpt.services` 包
- `ChatGPT` 的主要复杂度并不在桌面或浏览器自动化深度
  - 而在 registration / token / billing / external sync 四类 capability routing 的广度
  - 因此本轮没有复制 `Kiro` 式 token bootstrap service，也没有进入 `Grok` 式浏览器流深拆

另有一项明确的范围控制需要单独记录：

- `ChatGPTExternalSyncService` 只覆盖 `ChatGPTPlatform.execute_action()` 路径
- `api/chatgpt.py` / `services.external_sync.py` 这类平行直连调用链按计划继续保留，不在本轮迁移范围内

## Pilot Assessment

对“`Cursor / Trae / Kiro / Grok` 模式能否复制到 `ChatGPT`”的结论是：

- 可以，且整体复制仍然干净
- 这次复制不是逐文件照搬
  - `ChatGPT` 需要额外处理 legacy 模块的 import-safe 兼容问题
  - 也更强调 capability routing breadth，而不是浏览器 / 桌面副作用链深拆
- 当前剩余问题主要是：
  - legacy `payment.py / token_refresh.py` 的兼容债
  - `api/chatgpt.py` / `services.external_sync.py` 尚未纳入统一 sync service 边界

因此可以把 `ChatGPT` 视为第五个参考实现；剩余问题属于轻微实现偏差、兼容性保留与明确延期范围，而不是模式失效。

## Final Acceptance Verification (2026-03-31)

本设计在 `feature/chatgpt-refactor-spec` 分支完成文档与试点验收时，执行了以下最终验证命令：

```bash
cd /root/any-auto-register/.worktrees/chatgpt-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_chatgpt_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

结果记录如下：

- Python 验证
  - 命令：`pytest tests/platforms/test_chatgpt_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q`
  - 结果：通过，`50 passed in 4.27s`
- Go 控制面验证
  - 命令：`cd go-control-plane && go test ./...`
  - 结果：通过，exit code `0`；所有列出的 package 均返回 `ok` 或 `[no test files]`
- 前端构建验证
  - 命令：`cd ../frontend && npm run build`
  - 结果：通过，`vite build` 完成并输出 `✓ built in 1.33s`

非阻塞环境备注：

- `go test ./...` 期间打印了 `ld.so` 预加载 `/$LIB/libonion.so` 的 warning
- 该 warning 未改变命令结果，Go 验证仍然完整通过，因此只记录为环境噪音
