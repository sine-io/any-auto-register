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

这一轮优先治理：

- `Cursor`
- `Trae`
- `Grok`

原因：

- 它们在桌面切换、升级链接、外部上传这类动作上最常被直接调用
- 也是最容易出现返回形状不一致的平台

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

建议后续按这条顺序推进：

1. `Cursor / Trae`
   - 因为桌面切换类 action 最直接影响用户操作体验
2. `Grok`
   - 因为浏览器自动化复杂度最高，且平台特定细节多
3. `Kiro`
   - 因为 token / desktop / manager sync 三段逻辑耦合最深
4. `ChatGPT`
   - 功能多，但边界已经比前几者略清楚

## Refactor Success Criteria

如果后续要继续拆分，一个平台至少应满足：

- 插件入口只做 capability 暴露和少量组装
- 注册编排、外部同步、桌面切换各自有独立服务边界
- action 结果统一遵守平台契约
- 至少有一层可脱离真实站点运行的最小单元测试
