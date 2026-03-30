# Kiro Platform Refactor Design

**Goal**

把 `Kiro` 平台从“插件入口同时承担注册编排、token 刷新、桌面切换和外部同步”重构为“薄插件 + 明确 service 边界”的结构，并且保持对外行为不变。

具体目标：

1. 不改变当前 `KiroPlatform` 的外部接口
2. 让注册编排、token 处理、桌面切换、外部同步边界更清楚
3. 把 `switch_account` 里那条长链从插件入口移走
4. 让 `Kiro` 成为继 `Cursor / Trae` 之后的第三个参考实现

## Decision

采用“中等拆分”方案：

- 保留 `platforms/kiro/core.py` 中的 Builder ID 注册协议与桌面 token 抓取实现
- 保留 `platforms/kiro/switch.py` 中的 token 刷新、桌面 token 文件写入、IDE 重启逻辑
- 保留 `platforms/kiro/account_manager_upload.py` 作为外部同步底层实现
- 将 `plugin.py` 收缩为薄入口
- 新增明确服务层

不采用“轻量拆分”的原因：

- `switch_account` 当前的“缺 desktop token -> 自动补抓 -> refresh -> 切换 -> 重启”是 `Kiro` 最复杂的链路
- 如果这条链路仍留在插件入口，本次试点的治理价值会很低

不采用“深拆 core.py / switch.py”的原因：

- 风险高
- 第三个试点更适合继续验证 service 拆分模式，而不是进入协议层重写

## Current Problems

### 1. Plugin Entry Is Too Fat

`platforms/kiro/plugin.py` 当前同时负责：

- 解析 `RegisterConfig`
- 邮箱与 OTP callback 组装
- Builder ID 注册编排
- `check_valid`
- `refresh_token`
- `switch_account`
- Kiro Manager 导入
- action 结果包装

插件入口承担了过多能力。

### 2. switch_account Contains The Longest Action Chain In The Current Platform Set

`switch_account` 当前在插件入口里直接串联：

- access token / refresh token / client credentials 判定
- 缺桌面凭据时自动补抓 desktop tokens
- 可选邮箱 OTP 处理
- refresh token
- switch account
- restart IDE
- 更新返回 payload

这已经不是简单 action routing，而是一条多阶段业务链。

### 3. Token Concerns Are Mixed Across Plugin, core.py, And switch.py

当前 token 相关职责分散在：

- `plugin.py`
  - `check_valid`
  - `refresh_token`
  - `switch_account` 的 token 分支判断
- `switch.py`
  - refresh token
  - 本地 token 文件写入
  - IDE 重启
- `core.py`
  - 桌面 token 抓取

这让 `Kiro` 的维护成本明显高于 `Cursor / Trae`。

### 4. External Sync Is Still Treated As Plugin Logic

`upload_kiro_manager` 目前只是一个 action 分支，但本质上它是：

- 外部系统同步
- 与注册/桌面切换无直接关系

这类能力应该独立于插件入口。

## Target Structure

建议目录演进为：

```text
platforms/kiro/
  plugin.py
  core.py
  switch.py
  account_manager_upload.py
  services/
    __init__.py
    registration.py
    token.py
    desktop.py
    manager_sync.py
```

## Responsibility Split

### plugin.py

只负责：

- 平台元数据
- `BasePlatform` 入口实现
- 调用 service
- 保持当前 action id 和返回形状不变

### services/registration.py

负责：

- 解析 `RegisterConfig`
- 处理邮箱与 OTP callback 组装
- 调用 `KiroRegister.register`
- 生成 `Account`

### services/token.py

负责：

- `check_valid`
- `refresh_token`
- `switch_account` 前的 desktop token bootstrap 前置链

它是 `Kiro` 相对 `Cursor / Trae` 新增的核心边界。

### services/desktop.py

负责：

- `switch_account`
- `restart_ide`
- 组合 `token service` 与 `switch.py`

它是面向本地桌面副作用的服务边界。

### services/manager_sync.py

负责：

- `upload_kiro_manager`

它是面向外部系统同步的服务边界。

## Why Keep core.py / switch.py / account_manager_upload.py Intact

### Why keep core.py intact

`platforms/kiro/core.py` 当前主要承担：

- Builder ID 注册流程
- Playwright 自动化与 OTP 页面推进
- 桌面 token 抓取

它已经很重，但当前问题不在协议步骤本身，而在外围编排与边界。

### Why keep switch.py intact

`platforms/kiro/switch.py` 当前主要承担：

- refresh token
- 写入 Kiro token 文件
- IDE 重启

它更适合作为桌面副作用底层实现，而不是这轮继续深拆。

### Why keep account_manager_upload.py intact

`platforms/kiro/account_manager_upload.py` 本质上已经是一个边界较清晰的底层同步实现；本轮只把它从插件 action 分支中抽到 `manager_sync service` 上层消费。

## Data Flow After Refactor

### Register

```text
KiroPlatform.register
  -> KiroRegistrationService.register
    -> mailbox get_email / wait_for_code
    -> KiroRegister.register
    -> Account
```

### Check Valid

```text
KiroPlatform.check_valid
  -> KiroTokenService.check_valid
    -> refresh_kiro_token
```

### Action: refresh_token

```text
KiroPlatform.execute_action("refresh_token")
  -> KiroTokenService.refresh_token
    -> refresh_kiro_token
```

### Action: switch_account

```text
KiroPlatform.execute_action("switch_account")
  -> KiroDesktopService.switch_account
    -> KiroTokenService.ensure_desktop_tokens
      -> optional mailbox + OTP callback
      -> KiroRegister.fetch_desktop_tokens
    -> KiroTokenService.refresh_token
    -> switch_kiro_account
    -> restart_kiro_ide
```

### Action: upload_kiro_manager

```text
KiroPlatform.execute_action("upload_kiro_manager")
  -> KiroManagerSyncService.upload
    -> upload_to_kiro_manager
```

## Error Handling

保持现有外部约定：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}`

并保持当前 `Kiro` 的几个关键错误语义不变：

- 缺少 `accessToken`
- 缺少 `refreshToken / clientId / clientSecret`
- 自动补抓桌面 token 失败
- refresh 失败
- manager 导入失败

## Testing Strategy

### Contract Tests

继续依赖现有：

- `tests/platforms/test_platform_contracts.py`

### Service-Level Tests

新增：

- `tests/platforms/test_kiro_services.py`

优先覆盖：

- registration service 正确组装 OTP callback
- token service 的 `check_valid / refresh_token`
- token service 的 desktop token bootstrap 分支
- desktop service 的 `switch_account` 长链包装
- manager sync service 的上传结果包装

## Migration Plan

### Step 1

新增：

- `services/registration.py`
- `services/token.py`
- `services/desktop.py`
- `services/manager_sync.py`

### Step 2

先让 services 承接当前 `plugin.py` 的行为，不改 `core.py / switch.py / account_manager_upload.py`。

### Step 3

让 `plugin.py` 改为调用这些 services，并保持 action id / 返回形状不变。

### Step 4

补 `Kiro` 专项测试，并确认不影响现有控制面和前端调用。

## Success Criteria

完成后，应满足：

- `KiroPlatform` 对外接口不变
- `plugin.py` 明显变薄
- `switch_account` 的长链从插件入口下沉到 services
- token 相关职责有独立边界
- manager 导入不再和主业务 action 路由混在一起
- `Kiro` 可以成为第三个参考实现

## Non-Goals

这轮不做：

- 深拆 `platforms/kiro/core.py`
- 深拆 `platforms/kiro/switch.py`
- 改动 Go worker 协议
- 改动前端
