# Cursor Platform Refactor Design

**Goal**

把 `Cursor` 平台从“插件入口同时承担注册编排、桌面切换和账号查询”重构为“薄插件 + 明确服务边界”的结构，并且保持对外行为不变。

具体目标：

1. 不改变当前 `CursorPlatform` 的外部接口
2. 让注册编排、桌面切换、账号查询边界更清楚
3. 为 `Trae / Grok / Kiro / ChatGPT` 的后续治理提供模板

## Decision

采用“中等拆分”方案：

- 保留 `platforms/cursor/core.py` 中的注册协议实现
- 不重写协议细节
- 将 `plugin.py` 收缩为薄入口
- 新增明确服务层

不采用“只做轻量整理”的原因：

- `switch.py` 里的桌面切换和本地 IDE 重启本身就是高耦合热点

不采用“深度重写 core.py”的原因：

- 风险高
- 试点收益不匹配

## Current Problems

### 1. Plugin Entry Is Too Fat

`platforms/cursor/plugin.py` 当前同时负责：

- 解析配置
- 邮箱取号
- OTP 收码
- 注册编排
- action 分发
- action 返回包装

这让插件入口承担了过多职责。

### 2. Desktop Side Effects Are Mixed With Platform Logic

`platforms/cursor/switch.py` 里同时做：

- 读取本地配置路径
- 原子写入 `storage.json`
- 平台判断
- 桌面 IDE 关闭/重启
- 远程用户信息查询

这些逻辑从“账户切换”到“本地桌面控制”跨度很大。

### 3. Contract Is Stable But Internals Are Not

当前控制面已经依赖这些稳定契约：

- `get_platform_actions()`
- `execute_action()`
- `check_valid()`

所以试点的关键不是改外部接口，而是稳定内部边界。

## Target Structure

建议目录演进为：

```text
platforms/cursor/
  plugin.py
  core.py
  switch.py           # 暂时保留，逐步收缩
  services/
    __init__.py
    registration.py
    desktop.py
    account.py
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
- 处理邮箱与 OTP 回调组装
- 调用 `CursorRegister`
- 生成 `Account`

### services/desktop.py

负责：

- `switch_account`
- `restart_ide`

它是面向本地桌面副作用的服务边界。

### services/account.py

负责：

- `check_valid`
- `get_user_info`

它是面向远程账户查询的服务边界。

## Why Keep core.py Intact

`platforms/cursor/core.py` 目前主要承担 Cursor 注册协议实现：

- session 获取
- 邮箱提交
- 密码提交
- OTP 提交
- token 提取

这些步骤已经有稳定的内部流程，当前问题不在协议步骤本身，而在外围编排和服务边界。

因此这轮不重构 `core.py`，只把它留在“协议实现层”。

## Data Flow After Refactor

### Register

```text
CursorPlatform.register
  -> CursorRegistrationService.register
    -> mailbox get_email / wait_for_code
    -> CursorRegister.register
    -> Account
```

### Check Valid

```text
CursorPlatform.check_valid
  -> CursorAccountService.check_valid
    -> curl_cffi GET /api/auth/me
```

### Action: switch_account

```text
CursorPlatform.execute_action("switch_account")
  -> CursorDesktopService.switch_account
    -> write storage.json
    -> restart Cursor IDE
    -> return standard action result
```

### Action: get_user_info

```text
CursorPlatform.execute_action("get_user_info")
  -> CursorAccountService.get_user_info
    -> curl_cffi GET /api/auth/me
```

## Error Handling

保持现有外部约定：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}`

内部服务层允许：

- 抛出异常（注册编排）
- 或返回 `(ok, msg)` 再由插件统一包装（桌面控制）

但最终由 `plugin.py` 对外统一结果。

## Testing Strategy

### Contract Tests

继续依赖现有：

- `tests/platforms/test_platform_contracts.py`

新增 Cursor 专项测试时优先覆盖：

- 插件入口仍返回相同 action id
- 缺 token 错误不变
- 注册路径仍能正确构造 `Account`

### Service-Level Tests

这轮建议新增最小单元测试：

- `tests/platforms/test_cursor_services.py`

覆盖：

- registration service 能正确组装 OTP callback
- desktop service 能把 `switch + restart` 结果包装为统一动作语义
- account service 能处理 `check_valid / get_user_info`

## Migration Plan

### Step 1

新增 `services/registration.py`、`services/desktop.py`、`services/account.py`

### Step 2

让 `plugin.py` 改为调用这些 service

### Step 3

保持 `switch.py` 和 `core.py` 不删，只作为底层依赖

### Step 4

补 Cursor 专项测试

## Success Criteria

完成后，应满足：

- `CursorPlatform` 对外行为不变
- `plugin.py` 明显变薄
- 桌面副作用逻辑不再直接散落在插件入口
- Cursor 可作为后续 `Trae` 试点的模板

## Non-Goals

这轮不做：

- 重写 `core.py`
- 改动 Go 控制面协议
- 改变前端动作入口
- 推进所有平台同时拆分
