# Trae Platform Refactor Design

**Goal**

把 `Trae` 平台从“插件入口同时承担注册编排、桌面切换、账号查询和升级链接生成”重构为“薄插件 + 明确服务边界”的结构，并且保持对外行为不变。

具体目标：

1. 不改变当前 `TraePlatform` 的外部接口
2. 让注册编排、桌面切换、账号查询、升级链接边界更清楚
3. 复用 `Cursor` 试点验证过的治理模式，为后续 `Kiro / Grok / ChatGPT` 提供模板

## Decision

采用“中等拆分”方案：

- 保留 `platforms/trae/core.py` 中的注册协议实现
- 保留 `platforms/trae/switch.py` 中的桌面副作用底层实现
- 不重写协议细节
- 将 `plugin.py` 收缩为薄入口
- 新增明确服务层

不采用“只做轻量整理”的原因：

- `get_cashier_url` 已经形成独立能力链，不适合继续塞在插件入口里
- `switch_account` 与本地 IDE 重启本身就是明显的桌面副作用边界

不采用“深度重写 core.py”的原因：

- 风险高
- 第二个试点不适合同时验证“边界拆分”和“协议重写”两件事

## Current Problems

### 1. Plugin Entry Is Too Fat

`platforms/trae/plugin.py` 当前同时负责：

- 解析配置
- 邮箱取号
- OTP 收码
- 注册编排
- action 分发
- 用户信息查询
- 升级链接生成
- action 返回包装

这让插件入口承担了过多职责。

### 2. Billing Flow Is Mixed With Platform Entry

`get_cashier_url` 当前直接在插件入口里做：

- 创建执行器
- 重新登录刷新 session
- 获取 token
- 回退到 account token
- 调用 `create_order`
- 返回前端可消费的 action result

这是一条完整的 billing 能力链，应该从插件入口独立出来。

### 3. Desktop Side Effects Are Bundled With Action Routing

`switch_account` 当前在插件入口里直接拼接：

- token 检查
- 用户信息参数组装
- 本地配置文件写入
- IDE 重启
- 消息拼接

这些更适合由桌面服务负责。

### 4. core.py Already Contains Multiple Concerns

`platforms/trae/core.py` 里目前混合了：

- 注册协议步骤
- Trae 登录刷新
- token 获取
- 订单创建

这说明 `core.py` 内部还不够纯，但这轮不处理它的深拆；这轮只先把外围编排边界理顺。

## Target Structure

建议目录演进为：

```text
platforms/trae/
  plugin.py
  core.py
  switch.py
  services/
    __init__.py
    registration.py
    account.py
    desktop.py
    billing.py
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
- 调用 `TraeRegister.register`
- 生成 `Account`

### services/account.py

负责：

- `check_valid`
- `get_user_info`

它是面向远程账户查询的服务边界。

### services/desktop.py

负责：

- `switch_account`
- `restart_ide`
- 本地桌面副作用结果包装

它是面向本地客户端切换的服务边界。

### services/billing.py

负责：

- `get_cashier_url`
- 重新登录刷新 session
- token 回退策略
- `create_order`
- 升级链接返回包装

它是 `Trae` 相对 `Cursor` 额外需要的一层边界。

## Why Keep core.py And switch.py Intact

### Why keep core.py intact

`platforms/trae/core.py` 当前主要承担：

- 发送验证码
- OTP 注册
- Trae 登录
- token 获取
- 订单创建

这些步骤已经有稳定的内部流程。当前第二个试点的重点不是重写协议本身，而是把外围编排和 action 路由拆清楚。

### Why keep switch.py intact

`platforms/trae/switch.py` 虽然耦合较高，但它本身更像“底层桌面副作用实现层”：

- 配置目录解析
- 原子写文件
- OS 判断
- 进程关闭/启动

这轮先把它放在 `desktop service` 下面消费，而不是直接继续深拆。

## Data Flow After Refactor

### Register

```text
TraePlatform.register
  -> TraeRegistrationService.register
    -> mailbox get_email / wait_for_code
    -> TraeRegister.register
    -> Account
```

### Check Valid

```text
TraePlatform.check_valid
  -> TraeAccountService.check_valid
```

### Action: switch_account

```text
TraePlatform.execute_action("switch_account")
  -> TraeDesktopService.switch_account
    -> switch_trae_account
    -> restart_trae_ide
    -> return standard action result
```

### Action: get_user_info

```text
TraePlatform.execute_action("get_user_info")
  -> TraeAccountService.get_user_info
    -> get_trae_user_info
```

### Action: get_cashier_url

```text
TraePlatform.execute_action("get_cashier_url")
  -> TraeBillingService.get_cashier_url
    -> TraeRegister.step4_trae_login
    -> TraeRegister.step5_get_token
    -> token fallback to account.token
    -> TraeRegister.step7_create_order
```

## Error Handling

保持现有外部约定：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}`

内部服务层可以：

- 抛出异常（注册编排）
- 或返回 `(ok, msg)` 再由 service / plugin 统一包装（桌面切换）

但最终仍由 `plugin.py` 对外保持稳定契约。

## Testing Strategy

### Contract Tests

继续依赖现有：

- `tests/platforms/test_platform_contracts.py`

新增 `Trae` 专项测试时优先覆盖：

- 插件入口仍返回相同 action id
- `check_valid` 行为不变
- 缺 token 错误不变
- `get_cashier_url` 的 fallback 行为不变

### Service-Level Tests

这轮建议新增：

- `tests/platforms/test_trae_services.py`

优先覆盖：

- registration service 能正确组装 OTP callback
- account service 能处理 `check_valid / get_user_info`
- desktop service 能把 `switch + restart` 结果包装成统一动作语义
- billing service 能处理 `token fallback + create_order`

## Migration Plan

### Step 1

新增：

- `services/registration.py`
- `services/account.py`
- `services/desktop.py`
- `services/billing.py`

### Step 2

让 `plugin.py` 改为调用这些 services。

### Step 3

保持 `core.py` 和 `switch.py` 继续存在，只作为底层依赖。

### Step 4

补 `Trae` 专项测试，并确认不影响现有控制面和前端调用。

## Success Criteria

完成后，应满足：

- `TraePlatform` 对外行为不变
- `plugin.py` 明显变薄
- 升级链接能力链不再直接塞在插件入口里
- 桌面副作用逻辑不再直接散落在 action 路由中
- `Trae` 可以作为 `Cursor` 之后的第二个参考实现

## Implementation Outcome

本次实现已按目标落地，当前实际状态为：

- `platforms/trae/services/` 已包含：
  - `registration.py`
  - `account.py`
  - `desktop.py`
  - `billing.py`
- `platforms/trae/plugin.py` 已收缩为薄入口，注册、账号查询、桌面切换、升级链接 action 均委托给对应 service
- `TraeRegistrationService` 已改为通过共享 helper `make_executor_from_config(config)` 创建执行器
- 注册前输出邮箱日志这一既有副作用已被保留
- 桌面重启已补出独立 service 入口 `TraeDesktopService.restart_ide()`

## Deviations Found During Implementation

实现过程中发现两点轻微偏差，需要记录：

- `TraeBillingService` 目前仍持有 `platform` 并调用 `platform._make_executor()`，执行器注入方式和 registration service 还不完全对称
- 为保留 `TraePlatform.register()` 里“先记录邮箱”的既有行为，插件层会先做一次 mailbox lookup；`TraeRegistrationService.register()` 为了完成实际注册仍会再次取 mailbox，因此存在一次轻量重复读取

这些偏差都属于小范围基础设施不对称，不影响本轮试点目标。

## Cursor Pattern Assessment

`Cursor` 的拆分模式复制到 `Trae` 总体是干净的。

复制成功的部分：

- 薄插件入口模式直接适用
- registration / account / desktop 三类 service 边界可以原样复用
- 契约测试与 service 测试可以直接承接试点验证方式

`Trae` 相对 `Cursor` 的额外差异主要只有一层：

- `billing` 能力链需要独立的 `TraeBillingService`

结论：

- `Cursor` pattern mostly copied cleanly to `Trae`
- 剩余问题仅是 billing 相关的轻微注入不对称，以及为保留日志副作用带来的 mailbox lookup 重复
- 这不构成对该模式的否定，反而说明该模式可以在存在平台特有能力链时做小幅扩展

## Non-Goals

这轮不做：

- 深拆 `platforms/trae/core.py`
- 修改 `switch.py` 的底层实现细节
- 改动 Go worker 协议
- 改动前端
