# Platform Plugin Guidelines

本文档定义平台插件的最小契约，以及 2026-03-31 平台统一收尾后的最终风格约定。当前参考实现与约束测试覆盖：

- `Cursor`
- `Trae`
- `Kiro`
- `Grok`
- `ChatGPT`
- `tests/platforms/test_platform_unification.py`
- `tests/platforms/test_platform_contracts.py`

统一目标不是让所有平台长得完全一样，而是让 `platforms/*/plugin.py` 的行为、helper 命名、方法顺序、导入纪律都可预测、可测试、可由控制面稳定消费。

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

## Final Unification Rules

### Helper Naming Rule

平台插件内的 service helper factory 统一使用 `_xxx_service()` 命名。

- `xxx` 必须描述 service 边界，而不是历史模块名或临时实现细节
- helper 名称必须和对应 service 语义一一对应
- 只有在平台确实多出一个独立 capability boundary 时，才新增新的 helper 名称
- helper 命名预期已经由 `tests/platforms/test_platform_unification.py` 固化，新平台应直接对齐现有命名集合

当前已经固化为参考命名的 helper 包括：

- `_registration_service()`
- `_account_service()`
- `_cookie_service()`
- `_token_service()`
- `_desktop_service()`
- `_billing_service()`
- `_sync_service()`
- `_external_sync_service()`
- `_manager_sync_service()`

这些名称已经由 `tests/platforms/test_platform_unification.py` 约束。新平台或后续重构优先复用这些命名，不要为同义职责发明新变体。

### Plugin Ordering Rule

平台类中的结构顺序按下面的稳定顺序组织：

1. metadata class attributes
2. `__init__`
3. service helper factories
4. `register`
5. `check_valid`
6. `get_platform_actions`
7. `execute_action`

补充约定：

- 一个平台存在多个 helper 时，helper 顺序应保持稳定，并与该平台 capability 的主路由顺序一致
- helper ordering 与 plugin method ordering 已由 `tests/platforms/test_platform_unification.py` 直接校验
- 不要求所有平台拥有相同数量的 helper
- 但不允许把 helper 穿插回 `register()` / `execute_action()` 之后，避免结构漂移

### Import Strategy Rule

导入策略按依赖重量和 import-time coupling 决定，而不是按平台历史或个人偏好决定。

#### Simple-export platforms

`Cursor / Trae` 目前保持 simple-export 平台：

- `services/__init__.py` 使用直接 re-export
- service 包允许顶层相对导入
- `plugin.py` 可以继续保持简单导入路径

这类平台的 service 包较轻，不需要为了统一而强行引入 lazy export 复杂度。

#### Lazy-export platforms

`Kiro / Grok / ChatGPT` 目前统一归为 lazy-export 平台：

- `services/__init__.py` 必须定义 `__all__`
- `services/__init__.py` 必须通过 `__getattr__` 按需加载 service
- `services/__init__.py` 不应保留 eager top-level relative imports
- `plugin.py` 不应在模块顶层直接导入 `platforms.<name>.services`
- 每个 helper factory 在函数体内从具体子模块做局部导入
- 这个组合规则就是 heavy 平台当前的最终 import discipline，不再按平台历史单独分叉

这个规则针对的是 import-time coupling 风险，例如：

- 浏览器自动化链路
- 桌面副作用链路
- cookie / token 引导链路
- legacy compatibility loader shim

### Grok Final Import Decision

`Grok` 的最终决定已经收敛到与 `Kiro / ChatGPT` 相同的导入纪律：

- `platforms/grok/services/__init__.py` 现在使用 lazy export（`__all__` + `__getattr__`）
- `plugin.py` 中的 helper factories 使用局部导入
- `GrokPlatform._registration_service()` 使用局部导入
- `GrokPlatform._cookie_service()` 使用局部导入
- `GrokPlatform._sync_service()` 使用局部导入

原因不是“Grok 历史上和谁更像”，而是：

- `Grok` 也属于 import-time coupling 更敏感的平台
- 因此它应遵守与 `Kiro / ChatGPT` 一样的 heavy-platform import discipline
- 这让 `Grok / Kiro / ChatGPT` 在 service package lazy export 与 helper 内局部导入两层上保持一致
- 允许 service 能力集合不同，但不再允许保留“只有 Grok 例外地 eager import service 包”的历史分叉

### Acceptable Differences

统一后允许保留的差异必须由 capability type 解释，而不是由迁移历史解释。

当前被接受的 capability 差异如下：

- `Cursor`: `registration / account / desktop`
- `Trae`: `registration / account / desktop / billing`
- `Kiro`: `registration / token / desktop / manager_sync`
- `Grok`: `registration / cookie / sync`
- `ChatGPT`: `registration / token / billing / external_sync`

因此，下面这些差异是合理的：

- service 数量不同
- `account` 与 `cookie` 并存
- `sync / external_sync / manager_sync` 并存
- `simple-export` 与 `lazy-export` 并存，只要它们由依赖重量解释
- `Cursor / Trae` 继续保持轻平台的简单直接导出
- `Kiro / Grok / ChatGPT` 继续保持重平台的 lazy-export + helper 局部导入

下面这些差异不再视为合理：

- 同类职责出现无规律 helper 命名
- 仅因历史原因保留不同 import 策略
- 把业务编排重新内联回 `plugin.py`
- helper 顺序或插件结构偏离统一契约

## Reference Classification

当前 5 个参考实现的最终分类如下：

- `Cursor / Trae`
  - 保持 simple-export 平台
  - 重点是薄插件 + 轻量 service re-export
- `Kiro / ChatGPT`
  - 保持 lazy-export + helper 内局部导入
  - 重点是控制 import-time coupling
- `Grok`
  - 现在也归入 lazy-export + helper 内局部导入
  - 导入纪律与 `Kiro / ChatGPT` 对齐

## Token Update Guidance

如果动作执行会刷新 token 或生成新的认证材料：

- 优先放入 `data`
- 字段名尽量使用当前系统已经追踪的名字
- 避免同一平台混用多套命名风格

## Error Handling Guidance

- 缺少关键凭据时，优先返回明确的 `error` 字段，而不是抛异常
- 未知 action 可以继续抛 `NotImplementedError`
- 外部依赖失败时，应尽量转换成可读错误字符串

## Test Enforcement

最终约定不是只写在文档里，也已经写进测试：

- `tests/platforms/test_platform_unification.py`
  - 校验 helper naming
  - 校验 plugin method ordering
  - 校验 heavy/light 平台的 import discipline
  - 校验 `services/__init__.py` 的 lazy-export 或 direct-export 契约
- `tests/platforms/test_platform_contracts.py`
  - 校验插件对外契约未被破坏

文档与测试不一致时，以测试中体现的最终契约为准，并同步更新本文档。
