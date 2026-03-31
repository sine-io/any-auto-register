# ChatGPT Platform Refactor Design

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
