# Platform Plugin Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 统一 5 个参考平台（Cursor / Trae / Kiro / Grok / ChatGPT）的 plugin / service / helper / import 风格，同时不改变任何平台对外行为。

**Architecture:** 本轮不再做新平台试点，也不深拆任何 `core.py / switch.py`。工作重点是统一 `plugin.py` 结构、helper factory 命名、`services/__init__.py` 的导入策略，以及补一层轻量统一性测试，让已经完成的五个参考实现从“分别成立”收敛为“整体一致”。

**Tech Stack:** Python, pytest, existing platform plugin architecture

---

## File Structure

### Modified Files

- `platforms/cursor/plugin.py`
  - 统一 helper / import 风格
- `platforms/trae/plugin.py`
  - 统一 helper / import 风格
- `platforms/kiro/plugin.py`
  - 统一 helper / import 风格（尽量保持现有 lazy pattern）
- `platforms/grok/plugin.py`
  - 收敛到明确的 helper / import 纪律
- `platforms/chatgpt/plugin.py`
  - 统一 helper / import 风格
- `platforms/cursor/services/__init__.py`
- `platforms/trae/services/__init__.py`
- `platforms/kiro/services/__init__.py`
- `platforms/grok/services/__init__.py`
- `platforms/chatgpt/services/__init__.py`
  - 按统一规则保留简单导出或 lazy export
- `tests/platforms/test_platform_contracts.py`
  - 保持现有契约测试
- `tests/platforms/test_platform_unification.py`
  - 新增轻量统一性测试
- `docs/platform-plugin-guidelines.md`
  - 更新统一约定与最终规则
- `docs/superpowers/specs/2026-03-31-platform-plugin-unification-design.md`
  - 回填实施结果

### Files Explicitly Not Refactored This Round

- 所有 `platforms/*/core.py`
- 所有 `platforms/*/switch.py`
- `api/chatgpt.py`
- `services/external_sync.py`
- `api/integrations.py`
- `services/grok2api_runtime.py`

这些都属于下一阶段工作，不在本轮统一收尾里处理。

---

### Task 1: 收敛统一约定到测试层

**Files:**
- Create: `tests/platforms/test_platform_unification.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Write the failing unification tests**

至少覆盖这些约定：

```python
def test_platform_plugins_expose_service_factory_helpers():
    ...

def test_heavy_platform_plugins_use_local_service_imports():
    ...

def test_heavy_service_packages_use_lazy_exports():
    ...
```

说明：
- “heavy platforms” 至少包括 `Kiro`、`ChatGPT`
- 对 `Grok` 本轮目标要明确下来：统一到 `Kiro` 风格，即避免整包 eager import

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd /root/any-auto-register/.worktrees/platform-unification-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_platform_unification.py -q
```

Expected:
- FAIL because the current plugin/service files are not fully unified yet

- [ ] **Step 3: Commit**

```bash
git add tests/platforms/test_platform_unification.py
git commit -m "test: add platform unification coverage"
```

### Task 2: 统一 plugin helper factory 风格

**Files:**
- Modify: `platforms/cursor/plugin.py`
- Modify: `platforms/trae/plugin.py`
- Modify: `platforms/kiro/plugin.py`
- Modify: `platforms/grok/plugin.py`
- Modify: `platforms/chatgpt/plugin.py`
- Test: `tests/platforms/test_platform_unification.py`

- [ ] **Step 1: Align helper naming and ordering**

Requirements:
- helper factory 命名保持现有推荐集合内
- `plugin.py` 尽量按统一顺序组织：
  1. metadata
  2. `__init__`
  3. helper factories
  4. `register`
  5. `check_valid`
  6. `get_platform_actions`
  7. `execute_action`

- [ ] **Step 2: Make Grok follow the heavy-platform import discipline**

Requirements:
- 不再在 `plugin.py` 顶层直接依赖整包 eager import
- helper 内改为局部导入，或与最终 `services/__init__.py` 规则相匹配

- [ ] **Step 3: Run focused tests**

Run:

```bash
pytest tests/platforms/test_platform_unification.py -q
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_cursor_services.py -q
pytest tests/platforms/test_trae_services.py -q
pytest tests/platforms/test_kiro_services.py -q
pytest tests/platforms/test_grok_services.py -q
pytest tests/platforms/test_chatgpt_services.py -q
```

Expected:
- PASS

- [ ] **Step 4: Commit**

```bash
git add platforms/cursor/plugin.py platforms/trae/plugin.py platforms/kiro/plugin.py platforms/grok/plugin.py platforms/chatgpt/plugin.py tests/platforms/test_platform_unification.py
 git commit -m "refactor: align platform plugin helper conventions"
```

### Task 3: 统一 services/__init__.py 导入规则

**Files:**
- Modify: `platforms/cursor/services/__init__.py`
- Modify: `platforms/trae/services/__init__.py`
- Modify: `platforms/kiro/services/__init__.py`
- Modify: `platforms/grok/services/__init__.py`
- Modify: `platforms/chatgpt/services/__init__.py`
- Test: `tests/platforms/test_platform_unification.py`

- [ ] **Step 1: Lock the explicit rule**

Requirements:
- `Cursor / Trae` 继续用简单导出，除非引入重依赖证据
- `Kiro / ChatGPT` 保留 lazy export
- `Grok` 明确二选一：
  - 若仍存在 import-time coupling 风险，则切到 lazy export
  - 否则保留简单导出并在 tests 中明确为轻依赖例外

推荐：`Grok` 与 `Kiro / ChatGPT` 对齐，切到 lazy export

- [ ] **Step 2: Run focused tests**

Run:

```bash
pytest tests/platforms/test_platform_unification.py -q
pytest tests/platforms/test_cursor_services.py -q
pytest tests/platforms/test_trae_services.py -q
pytest tests/platforms/test_kiro_services.py -q
pytest tests/platforms/test_grok_services.py -q
pytest tests/platforms/test_chatgpt_services.py -q
```

Expected:
- PASS

- [ ] **Step 3: Commit**

```bash
git add platforms/cursor/services/__init__.py platforms/trae/services/__init__.py platforms/kiro/services/__init__.py platforms/grok/services/__init__.py platforms/chatgpt/services/__init__.py tests/platforms/test_platform_unification.py
 git commit -m "refactor: unify platform service import strategy"
```

### Task 4: 文档与最终验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-03-31-platform-plugin-unification-design.md`

- [ ] **Step 1: Update guidelines with final unification rules**

至少写清：
- helper naming rule
- plugin ordering rule
- import strategy rule
- acceptable differences

- [ ] **Step 2: Record final Grok import decision explicitly**

把 reviewer 提到的模糊点写清楚：
- `Grok` 最终是简单导出还是 lazy export
- 为什么

- [ ] **Step 3: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/platform-unification-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_platform_unification.py tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_cursor_services.py -q
pytest tests/platforms/test_trae_services.py -q
pytest tests/platforms/test_kiro_services.py -q
pytest tests/platforms/test_grok_services.py -q
pytest tests/platforms/test_chatgpt_services.py -q
python - <<'PY'
from core.registry import load_all, list_platforms
load_all()
names = {item["name"] for item in list_platforms()}
required = {"cursor", "trae", "kiro", "grok", "chatgpt"}
missing = sorted(required - names)
if missing:
    raise SystemExit(f"missing registrations: {missing}")
print("registry smoke ok")
PY
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 4: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-03-31-platform-plugin-unification-design.md tests/platforms/test_platform_unification.py
 git commit -m "docs: record platform unification outcomes"
```

---

## Success Criteria

- 5 个参考实现的 `plugin.py` 风格统一到可预期模式
- helper naming 规则清晰
- `services/__init__.py` 的 import 策略被明确并被测试覆盖
- `Grok` 的最终 import 策略不再模糊
- 不改变任何平台对外行为
