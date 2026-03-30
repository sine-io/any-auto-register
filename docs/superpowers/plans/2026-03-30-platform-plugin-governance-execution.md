# Platform Plugin Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `platforms/*` 从“能跑但风格不一”推进到“契约稳定、错误返回一致、可逐步拆分”的状态。

**Architecture:** 这一轮不做大重构，先建立平台契约测试与文档，再对高频平台动作返回值和边界行为做最小统一。以 `BasePlatform` 为中心，把成功/失败返回形状、action 元数据形状和后续拆分候选固定下来。

**Tech Stack:** Python, dataclasses, pytest, existing plugin registry, platform plugins

---

## Scope

本计划只覆盖三类事情：

1. **契约固定**
   - 元数据
   - action 列表形状
   - action 执行返回形状
2. **高频平台统一**
   - Cursor
   - Trae
   - Grok
   - Kiro
   - ChatGPT
3. **耦合点清单**
   - 记录后续重构优先级

不在这一轮做的事：

- 不改注册主流程
- 不改 Go 控制面协议
- 不做大规模 platform/core 拆分类重构

---

### Task 1: 固定平台插件最小契约

**Files:**
- Modify: `core/base_platform.py`
- Create: `tests/platforms/test_platform_contracts.py`
- Create: `docs/platform-plugin-guidelines.md`

- [ ] Step 1: 写失败测试，要求主力平台插件都具备 `name / display_name / version / supported_executors`。
- [ ] Step 2: 写失败测试，要求 `get_platform_actions()` 中每个 action 至少有 `id / label / params`。
- [ ] Step 3: 在 `BasePlatform` 增加统一 action result helper，例如成功和失败的标准返回构造。
- [ ] Step 4: 运行 `pytest tests/platforms/test_platform_contracts.py -q`，确认契约测试转绿。
- [ ] Step 5: 提交一次独立 commit，例如 `refactor: standardize primary platform action contracts`。

### Task 2: 统一高频平台 action 返回形状

**Files:**
- Modify: `platforms/cursor/plugin.py`
- Modify: `platforms/trae/plugin.py`
- Modify: `platforms/grok/plugin.py`
- Modify: `platforms/kiro/plugin.py`
- Modify: `platforms/chatgpt/plugin.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] Step 1: 写失败测试，覆盖 `Cursor / Trae` 缺 token 时必须返回 `{"ok": false, "error": ...}`。
- [ ] Step 2: 写失败测试，覆盖 `Grok / Kiro / ChatGPT` 的上传类 action 失败时必须有 `error` 字段。
- [ ] Step 3: 先修 `Cursor / Trae / Grok`，用 `BasePlatform` helper 统一 success/error 形状。
- [ ] Step 4: 再修 `Kiro / ChatGPT` 的动作返回，保证 success/error 形状一致。
- [ ] Step 5: 运行：
  - `pytest tests/platforms/test_platform_contracts.py -q`
  - `pytest tests/test_risk_hardening.py -q`
- [ ] Step 6: 提交一次独立 commit，例如 `refactor: align primary platform action results`。

### Task 3: 记录高频平台耦合热点

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`

- [ ] Step 1: 逐个审 `Cursor / Trae / Grok / Kiro / ChatGPT` 的 `plugin.py` 和核心依赖文件。
- [ ] Step 2: 在文档中明确记录每个平台当前的混杂点：
  - 注册编排
  - 外部 API
  - 桌面客户端控制
  - token 刷新
  - 外部系统同步
- [ ] Step 3: 为每个平台列出“后续拆分候选”，但不在本轮实现。
- [ ] Step 4: 给出推荐重构顺序。
- [ ] Step 5: 提交一次独立 commit，例如 `docs: inventory platform plugin coupling hotspots`。

### Task 4: 补平台治理评估文档

**Files:**
- Create: `docs/postgres-migration-evaluation.md`
- Create: `docs/worker-scaling-evaluation.md`

- [ ] Step 1: 记录当前 SQLite 为什么还能用，以及真实瓶颈在哪。
- [ ] Step 2: 评估 PostgreSQL 迁移收益、成本和触发条件。
- [ ] Step 3: 评估多 Worker / 远程 Worker / 队列化的演进路径。
- [ ] Step 4: 明确“现在不建议立刻迁”的原因。
- [ ] Step 5: 提交一次独立 commit，例如 `docs: evaluate database and worker scaling paths`。

### Task 5: 完成阶段性验证

**Files:**
- No code changes required

- [ ] Step 1: 运行 `pytest tests/platforms/test_platform_contracts.py -q`
- [ ] Step 2: 运行 `pytest tests/test_risk_hardening.py -q`
- [ ] Step 3: 运行 `go test ./...`
- [ ] Step 4: 运行 `npm run build`
- [ ] Step 5: 汇总“已统一的平台”和“尚未统一的平台”清单

---

## Priority Order Inside P2

1. 契约测试落地
2. Cursor / Trae / Grok 统一
3. Kiro / ChatGPT 统一
4. 耦合热点清单
5. 数据 / Worker 评估文档

## Exit Criteria

完成后，应满足：

- 主力平台 action 返回形状一致
- 平台插件存在最小契约测试
- 文档中明确列出后续重构候选
- 不引入新的控制面或执行面行为回归
