# Kiro Platform Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `Kiro` 平台插件从“厚插件入口”重构为“薄插件 + 明确 service 边界”，同时保持当前对外行为不变。

**Architecture:** 保留 `platforms/kiro/core.py` 作为注册协议与桌面 token 抓取层，保留 `platforms/kiro/switch.py` 作为 token 刷新/桌面副作用层，保留 `platforms/kiro/account_manager_upload.py` 作为外部同步底层实现。新增 `registration / token / desktop / manager_sync` 四个 service，`plugin.py` 只负责平台入口、service 调度和统一返回包装。

**Tech Stack:** Python, pytest, Playwright-based existing Kiro automation, curl_cffi, existing platform plugin architecture

---

## File Structure

### New Files

- `platforms/kiro/services/__init__.py`
  - service 包入口
- `platforms/kiro/services/registration.py`
  - 负责邮箱/OTP/Builder ID 注册编排
- `platforms/kiro/services/token.py`
  - 负责 `check_valid` / `refresh_token` / desktop token bootstrap
- `platforms/kiro/services/desktop.py`
  - 负责 `switch_account` 长链与 `restart_ide`
- `platforms/kiro/services/manager_sync.py`
  - 负责导入 Kiro Manager
- `tests/platforms/test_kiro_services.py`
  - Kiro 专项 service 测试

### Modified Files

- `platforms/kiro/plugin.py`
  - 收缩为薄插件入口
- `tests/platforms/test_platform_contracts.py`
  - 保留并在需要时补 Kiro 相关断言
- `docs/platform-plugin-guidelines.md`
  - 记录 Kiro 作为第三个参考实现的结果
- `docs/superpowers/specs/2026-03-30-kiro-refactor-design.md`
  - 记录试点 outcome / deviations / pattern assessment

### Files Explicitly Not Refactored This Round

- `platforms/kiro/core.py`
- `platforms/kiro/switch.py`
- `platforms/kiro/account_manager_upload.py`

这些文件继续保留为底层依赖层，避免把第三个试点范围扩得过大。

---

### Task 1: 建立 Kiro service 测试骨架

**Files:**
- Create: `tests/platforms/test_kiro_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Write the failing registration service test**

```python
def test_kiro_registration_service_builds_otp_callback():
    ...
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /root/any-auto-register/.worktrees/kiro-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_kiro_services.py::test_kiro_registration_service_builds_otp_callback -q
```

Expected:
- FAIL because `KiroRegistrationService` does not exist yet

- [ ] **Step 3: Write the failing token / desktop / manager sync tests**

```python
def test_kiro_token_service_check_valid_uses_refresh_credentials():
    ...

def test_kiro_token_service_refresh_token_wraps_success():
    ...

def test_kiro_token_service_ensure_desktop_tokens_wraps_missing_credentials():
    ...

def test_kiro_desktop_service_switch_account_wraps_restart_result():
    ...

def test_kiro_manager_sync_service_upload_wraps_success():
    ...
```

- [ ] **Step 4: Run tests to verify they fail**

Run:

```bash
pytest tests/platforms/test_kiro_services.py -q
```

Expected:
- FAIL with missing service classes/functions

- [ ] **Step 5: Commit**

```bash
git add tests/platforms/test_kiro_services.py
git commit -m "test: add kiro service contract coverage"
```

### Task 2: 提取 Registration / Token / Desktop / Manager Sync services

**Files:**
- Create: `platforms/kiro/services/__init__.py`
- Create: `platforms/kiro/services/registration.py`
- Create: `platforms/kiro/services/token.py`
- Create: `platforms/kiro/services/desktop.py`
- Create: `platforms/kiro/services/manager_sync.py`
- Test: `tests/platforms/test_kiro_services.py`

- [ ] **Step 1: Implement minimal `KiroRegistrationService`**

Requirements:
- 接受 `RegisterConfig`、`mailbox`、`log_fn`
- 组装 OTP callback
- 调用现有 `KiroRegister.register`
- 返回 `Account`

- [ ] **Step 2: Implement minimal `KiroTokenService`**

Requirements:
- 封装 `check_valid`
- 封装 `refresh_token`
- 封装 desktop token bootstrap 前置链
- 不改现有 token 语义与错误语义

- [ ] **Step 3: Implement minimal `KiroDesktopService`**

Requirements:
- 封装 `switch_account`
- 封装 `restart_ide`
- 组合 `token service` 与 `switch.py`
- 保持当前返回 payload 字段

- [ ] **Step 4: Implement minimal `KiroManagerSyncService`**

Requirements:
- 调用 `upload_to_kiro_manager`
- 返回统一的 action result 语义

- [ ] **Step 5: Run tests to verify they pass**

Run:

```bash
pytest tests/platforms/test_kiro_services.py -q
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add platforms/kiro/services tests/platforms/test_kiro_services.py
git commit -m "refactor: extract kiro services"
```

### Task 3: 收缩 Kiro 插件入口

**Files:**
- Modify: `platforms/kiro/plugin.py`
- Test: `tests/platforms/test_kiro_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Rewrite `KiroPlatform.register` to delegate to registration service**

Requirements:
- 插件入口不再直接管理 OTP callback 细节
- 保留当前日志与账号返回行为

- [ ] **Step 2: Rewrite `KiroPlatform.check_valid` to delegate to token service**

- [ ] **Step 3: Rewrite `KiroPlatform.execute_action` to delegate to token / desktop / manager sync services**

Requirements:
- `refresh_token` 由 token service 处理
- `switch_account` 由 desktop service 处理
- `upload_kiro_manager` 由 manager sync service 处理
- action id 不变
- 返回形状不变

- [ ] **Step 4: Run focused platform tests**

Run:

```bash
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_kiro_services.py -q
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
git add platforms/kiro/plugin.py tests/platforms
git commit -m "refactor: slim kiro platform plugin"
```

### Task 4: 文档与试点验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-03-30-kiro-refactor-design.md`

- [ ] **Step 1: Update plugin guidelines with Kiro as the third reference implementation**

- [ ] **Step 2: Note deviations discovered during implementation**

- [ ] **Step 3: Record whether the Cursor / Trae pattern copied cleanly to Kiro**

- [ ] **Step 4: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/kiro-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_kiro_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 5: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-03-30-kiro-refactor-design.md
git commit -m "docs: record kiro refactor outcomes"
```

---

## Success Criteria

- `KiroPlatform` 对外接口不变
- `plugin.py` 明显变薄
- `registration / token / desktop / manager_sync` 边界清晰
- `switch_account` 长链从插件入口下沉
- 专项测试存在并通过
- `Kiro` 成为第三个参考实现
