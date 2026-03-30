# Platform Plugin Guidelines

本文档定义平台插件的最小契约，目标是让 `platforms/*/plugin.py` 的行为可预测、可测试、可由控制面稳定消费。

## Required Metadata

每个平台插件都应显式定义：

- `name`
- `display_name`
- `version`
- `supported_executors`

## Action Contract

`get_platform_actions()` 返回列表时，每个 action 至少应包含：

- `id`
- `label`
- `params`

其中：

- `id` 必须稳定，可作为 API/前端路由参数
- `label` 用于面向用户展示
- `params` 必须始终存在，即使为空也要返回 `[]`

## Execute Action Contract

`execute_action()` 应遵循统一返回形状：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}` 

不要在失败时只返回 `data.message`，否则控制面和前端需要为每个平台写特殊分支。

## Token Update Guidance

如果动作执行会刷新 token 或生成新的认证材料：

- 优先放入 `data`
- 字段名尽量使用当前系统已经追踪的名字
- 避免同一平台混用多套命名风格

## Error Handling Guidance

- 缺少关键凭据时，优先返回明确的 `error` 字段，而不是抛异常
- 未知 action 可以继续抛 `NotImplementedError`
- 外部依赖失败时，应尽量转换成可读错误字符串

## Current Priority Targets

当前已完成参考实现：

- `Cursor`
- `Trae`
- `Kiro`

下一轮优先治理建议：

- `Grok`
- `ChatGPT`

原因：

- `Grok` 的浏览器自动化、验证码与平台细节复杂度最高
- `ChatGPT` 能力面最广，插件入口仍承担较多 capability routing

## Future Cleanup Candidates

后续可以继续拆分的典型问题：

- 插件层同时承担业务决策、外部 IO、桌面客户端控制
- 动作返回值未统一，控制面不得不做兼容处理
- `platforms/*/core.py` 中协议逻辑和副作用逻辑混在一起，不利于单测

## Hotspot Inventory

下面记录当前最明显的耦合热点，目的是给后续重构排优先级，而不是要求本轮一次性拆完。

### Cursor

**当前混杂点**

- `platforms/cursor/plugin.py`
  - 同时负责：
    - 配置解析
    - 邮箱取号 / 收码
    - 注册 orchestration
    - action 结果包装
- `platforms/cursor/switch.py`
  - 同时负责：
    - token 驱动的账户切换
    - 桌面 IDE 重启
    - OS 级 `subprocess` 调用

**后续拆分候选**

- `CursorRegistrationService`
  - 只处理注册编排
- `CursorDesktopSwitcher`
  - 只处理桌面端切换与重启
- `CursorAccountAPI`
  - 只处理 `get_user_info` / `check_valid`

### Trae

**当前混杂点**

- `platforms/trae/plugin.py`
  - 注册动作、桌面切换、用户信息查询、升级链接生成都在一个插件入口里
- `platforms/trae/core.py`
  - 登录、token 获取、升级订单创建都混在同一协议实现里
- `platforms/trae/switch.py`
  - token 切换和桌面重启仍然绑在一起

**后续拆分候选**

- `TraeRegistrationService`
- `TraeBillingService`
  - 专门负责 `cashier_url`
- `TraeDesktopSwitcher`

### Grok

**当前混杂点**

- `platforms/grok/plugin.py`
  - 同时负责：
    - captcha 选择
    - 邮箱重试策略
    - 域名被拒绝时的重试决策
    - action 包装
- `platforms/grok/core.py`
  - 浏览器驱动、Turnstile 交互、页面推进、cookie 提取高度耦合
  - 还包含明显的 Windows 平台细节（如 `ctypes.windll`）

**后续拆分候选**

- `GrokRegistrationService`
  - 编排邮箱/密码/重试
- `GrokBrowserFlow`
  - 专门处理页面自动化
- `GrokAuthCookieExtractor`
  - 专门处理 `sso` / `sso-rw` 提取

### Kiro

**当前混杂点**

- `platforms/kiro/plugin.py`
  - 既做注册，又做 refresh，又做桌面切换，还做 Kiro Manager 导入
  - `switch_account` 内包含一条很长的“缺 token -> 自动补抓桌面 token -> refresh -> 切换 -> 重启”链路
- `platforms/kiro/switch.py`
  - 既做 token 刷新，也做本地桌面客户端重启
- `platforms/kiro/account_manager_upload.py`
  - 与插件 action 强耦合，但本质是外部系统同步

**后续拆分候选**

- `KiroTokenService`
  - refresh / desktop token fetch
- `KiroDesktopSwitcher`
- `KiroManagerSyncService`

### ChatGPT

**当前混杂点**

- `platforms/chatgpt/plugin.py`
  - 注册逻辑里直接内嵌 mailbox adapter 适配层
  - action 逻辑里直接拼接 CPA / Team Manager / payment / token refresh 这几类能力
- `platforms/chatgpt/*`
  - OAuth、支付、CPA 上传、token 刷新是分文件了，但插件入口仍承担了太多 capability routing

**后续拆分候选**

- `ChatGPTRegistrationService`
- `ChatGPTBillingService`
- `ChatGPTExternalSyncService`
  - CPA / Team Manager
- `ChatGPTTokenService`

## Suggested Refactor Order

剩余平台建议按这条顺序推进：

1. `Grok`
   - 浏览器自动化复杂度最高，且平台特定细节多
2. `ChatGPT`
   - 功能面最广，插件入口仍承担较多 capability routing
3. `Kiro` 深拆（如后续仍有收益）
   - 当前已完成 service 层治理，但 `core.py / switch.py` 仍可作为后续深拆候选

## Refactor Success Criteria

如果后续要继续拆分，一个平台至少应满足：

- 插件入口只做 capability 暴露和少量组装
- 注册编排、外部同步、桌面切换各自有独立服务边界
- action 结果统一遵守平台契约
- 至少有一层可脱离真实站点运行的最小单元测试

## Reference Trial: Cursor

`Cursor` 现在可以作为平台治理的第一个参考实现。

本次试点的实际结果是：

- `platforms/cursor/plugin.py`
  - 收缩为薄插件入口
- 新增 service 边界：
  - `CursorRegistrationService`
  - `CursorAccountService`
  - `CursorDesktopService`
- `core.py` 保持为协议实现层
- `switch.py` 保持为桌面副作用底层实现

这说明：

- 现有插件并不需要一次性重写
- 先拆“编排层”和“副作用边界”是可行的
- 契约测试 + service 测试足以支撑第一轮试点

## Reference Trial: Trae

`Trae` 已作为第二个参考实现完成试点，用来验证 `Cursor` 的拆分模式能否跨平台复制。

本次试点的实际结果是：

- `platforms/trae/plugin.py`
  - 收缩为薄插件入口
  - 注册、账号查询、桌面切换、升级链接能力均委托给 services
- 新增 service 边界：
  - `TraeRegistrationService`
  - `TraeAccountService`
  - `TraeDesktopService`
  - `TraeBillingService`
- 共享执行器创建统一复用 `make_executor_from_config(config)`
- 原有注册日志副作用保持不变
- 桌面 IDE 重启补出了独立 service 入口 `restart_ide()`

这说明：

- `Cursor` 的“薄插件 + services”模式可以基本原样复制到 `Trae`
- 当平台存在额外 billing 能力链时，可以新增独立 billing service，而不需要把它重新塞回插件入口
- 第二个试点继续证明：先拆编排层与副作用边界，比直接重写协议层更稳妥

## Trae Pilot Observations

`Trae` 虽然整体复制顺利，但实现中也暴露了两点小偏差，值得记录给后续平台参考：

- `TraeBillingService` 目前仍通过 `platform._make_executor()` 取执行器，基础设施注入还没有像 registration service 一样完全解耦
- `TraePlatform.register()` 为了保留“先记录邮箱日志”的既有副作用，会先做一次 mailbox lookup；而 `TraeRegistrationService.register()` 仍会自己再取一次 mailbox，形成轻微重复

结论：

- `Cursor` 模式复制到 `Trae` 基本是干净的
- 剩余问题属于轻微不对称，而不是结构性失败
- 后续复制到其他平台时，应优先统一执行器注入方式，并避免为保留日志副作用而重复读取 mailbox

## Reference Trial: Kiro

`Kiro` 已作为第三个参考实现完成试点，用来验证 `Cursor / Trae` 的“薄插件 + services”模式能否承接更长的 token / desktop / external sync 链路。

本次试点的实际结果是：

- `platforms/kiro/plugin.py`
  - 收缩为薄插件入口
  - 注册、token、桌面切换、Kiro Manager 同步均委托给 services
- 新增 service 边界：
  - `KiroRegistrationService`
  - `KiroTokenService`
  - `KiroDesktopService`
  - `KiroManagerSyncService`
- 注册阶段的 mailbox lookup 保持单次解析，并把同一个 mailbox account 继续传给 OTP 流程
- `platforms/kiro/services/__init__.py`
  - 使用 lazy import 避免插件导入阶段与 service 包产生新的 import-time coupling

这说明：

- `Cursor / Trae` 的模式复制到 `Kiro` 整体是干净的
- 但 `Kiro` 的桌面切换链路更复杂，需要单独的 token service 来吸收 desktop bootstrap / refresh 前置逻辑
- 第三个试点继续证明：优先拆编排层和副作用边界，仍比直接重写协议层更稳妥

## Kiro Pilot Observations

`Kiro` 的复制虽然总体顺利，但实现中也暴露了几处值得记录的小偏差：

- `KiroManagerSyncService` 目前仍手写简单的 action envelope，而不是直接复用 `BasePlatform` 辅助能力
- `KiroPlatform.register()` 仍保留一小段 mailbox 解析与日志职责，以保持旧版“邮箱:”日志和单次 mailbox 解析行为不变
- `Kiro` 的 token / bootstrap / desktop 分层明显比 `Cursor / Trae` 更复杂，因此把 token bootstrap 独立成 service 是必要扩展，而不是模式失效

结论：

- `Cursor / Trae` 模式复制到 `Kiro` 足够干净，可以把它视为第三个参考实现
- 剩余问题属于兼容性保留和轻微基础设施不对称
- 当前没有发现阻止后续复制到其他平台的结构性问题

