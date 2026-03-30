# Trae Platform Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `Trae` 平台插件从“厚插件入口”重构为“薄插件 + 明确 service 边界”，同时保持当前对外行为不变。

**Architecture:** 保留 `platforms/trae/core.py` 作为协议实现层，保留 `platforms/trae/switch.py` 作为桌面副作用底层层。新增 `registration / account / desktop / billing` 四个 service，`plugin.py` 只负责平台入口、service 调度和统一返回包装。

**Tech Stack:** Python, pytest, curl_cffi, existing platform plugin architecture

---

## File Structure

### New Files

- `platforms/trae/services/__init__.py`
  - service 包入口
- `platforms/trae/services/registration.py`
  - 负责邮箱/OTP/注册编排
- `platforms/trae/services/account.py`
  - 负责 `check_valid` / `get_user_info`
- `platforms/trae/services/desktop.py`
  - 负责桌面切换与 IDE 重启包装
- `platforms/trae/services/billing.py`
  - 负责 `get_cashier_url` 与 token fallback / create_order
- `tests/platforms/test_trae_services.py`
  - Trae 专项 service 测试

### Modified Files

- `platforms/trae/plugin.py`
  - 收缩为薄插件入口
- `tests/platforms/test_platform_contracts.py`
  - 保留并在需要时补 Trae 相关断言
- `docs/platform-plugin-guidelines.md`
  - 如有必要，补充 Trae 作为 Cursor 模式的复制验证结果
- `docs/superpowers/specs/2026-03-30-trae-refactor-design.md`
  - 记录试点实现后的 outcome notes

### Files Explicitly Not Refactored This Round

- `platforms/trae/core.py`
- `platforms/trae/switch.py`

它们继续保留为底层依赖层，避免把第二个试点范围扩得过大。

---

### Task 1: 建立 Trae service 测试骨架

**Files:**
- Create: `tests/platforms/test_trae_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Write the failing registration service test**

```python
def test_trae_registration_service_builds_otp_callback():
    ...
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /root/any-auto-register/.worktrees/trae-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_trae_services.py::test_trae_registration_service_builds_otp_callback -q
```

Expected:
- FAIL because `TraeRegistrationService` does not exist yet

- [ ] **Step 3: Write the failing account / desktop / billing service tests**

```python
def test_trae_account_service_check_valid_uses_token():
    ...

def test_trae_account_service_get_user_info_wraps_failure():
    ...

def test_trae_desktop_service_switch_account_wraps_restart_result():
    ...

def test_trae_billing_service_falls_back_to_account_token():
    ...
```

- [ ] **Step 4: Run tests to verify they fail**

Run:

```bash
pytest tests/platforms/test_trae_services.py -q
```

Expected:
- FAIL with missing service classes/functions

- [ ] **Step 5: Commit**

```bash
git add tests/platforms/test_trae_services.py
git commit -m "test: add trae service contract coverage"
```

### Task 2: 提取 Registration / Account / Desktop / Billing services

**Files:**
- Create: `platforms/trae/services/__init__.py`
- Create: `platforms/trae/services/registration.py`
- Create: `platforms/trae/services/account.py`
- Create: `platforms/trae/services/desktop.py`
- Create: `platforms/trae/services/billing.py`
- Test: `tests/platforms/test_trae_services.py`

- [ ] **Step 1: Implement minimal `TraeRegistrationService`**

Requirements:
- 接受 `RegisterConfig`、`mailbox`、`log_fn`
- 组装 `otp_callback`
- 调用现有 `TraeRegister.register`
- 返回 `Account`

- [ ] **Step 2: Implement minimal `TraeAccountService`**

Requirements:
- 封装 `check_valid`
- 封装 `get_user_info`
- 不改现有远程接口行为

- [ ] **Step 3: Implement minimal `TraeDesktopService`**

Requirements:
- 封装 `switch_account`
- 封装 `restart_trae_ide`
- 返回统一的 action 结果构造输入

- [ ] **Step 4: Implement minimal `TraeBillingService`**

Requirements:
- 创建执行器
- 调用 `TraeRegister.step4_trae_login`
- 调用 `TraeRegister.step5_get_token`
- 在拿不到新 token 时回退到 `account.token`
- 调用 `TraeRegister.step7_create_order`
- 返回统一的 action 结果构造输入

- [ ] **Step 5: Run tests to verify they pass**

Run:

```bash
pytest tests/platforms/test_trae_services.py -q
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add platforms/trae/services tests/platforms/test_trae_services.py
git commit -m "refactor: extract trae services"
```

### Task 3: 收缩 Trae 插件入口

**Files:**
- Modify: `platforms/trae/plugin.py`
- Test: `tests/platforms/test_trae_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Rewrite `TraePlatform.register` to delegate to registration service**

Requirements:
- 插件入口不再直接管理 OTP callback 细节
- 对外行为保持不变

- [ ] **Step 2: Rewrite `TraePlatform.check_valid` to delegate to account service**

- [ ] **Step 3: Rewrite `TraePlatform.execute_action` to delegate to account / desktop / billing services**

Requirements:
- `switch_account` 由 desktop service 处理
- `get_user_info` 由 account service 处理
- `get_cashier_url` 由 billing service 处理
- action id 不变
- 返回形状不变

- [ ] **Step 4: Run focused platform tests**

Run:

```bash
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_trae_services.py -q
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
git add platforms/trae/plugin.py tests/platforms
git commit -m "refactor: slim trae platform plugin"
```

### Task 4: 文档与试点验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-03-30-trae-refactor-design.md`

- [ ] **Step 1: Update plugin guidelines with Trae as the second reference implementation**

- [ ] **Step 2: Note any deviations discovered during implementation**

- [ ] **Step 3: Record whether the Cursor pattern copied cleanly to Trae**

- [ ] **Step 4: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/trae-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_trae_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 5: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-03-30-trae-refactor-design.md
git commit -m "docs: record trae refactor outcomes"
```

---

## Success Criteria

- `TraePlatform` 对外接口不变
- `plugin.py` 明显变薄
- `registration / account / desktop / billing` 边界清晰
- 专项测试存在并通过
- Cursor 的拆分模式已被 Trae 成功复用
