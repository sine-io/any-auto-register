# Grok Platform Refactor Design

**Goal**

把 `Grok` 平台从“插件入口同时承担注册编排、邮箱重试策略、验证码求解配置和外部同步动作”重构为“薄插件 + 明确 service 边界”的结构，并且保持对外行为不变。

具体目标：

1. 不改变当前 `GrokPlatform` 的外部接口
2. 让注册编排、cookie 轻判断、外部同步边界更清楚
3. 把邮箱域名被拒绝重试、OTP callback、captcha solver 组装从插件入口移走
4. 让 `Grok` 成为继 `Cursor / Trae / Kiro` 之后的第四个参考实现

## Decision

采用“中等拆分”方案：

- 保留 `platforms/grok/core.py` 中的浏览器自动化、Turnstile 处理、cookie 提取实现
- 不重写 `core.py` 内的浏览器流与页面细节
- 将 `plugin.py` 收缩为薄入口
- 新增明确 service 层

不采用“轻量拆分”的原因：

- `Grok` 当前最有价值的治理点在于把“邮箱重试 + OTP callback + captcha solver 组装”从插件入口下沉
- 如果只做轻量整理，`plugin.py` 仍会继续承担真正复杂的编排职责

不采用“深拆 core.py”的原因：

- `platforms/grok/core.py` 当前高度耦合浏览器自动化与 Turnstile 细节
- 第四个试点更适合继续验证编排层拆分模式，而不是进入高风险浏览器流重写

## Current Problems

### 1. Plugin Entry Is Too Fat

`platforms/grok/plugin.py` 当前同时负责：

- 读取 captcha solver 配置
- 组装 captcha solver
- 组装 `GrokRegister`
- 处理邮箱重试策略
- 处理 OTP callback
- 处理邮箱域名被拒绝后的重试决策
- action 路由与返回包装

插件入口承担了过多职责。

### 2. Registration Orchestration Is Mixed With Platform Wiring

`register()` 里当前混合了：

- 全局配置读取 (`yescaptcha_key`)
- captcha solver 创建
- mailbox attempts 策略
- mailbox 取号
- OTP callback
- 域名被拒绝后的 retry loop
- `GrokRegister.register()` 调用

这已经是一条完整的 orchestration 能力链，应该独立于插件入口。

### 3. check_valid Is Trivial But Still Belongs To A Boundary

`check_valid()` 当前只是判断 `account.extra["sso"]` 是否存在。虽然很轻，但它仍然属于 cookie/session 相关边界，不适合长期继续散落在插件入口里。

### 4. External Sync Is Still Treated As Plugin Logic

`upload_grok2api` 当前只是 `plugin.py` 里的一个 action 分支，但本质上它是：

- 外部系统同步
- 与注册浏览器流无直接关系

应当独立成 service 边界。

## Target Structure

建议目录演进为：

```text
platforms/grok/
  plugin.py
  core.py
  services/
    __init__.py
    registration.py
    cookie.py
    sync.py
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

- captcha solver 组装
- yescaptcha key 读取与 fallback
- mailbox attempts / 邮箱域名被拒绝重试
- OTP callback 组装
- 调用 `GrokRegister.register`
- 生成 `Account`

### services/cookie.py

负责：

- `check_valid`
- 与 `sso` / `sso-rw` 等 cookie/session 轻判断相关的最小逻辑

### services/sync.py

负责：

- `upload_grok2api`

它是面向外部同步动作的 service 边界。

## Why Keep core.py Intact

`platforms/grok/core.py` 当前主要承担：

- 浏览器启动
- 注册页推进
- 邮箱验证码页推进
- Turnstile widget 查找与点击
- token 注入与故障恢复
- `sso / sso-rw` cookie 提取

这些都属于高度平台特化的自动化实现层。

本轮问题不在这些步骤的内部页面逻辑，而在外围编排层和插件入口职责。因此这轮不深拆 `core.py`，只把外围 orchestration 清理出来。

## Data Flow After Refactor

### Register

```text
GrokPlatform.register
  -> GrokRegistrationService.register
    -> load captcha config
    -> build captcha solver
    -> mailbox retry loop
    -> OTP callback
    -> GrokRegister.register
    -> Account
```

### Check Valid

```text
GrokPlatform.check_valid
  -> GrokCookieService.check_valid
    -> inspect account.extra["sso"]
```

### Action: upload_grok2api

```text
GrokPlatform.execute_action("upload_grok2api")
  -> GrokSyncService.upload_grok2api
    -> upload_to_grok2api
```

## Error Handling

保持现有外部约定：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}`

并保持当前 `Grok` 的几个关键错误语义不变：

- 邮箱域名被拒绝
- Grok 注册失败
- 外部同步失败

## Testing Strategy

### Contract Tests

继续依赖现有：

- `tests/platforms/test_platform_contracts.py`

### Service-Level Tests

新增：

- `tests/platforms/test_grok_services.py`

优先覆盖：

- registration service 正确组装 OTP callback
- registration service 正确处理邮箱域名被拒绝重试
- cookie service 的 `check_valid`
- sync service 的上传结果包装
- plugin-level delegation tests（register / execute_action）

## Migration Plan

### Step 1

新增：

- `services/registration.py`
- `services/cookie.py`
- `services/sync.py`

### Step 2

先让 services 承接当前 `plugin.py` 的行为，不改 `core.py`。

### Step 3

让 `plugin.py` 改为调用这些 services，并保持 action id / 返回形状不变。

### Step 4

补 `Grok` 专项测试，并确认不影响现有控制面和前端调用。

## Success Criteria

完成后，应满足：

- `GrokPlatform` 对外接口不变
- `plugin.py` 明显变薄
- 注册编排与外部同步各自有独立边界
- `check_valid` 也有明确 service 落点
- 专项测试存在并通过
- `Grok` 可以成为第四个参考实现

## Non-Goals

这轮不做：

- 深拆 `platforms/grok/core.py`
- 重写浏览器自动化 / Turnstile 页面流
- 改动 Go worker 协议
- 改动前端
