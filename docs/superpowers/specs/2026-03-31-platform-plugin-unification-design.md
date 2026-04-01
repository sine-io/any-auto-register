# Platform Plugin Unification Design

**Status**

Accepted and implemented on 2026-03-31. This document records the final outcome of the five-platform unification work, not a pending proposal.

**Goal**

把已经完成试点的 5 个平台（`Cursor / Trae / Kiro / Grok / ChatGPT`）统一到一套稳定的插件 / service / helper / import 约定上，降低后续维护与扩平台的心智负担，同时不改变任何平台对外行为。

## Scope

本次统一收尾覆盖：

- `platforms/cursor/*`
- `platforms/trae/*`
- `platforms/kiro/*`
- `platforms/grok/*`
- `platforms/chatgpt/*`
- `tests/platforms/test_platform_unification.py`
- `tests/platforms/test_platform_contracts.py`
- 各平台的 `test_*_services.py`
- `docs/platform-plugin-guidelines.md`

本轮明确不覆盖：

- 深拆任何 `core.py / switch.py`
- `api/chatgpt.py`
- `services/external_sync.py`
- `api/integrations.py`
- `services/grok2api_runtime.py`
- 多用户 / RBAC / PostgreSQL / 多 Worker

## Final Outcome

本轮最终收敛到下面这组结论：

- `Cursor / Trae` 保持 simple-export 平台
- `Kiro / ChatGPT` 保持 lazy-export + helper 内局部导入的平台
- `Grok` 已调整为与 `Kiro / ChatGPT` 相同的 import discipline
- helper factory 命名与 plugin method ordering 已由测试固化
- 可接受差异现在按 capability type 解释，而不是按历史迁移路径解释
- 所有变化都以“不改变现有平台对外行为”为前提

## Final Rules

### 1. Helper Naming Rule

插件内的 service factory helper 统一使用 `_xxx_service()` 命名。

已固化的 helper 命名集合：

- `_registration_service()`
- `_account_service()`
- `_cookie_service()`
- `_token_service()`
- `_desktop_service()`
- `_billing_service()`
- `_sync_service()`
- `_external_sync_service()`
- `_manager_sync_service()`

决策原则：

- `xxx` 必须代表一个真实 capability boundary
- 同义职责不再发明新 helper 名称
- 平台可以拥有不同 helper 集合，但每个 helper 都必须能映射到明确 service 职责
- helper naming 规则已经由 `tests/platforms/test_platform_unification.py` 锁定

### 2. Plugin Ordering Rule

平台类中的稳定结构顺序为：

1. metadata class attributes
2. `__init__`
3. helper factories
4. `register`
5. `check_valid`
6. `get_platform_actions`
7. `execute_action`

这条规则的重点不是格式洁癖，而是：

- 让 5 个参考实现一眼可读
- 让 helper 能稳定地位于 capability routing 之前
- 让 AST 级统一性测试可以直接表达约束
- helper ordering 与 method ordering 都属于最终测试契约的一部分

### 3. Import Strategy Rule

导入策略按依赖重量决定，而不是按平台历史决定。

#### Simple-export platforms

`Cursor / Trae` 保持 simple-export：

- `services/__init__.py` 直接 re-export service 类
- 顶层相对导入是可接受的
- 不要求引入 lazy export

#### Lazy-export platforms

`Kiro / Grok / ChatGPT` 保持 lazy-export：

- `services/__init__.py` 定义 `__all__`
- `services/__init__.py` 通过 `__getattr__` 做按需导出
- `services/__init__.py` 不保留 eager top-level relative imports
- `plugin.py` helper factory 在函数体内对具体 service 子模块做局部导入
- `plugin.py` 不在模块顶层直接导入 `platforms.<name>.services`
- 这组规则一起构成 heavy 平台的统一 import discipline

这样区分的原因是这些平台存在更明显的 import-time coupling 风险，例如浏览器自动化、桌面链路、cookie/token 引导、legacy compatibility shim。

### 4. Acceptable Differences

统一后允许保留的差异由 capability type 决定：

- `Cursor`: `registration / account / desktop`
- `Trae`: `registration / account / desktop / billing`
- `Kiro`: `registration / token / desktop / manager_sync`
- `Grok`: `registration / cookie / sync`
- `ChatGPT`: `registration / token / billing / external_sync`

因此，以下差异被接受：

- service 数量不同
- `account` 与 `cookie` 的差异
- `sync / external_sync / manager_sync` 的差异
- `simple-export` 与 `lazy-export` 的差异，只要能用依赖重量解释

以下差异不再接受：

- 仅由历史原因造成的 helper 命名漂移
- 仅由历史原因造成的 eager vs lazy import 分叉
- 把编排逻辑重新塞回 `plugin.py`
- 让插件结构偏离统一顺序

## Final Grok Import Decision

`Grok` 的最终导入决策已经明确并落地：

- `platforms/grok/services/__init__.py` 现在使用 lazy export
- 实现形式是 `__all__` + `_SERVICE_MODULES` + `__getattr__`
- `platforms/grok/plugin.py` 中的 helper factory 使用局部子模块导入
- `GrokPlatform._registration_service()` 从 `platforms.grok.services.registration` 局部导入
- `GrokPlatform._cookie_service()` 从 `platforms.grok.services.cookie` 局部导入
- `GrokPlatform._sync_service()` 从 `platforms.grok.services.sync` 局部导入

这项决定与 `Kiro / ChatGPT` 对齐，原因是：

- 统一标准现在是 import-time coupling 风险，而不是平台先后顺序
- `Grok` 与 `Kiro / ChatGPT` 一样，都更适合避免插件导入阶段把整个 service 包 eager 拉起
- 三者的 capability 集合不同，但 import discipline 应保持一致
- 对齐点不仅是 lazy-export，还包括 plugin helper factories 一律做局部导入

## Platform Matrix

### Cursor

- 定位：simple-export 平台
- helper：`registration / account / desktop`
- 说明：保持轻量直接导出，不额外引入 lazy export 复杂度

### Trae

- 定位：simple-export 平台
- helper：`registration / account / desktop / billing`
- 说明：`billing` 是 capability difference，不是风格例外

### Kiro

- 定位：lazy-export 平台
- helper：`registration / token / desktop / manager_sync`
- 说明：继续作为重依赖平台的 import discipline 参考实现

### Grok

- 定位：lazy-export 平台
- helper：`registration / cookie / sync`
- 说明：现在在导入纪律上与 `Kiro / ChatGPT` 对齐

### ChatGPT

- 定位：lazy-export 平台
- helper：`registration / token / billing / external_sync`
- 说明：保留 lazy export + 本地 helper 导入，用于隔离 legacy import-safe 风险

## Testing Strategy

最终设计由下面几层验证共同覆盖：

- `tests/platforms/test_platform_unification.py`
  - AST 级检查 helper naming
  - AST 级检查 plugin ordering
  - AST 级检查 heavy/light import discipline
  - AST 级检查 `services/__init__.py` 的 lazy-export 或 direct-export 契约
- `tests/platforms/test_platform_contracts.py`
  - 检查平台对外契约未回退
- 各平台 `test_*_services.py`
  - 检查 service delegation 与行为兼容性
- registry smoke
  - 检查五个平台都能被正常注册

## Success Criteria

本轮完成后的最终判定标准是：

- 5 个参考实现的 `plugin.py` 结构可预测
- helper naming 有规律且被测试锁定
- import strategy 可解释且不再出现 Grok 特例
- 合理差异按 capability type 解释
- 所有平台对外行为保持不变

## Non-Goals

这轮不做：

- 深拆任何 `core.py / switch.py`
- 收口旁路调用链
- auth / RBAC / secrets / PostgreSQL / Worker 扩展
