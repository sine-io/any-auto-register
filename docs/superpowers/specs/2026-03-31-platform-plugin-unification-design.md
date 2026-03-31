# Platform Plugin Unification Design

**Goal**

把当前已经完成试点的 5 个平台（`Cursor / Trae / Kiro / Grok / ChatGPT`）从“分别可用、风格接近但不完全一致”的状态，统一到一套更稳定的插件 / service / helper / import 约定上，为后续维护和继续扩平台降低心智负担。

具体目标：

1. 统一 `plugin.py` 的薄入口模式
2. 统一 service factory 命名和职责边界命名
3. 统一 `services/__init__.py` 与 plugin helper 的 import 策略
4. 明确哪些平台差异是允许的，哪些风格差异应当消除
5. 不改变现有平台对外行为，不引入新的产品功能

## Scope

本次统一收尾仅覆盖：

- `platforms/cursor/*`
- `platforms/trae/*`
- `platforms/kiro/*`
- `platforms/grok/*`
- `platforms/chatgpt/*`
- 对应平台测试文件：
  - `tests/platforms/test_platform_contracts.py`
  - 各平台的 `test_*_services.py`
- 辅助文档：
  - `docs/platform-plugin-guidelines.md`

本轮**不覆盖**：

- 深拆 `core.py / switch.py`
- `api/chatgpt.py`
- `services/external_sync.py`
- `api/integrations.py`
- `services/grok2api_runtime.py`
- 多用户 / RBAC / PostgreSQL / 多 Worker

## Current Problems

### 1. plugin.py 已经普遍变薄，但风格不一致

当前 5 个平台都已经进入“薄插件”方向，但仍存在差异：

- `Cursor / Trae`
  - 顶层直接导入 service 包里的类
- `Kiro / ChatGPT`
  - helper 内局部导入具体 service 子模块
- `Grok`
  - 仍直接导入 `platforms.grok.services`

这些差异不是全部都有必要，后续维护时会带来认知噪音。

### 2. service factory 命名已接近统一，但还没正式固化为约定

当前已有模式包括：

- `_registration_service()`
- `_account_service()`
- `_cookie_service()`
- `_token_service()`
- `_desktop_service()`
- `_billing_service()`
- `_sync_service()`
- `_external_sync_service()`
- `_manager_sync_service()`

这些模式已经自然长出来了，但还没有被正式定义成“统一约定”。

### 3. services/__init__.py 风格不一致

当前大致分两类：

#### 直接导出型
- `platforms/cursor/services/__init__.py`
- `platforms/trae/services/__init__.py`
- `platforms/grok/services/__init__.py`

#### lazy export 型
- `platforms/kiro/services/__init__.py`
- `platforms/chatgpt/services/__init__.py`

这背后其实是“是否存在重依赖 / import-time coupling”的差异，但目前没有统一规则。

### 4. 并不是所有平台都需要相同数量的 service

当前平台差异是合理存在的：

- `Cursor`
  - registration / account / desktop
- `Trae`
  - registration / account / desktop / billing
- `Kiro`
  - registration / token / desktop / manager_sync
- `Grok`
  - registration / cookie / sync
- `ChatGPT`
  - registration / token / billing / external_sync

问题不在“数量不同”，而在：
- 哪些差异是合理的能力差异
- 哪些只是历史遗留导致的风格差异

## Design Principles

### Principle 1: 统一约定优先于继续发明新风格

如果已有模式可以表达当前平台能力，优先复用已有命名和结构，而不是为某个平台再发明新的 helper / service 命名。

### Principle 2: 合理差异允许存在

统一的目标不是让 5 个平台“文件数量完全一样”，而是让它们：

- 一眼能看懂
- service 边界命名有规律
- import 方式可解释
- 外部行为一致

### Principle 3: import 策略按依赖重量决定，而不是按个人偏好决定

#### 轻依赖 service 包
如果 service 包不会拉起重依赖：
- 可以允许 `plugin.py` 直接导入 service 类

#### 重依赖 service 包
如果 service 包会牵出：
- Playwright
- 浏览器自动化
- 桌面副作用
- legacy import-safe shim

则：
- `plugin.py` 应使用 helper 内的局部导入
- `services/__init__.py` 应使用 lazy export

## Target Conventions

### 1. plugin.py 统一结构

每个平台插件都尽量收敛为：

1. metadata
2. `__init__`
3. service helper factories
4. `register`
5. `check_valid`
6. `get_platform_actions`
7. `execute_action`

目标不是强制完全相同顺序，而是让读者可以稳定预期结构。

### 2. service helper 命名统一

约定以下命名为推荐标准：

- `_registration_service()`
- `_account_service()`
- `_cookie_service()`
- `_token_service()`
- `_desktop_service()`
- `_billing_service()`
- `_sync_service()`
- `_external_sync_service()`
- `_manager_sync_service()`

其中：
- `account` 与 `cookie` 属于能力差异，不要求二选一统一
- `sync` / `external_sync` / `manager_sync` 也允许因语义更精确而保留差异

### 3. services/__init__.py 统一规则

#### 规则 A：默认用简单导出
对于轻依赖平台，允许：

```python
from .registration import FooRegistrationService
...
```

#### 规则 B：一旦会拉起重依赖，就用 lazy export
对于像 `Kiro / ChatGPT` 这类存在明显 import-time coupling 风险的平台：
- 保留 `__getattr__` + `importlib` 的 lazy export 模式

#### 规则 C：plugin helper 可继续局部导入
即使 `services/__init__.py` 是 lazy export，`plugin.py` 仍优先使用：
- helper 内的局部直接子模块导入

因为这更明确，也更容易控制导入边界。

### 4. 可接受的平台差异

以下差异视为**合理差异**，不强制统一：

- `Trae` 保留独立 `billing service`
- `Kiro` 保留独立 `token service`
- `Grok` 保留 `cookie service`
- `ChatGPT` 保留 `external_sync service`

因为这些是能力结构差异，不是风格漂移。

## Platform-by-Platform Target State

### Cursor
目标：
- 保持现有 3-service 结构
- 只做风格统一，不扩大范围

### Trae
目标：
- 保持 `billing service`
- 与 Cursor/Kiro/ChatGPT 在 helper 风格上更一致

### Kiro
目标：
- 保持 lazy export + 局部导入
- 作为“重依赖平台”的 import 样板

### Grok
目标：
- 从当前的直接导入 service 包，调整到更接近 Kiro 的导入纪律
- 至少让 plugin helper 不再依赖整包 eager import

### ChatGPT
目标：
- 保持 lazy export + 局部导入
- 保持 compatibility loader shim 边界清晰

## Testing Strategy

本轮不追求补大量新功能测试，但要保证：

### Contract-level
- `tests/platforms/test_platform_contracts.py` 继续通过

### Platform-level
对 5 个平台至少保证：
- service 测试仍通过
- plugin delegation 测试仍通过

### New Unification Checks
建议补一组轻量统一性测试，验证：
- helper factory 是否存在
- import 策略是否符合各自平台约定
- plugin 入口不再回退到旧内联逻辑

## Migration Plan

### Step 1

定义统一约定：
- helper naming
- import strategy rule
- acceptable differences

### Step 2

收敛 5 个平台的 `plugin.py` helper 风格。

### Step 3

收敛 `services/__init__.py` 风格：
- 明确保留的 lazy export
- 明确保留的简单导出

### Step 4

补轻量统一性测试与文档更新。

## Success Criteria

完成后，应满足：

- 5 个参考实现的 `plugin.py` 结构一眼可读
- helper 命名有规律
- import 策略可以解释清楚
- 合理差异与历史噪音被明确区分
- 不改变任何平台对外行为

## Non-Goals

这轮不做：

- 深拆任何 `core.py / switch.py`
- 收口旁路调用链
- auth / RBAC / secrets / PostgreSQL / Worker 扩展
