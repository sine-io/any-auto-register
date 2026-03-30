# Cursor Platform Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `Cursor` 平台插件从“厚插件入口”重构为“薄插件 + 明确 service 边界”，同时保持当前对外行为不变。

**Architecture:** 保留 `platforms/cursor/core.py` 作为协议实现层，不动注册协议细节。新增 `registration / desktop / account` 三个 service，`plugin.py` 只负责平台入口、service 调度和统一返回包装。

**Tech Stack:** Python, dataclasses, pytest, curl_cffi, existing platform plugin architecture

---

## File Structure

### New Files

- `platforms/cursor/services/__init__.py`
  - service 包入口
- `platforms/cursor/services/registration.py`
  - 负责邮箱/OTP/注册编排
- `platforms/cursor/services/desktop.py`
  - 负责桌面切换与 IDE 重启包装
- `platforms/cursor/services/account.py`
  - 负责 `check_valid` / `get_user_info`
- `tests/platforms/test_cursor_services.py`
  - Cursor 专项 service 测试

### Modified Files

- `platforms/cursor/plugin.py`
  - 收缩为薄插件入口
- `tests/platforms/test_platform_contracts.py`
  - 保留并在需要时补 Cursor 相关断言

### Files Explicitly Not Refactored This Round

- `platforms/cursor/core.py`
- `platforms/cursor/switch.py`

它们继续保留为底层依赖层，避免把试点范围扩得过大。

---

### Task 1: 建立 Cursor service 测试骨架

**Files:**
- Create: `tests/platforms/test_cursor_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Write the failing registration service test**

```python
def test_cursor_registration_service_builds_otp_callback():
    ...
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /root/any-auto-register/.worktrees/cursor-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_cursor_services.py::test_cursor_registration_service_builds_otp_callback -q
```

Expected:
- FAIL because `CursorRegistrationService` does not exist yet

- [ ] **Step 3: Write the failing account service tests**

```python
def test_cursor_account_service_check_valid_uses_token():
    ...

def test_cursor_account_service_get_user_info_wraps_failure():
    ...
```

- [ ] **Step 4: Run tests to verify they fail**

Run:

```bash
pytest tests/platforms/test_cursor_services.py -q
```

Expected:
- FAIL with missing service classes/functions

- [ ] **Step 5: Commit**

```bash
git add tests/platforms/test_cursor_services.py
git commit -m "test: add cursor service contract coverage"
```

### Task 2: 提取 Registration / Account / Desktop services

**Files:**
- Create: `platforms/cursor/services/__init__.py`
- Create: `platforms/cursor/services/registration.py`
- Create: `platforms/cursor/services/account.py`
- Create: `platforms/cursor/services/desktop.py`
- Test: `tests/platforms/test_cursor_services.py`

- [ ] **Step 1: Implement minimal `CursorRegistrationService`**

Requirements:
- 接受 `RegisterConfig`、`mailbox`、`log_fn`
- 组装 `otp_callback`
- 调用现有 `CursorRegister.register`
- 返回 `Account`

- [ ] **Step 2: Implement minimal `CursorAccountService`**

Requirements:
- 封装 `check_valid`
- 封装 `get_user_info`
- 不改现有远程接口行为

- [ ] **Step 3: Implement minimal `CursorDesktopService`**

Requirements:
- 封装 `switch_account`
- 封装 `restart_ide`
- 返回统一的 action 结果构造输入

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
pytest tests/platforms/test_cursor_services.py -q
```

Expected:
- PASS

- [ ] **Step 5: Commit**

```bash
git add platforms/cursor/services tests/platforms/test_cursor_services.py
git commit -m "refactor: extract cursor services"
```

### Task 3: 收缩 Cursor 插件入口

**Files:**
- Modify: `platforms/cursor/plugin.py`
- Test: `tests/platforms/test_cursor_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Rewrite `CursorPlatform.register` to delegate to registration service**

Requirements:
- 插件入口不再直接管理 OTP callback 细节
- 对外行为保持不变

- [ ] **Step 2: Rewrite `CursorPlatform.check_valid` to delegate to account service**

- [ ] **Step 3: Rewrite `CursorPlatform.execute_action` to delegate to account/desktop services**

Requirements:
- `switch_account` 由 desktop service 处理
- `get_user_info` 由 account service 处理
- action id 不变
- 返回形状不变

- [ ] **Step 4: Run focused platform tests**

Run:

```bash
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_cursor_services.py -q
```

Expected:
- PASS

- [ ] **Step 5: Run broader regression**

Run:

```bash
pytest tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 6: Commit**

```bash
git add platforms/cursor/plugin.py tests/platforms
git commit -m "refactor: slim cursor platform plugin"
```

### Task 4: 文档与试点验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-03-30-cursor-refactor-design.md`

- [ ] **Step 1: Update plugin guidelines with Cursor as the reference implementation**

- [ ] **Step 2: Note any deviations discovered during implementation**

- [ ] **Step 3: Record whether this split is suitable for Trae**

- [ ] **Step 4: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/cursor-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_cursor_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 5: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-03-30-cursor-refactor-design.md
git commit -m "docs: record cursor refactor outcomes"
```

---

## Success Criteria

- `CursorPlatform` 对外接口不变
- `plugin.py` 明显变薄
- `registration / account / desktop` 边界清晰
- 专项测试存在并通过
- 可以把该模式复制到 `Trae`
