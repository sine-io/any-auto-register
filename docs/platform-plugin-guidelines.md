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
- `Grok`
- `ChatGPT`

下一轮优先治理建议：

- `Kiro` 深拆（如后续仍有收益）
- `Grok` 深拆（仅在后续仍确认有收益时）
- `ChatGPT` 兼容债 / 平行调用链治理（仅在后续仍确认值得时）

原因：

- `Kiro` 已完成 service 层治理，但 `core.py / switch.py` 仍有继续降耦空间
- `Grok` 已完成第四个试点，剩余问题主要集中在 `core.py` 深拆与外部同步总线统一
- `ChatGPT` 已完成第五个试点，插件入口已变薄；剩余问题主要是 legacy `payment.py / token_refresh.py` 的 import-safe 兼容债，以及 `api/chatgpt.py / services.external_sync.py` 这类平行直连路径尚未统一

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

**当前状态 / 剩余耦合点**

- `platforms/grok/plugin.py`
  - 已收缩为薄插件入口
  - `register / check_valid / upload_grok2api` 已委托给 services
  - 当前直接 `from platforms.grok.services import ...`，导入时比 `Kiro` 的按需加载略更 eager
- `platforms/grok/services/registration.py`
  - 已承接 captcha 组装、邮箱重试、OTP callback、`Account` 构造
  - 显式保持“固定 email 不轮换 mailbox”和“无 mailbox 时保留手工 OTP fallback”语义
- `platforms/grok/services/sync.py`
  - 只承接插件 action 路径的 `upload_grok2api`
  - `services.external_sync.py` / `api.integrations.py` / `services.grok2api_runtime.py` 仍是独立外部同步调用链
- `platforms/grok/core.py`
  - 仍同时负责浏览器驱动、Turnstile 交互、页面推进、cookie 提取
  - 仍包含明显的 Windows 平台细节（如 `ctypes.windll`）

**后续清理候选**

- 继续深拆 `platforms/grok/core.py`
  - 将浏览器流、Turnstile 处理、cookie 提取进一步分离
- 如后续需要，再统一 `platforms/grok/services/__init__.py` / `plugin.py` 的按需导入
  - 让 import-time coupling 更接近 `Kiro` 模式
- 如果未来要治理外部集成总线，再把 `services.external_sync.py` / `api.integrations.py` / `services.grok2api_runtime.py` 纳入统一 sync service 边界

### Kiro

**当前状态 / 剩余耦合点**

- `platforms/kiro/plugin.py`
  - 已收缩为薄插件入口
  - 仍保留少量 mailbox lookup / 兼容日志职责
  - 通过本地 helper import 按需加载具体 service
- `platforms/kiro/services/token.py` + `platforms/kiro/services/desktop.py`
  - 已承接 desktop token bootstrap / refresh / switch orchestration 主链路
  - 但这条桌面切换链仍需要跨 service 协调 `core.py` 与 `switch.py`，复杂度仍高于 `Cursor / Trae`
- `platforms/kiro/switch.py`
  - 仍同时负责 token 刷新、本地 token 文件写入、桌面客户端重启
- `platforms/kiro/account_manager_upload.py`
  - 已从插件 action 路由中抽离
  - 但底层仍是独立外部同步副作用实现

**后续清理候选**

- 继续深拆 `platforms/kiro/switch.py`
  - 将 refresh、token 文件写入、IDE 重启进一步解耦
- 继续深拆 `platforms/kiro/core.py`
  - 将 desktop token bootstrap 与注册浏览器流进一步分离
- 为 `manager_sync` 补统一 action/result helper
  - 减少各平台手写 envelope 的轻微不对称

### ChatGPT

**当前状态 / 剩余耦合点**

- `platforms/chatgpt/plugin.py`
  - 已收缩为薄插件入口
  - `register / check_valid / refresh_token / payment_link / upload_cpa / upload_tm` 已委托给 services
  - helper factories 使用本地直接子模块导入，避免在插件导入阶段提前拉起 `platforms.chatgpt.services` 与 `register_v2.py` 注册路径
- `platforms/chatgpt/services/registration.py`
  - 已承接 mailbox adapter 组装、`register_max_retries` 读取、`Account` 构造
  - 保留固定 `email` + 自定义 mailbox 与默认 tempmail fallback 的既有语义
- `platforms/chatgpt/services/token.py` + `platforms/chatgpt/services/billing.py`
  - 已承接 `check_valid / refresh_token / payment_link`
  - 为兼容 preserved legacy `token_refresh.py / payment.py` 当前并非直接 import-safe 的现实，service 层保留了 compatibility loader shims
- `platforms/chatgpt/services/external_sync.py`
  - 只承接 `ChatGPTPlatform.execute_action()` 路径上的 `upload_cpa / upload_tm`
  - `api/chatgpt.py` / `services.external_sync.py` 仍是平行直连调用链
- `platforms/chatgpt/*`
  - 当前剩余复杂度主要来自 capability routing breadth 与 legacy 模块兼容债，而不是 `Kiro` 式桌面链路或 `Grok` 式浏览器页面流深度

**后续清理候选**

- 修复 `payment.py` / `token_refresh.py` 的 import-safe 缺陷
  - 让 `token / billing` services 后续有机会移除 compatibility loader shims
- 如后续要统一外部同步边界，再把 `api/chatgpt.py` / `services.external_sync.py` 纳入同一 sync service 范围
- 仅在后续仍确认有收益时，再继续深拆 payment / token 底层协议实现

## Suggested Refactor Order

剩余治理建议按这条顺序推进：

1. `Kiro` 深拆（如后续仍有收益）
   - 当前已完成 service 层治理，但 `core.py / switch.py` 仍可作为后续深拆候选
2. `Grok` 深拆（仅在后续仍确认有收益时）
   - 当前插件入口治理已完成，剩余工作主要是 `core.py` 浏览器自动化与外部同步总线的深拆
3. `ChatGPT` 兼容债 / 平行调用链治理（仅在后续仍确认值得时）
   - 当前插件入口治理已完成，剩余工作主要是 legacy `payment.py / token_refresh.py` 的 import-safe 修复，以及 `api/chatgpt.py / services.external_sync.py` 的边界统一

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
- `platforms/kiro/services/__init__.py` 与 `plugin.py` 中的 helper methods
  - 都改为按需本地导入，减少插件导入阶段与 service 包之间的 import-time coupling

这说明：

- `Cursor / Trae` 的模式复制到 `Kiro` 整体是干净的
- 但 `Kiro` 的桌面切换链路更复杂，因此按设计保留了独立 token service 来吸收 desktop bootstrap / refresh 前置逻辑
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

## Reference Trial: Grok

`Grok` 已作为第四个参考实现完成试点，用来验证 `Cursor / Trae / Kiro` 的“薄插件 + services”模式能否继续复制到浏览器自动化更重、邮箱重试语义更敏感的平台。

本次试点的实际结果是：

- `platforms/grok/plugin.py`
  - 收缩为薄插件入口
  - `register / check_valid / upload_grok2api` 全部委托给 services
- 新增 service 边界：
  - `GrokRegistrationService`
  - `GrokCookieService`
  - `GrokSyncService`
- 注册行为保持兼容：
  - 保留邮箱域名被拒绝时的 mailbox retry 规则
  - 保留“固定 email 不轮换 mailbox”的语义
  - 保留“无 mailbox 时 `otp_callback=None`”的手工 OTP fallback 语义
- 插件 action 之外的外部同步路径
  - `services.external_sync.py`
  - `api.integrations.py`
  - `services.grok2api_runtime.py`
  - 本轮按计划保持不变

这说明：

- `Cursor / Trae / Kiro` 的模式复制到 `Grok` 整体是干净的
- 即使平台内部 `core.py` 仍然高度浏览器特化，也可以先稳定拆出外围 orchestration
- 第四个试点继续证明：先收缩插件入口和 service 边界，比直接重写自动化页面流更稳妥

## Grok Pilot Observations

`Grok` 的复制整体顺利，但实现中也记录了三点轻微偏差：

- 相比 `Kiro`，`Grok` 的 service 包更简单；这轮不需要额外的 token / bootstrap service
- `platforms/grok/plugin.py` 当前直接导入 `platforms.grok.services`，比 `Kiro` 的按需本地导入略更 eager，但目前可接受
- `GrokSyncService` 只承接插件 action 路径；`services.external_sync.py` / `api.integrations.py` / `services.grok2api_runtime.py` 刻意留在试点范围之外

结论：

- `Cursor / Trae / Kiro` 模式复制到 `Grok` 足够干净，可以把它视为第四个参考实现
- 剩余问题属于轻微实现偏差和明确延期范围，而不是模式本身失效
- 当前没有发现阻止后续复制到其他平台的结构性问题

## Reference Trial: ChatGPT

`ChatGPT` 已作为第五个参考实现完成试点，用来验证 `Cursor / Trae / Kiro / Grok` 的“薄插件 + services”模式能否继续复制到 capability routing 更宽、但桌面 / 浏览器流并不是主要复杂度来源的平台。

本次试点的实际结果是：

- `platforms/chatgpt/services/` 已包含：
  - `registration.py`
  - `token.py`
  - `billing.py`
  - `external_sync.py`
- `platforms/chatgpt/plugin.py`
  - 已收缩为薄插件入口
  - `register / check_valid / refresh_token / payment_link / upload_cpa / upload_tm` 全部委托给 services
- `plugin.py` 中的 helper factories
  - 使用本地直接子模块导入
  - 避免在插件导入阶段提前加载 `platforms.chatgpt.services` 与 `register_v2.py` 注册路径
- `services/token.py` 与 `services/billing.py`
  - 显式保留对 legacy `token_refresh.py / payment.py` 的 compatibility loader shims
  - 以兼容这些旧模块当前并非直接 import-safe 的现实
- 插件 action 路径之外的平行调用链
  - `api/chatgpt.py`
  - `services.external_sync.py`
  - 本轮按计划保持不变

这说明：

- `Cursor / Trae / Kiro / Grok` 的模式复制到 `ChatGPT` 整体仍是干净的
- `ChatGPT` 的主要复杂度在 capability routing breadth，而不是 `Kiro` 式桌面链或 `Grok` 式浏览器页面流深度
- 第五个试点继续证明：即使平台同时承载 registration / token / billing / external sync 多类能力，也可以先收缩插件入口，而不必先重写底层协议文件

## ChatGPT Pilot Observations

`ChatGPT` 的复制整体顺利，但实现中也记录了三点值得后续参考的偏差：

- legacy `payment.py / token_refresh.py` 目前不能直接作为稳定 service 依赖导入
  - `token / billing` services 因此保留了 compatibility loader shims
  - 这是兼容旧模块导入缺陷，而不是新的 service 边界设计失效
- 为避免导入阶段把注册路径整条提前拉起，`plugin.py` 没有像 `Grok` 那样直接导入 `platforms.chatgpt.services`
  - 而是继续使用本地 helper + 直接子模块导入
  - 这让 ChatGPT 的 import-time coupling 更接近 `Kiro`
- `ChatGPTExternalSyncService` 只覆盖 `ChatGPTPlatform.execute_action()` 路径
  - `api/chatgpt.py` / `services.external_sync.py` 这类平行直连路径被刻意留在本轮范围之外

结论：

- `Cursor / Trae / Kiro / Grok` 模式复制到 `ChatGPT` 足够干净，可以把它视为第五个参考实现
- 剩余问题主要属于 legacy 兼容债与明确延期范围，而不是模式本身失效
- 当前没有发现阻止继续沿用该治理模式的结构性问题

