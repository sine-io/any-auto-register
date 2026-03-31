# Grok Platform Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `Grok` 平台插件从“厚插件入口”重构为“薄插件 + 明确 service 边界”，同时保持当前对外行为不变。

**Architecture:** 保留 `platforms/grok/core.py` 作为浏览器自动化、Turnstile 处理和 cookie 提取层，不深改页面流。新增 `registration / cookie / sync` 三个 service，`plugin.py` 只负责平台入口、service 调度和统一结果包装；插件 action 路径外的 `services.external_sync` / `api.integrations` / `services.grok2api_runtime` 本轮保持不动。

**Tech Stack:** Python, pytest, existing Grok browser automation flow, existing platform plugin architecture

---

## File Structure

### New Files

- `platforms/grok/services/__init__.py`
  - service 包入口
- `platforms/grok/services/registration.py`
  - 负责 captcha solver 组装、邮箱 retry、OTP callback、注册编排
- `platforms/grok/services/cookie.py`
  - 负责 `check_valid`
- `platforms/grok/services/sync.py`
  - 负责 `upload_grok2api` 的插件 action 路径包装
- `tests/platforms/test_grok_services.py`
  - Grok 专项 service 测试

### Modified Files

- `platforms/grok/plugin.py`
  - 收缩为薄插件入口
- `tests/platforms/test_platform_contracts.py`
  - 保留并在需要时补 Grok 相关断言
- `docs/platform-plugin-guidelines.md`
  - 记录 Grok 作为第四个参考实现的结果
- `docs/superpowers/specs/2026-03-31-grok-refactor-design.md`
  - 记录试点 outcome / deviations / pattern assessment

### Files Explicitly Not Refactored This Round

- `platforms/grok/core.py`
- `services/external_sync.py`
- `api/integrations.py`
- `services/grok2api_runtime.py`

这些文件继续保持原样，避免把 Grok 试点扩大成“浏览器流重写 + 外部集成总线重构”。

---

### Task 1: 建立 Grok service 测试骨架

**Files:**
- Create: `tests/platforms/test_grok_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Write the failing registration service test**

```python
def test_grok_registration_service_builds_otp_callback():
    ...
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /root/any-auto-register/.worktrees/grok-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_grok_services.py::test_grok_registration_service_builds_otp_callback -q
```

Expected:
- FAIL because `GrokRegistrationService` does not exist yet

- [ ] **Step 3: Write the failing cookie / sync tests**

```python
def test_grok_registration_service_retries_rejected_mailbox_domain():
    ...

def test_grok_cookie_service_check_valid_uses_sso():
    ...

def test_grok_sync_service_upload_wraps_success():
    ...
```

- [ ] **Step 4: Run tests to verify they fail**

Run:

```bash
pytest tests/platforms/test_grok_services.py -q
```

Expected:
- FAIL with missing service classes/functions

- [ ] **Step 5: Commit**

```bash
git add tests/platforms/test_grok_services.py
git commit -m "test: add grok service contract coverage"
```

### Task 2: 提取 Registration / Cookie / Sync services

**Files:**
- Create: `platforms/grok/services/__init__.py`
- Create: `platforms/grok/services/registration.py`
- Create: `platforms/grok/services/cookie.py`
- Create: `platforms/grok/services/sync.py`
- Test: `tests/platforms/test_grok_services.py`

- [ ] **Step 1: Implement minimal `GrokRegistrationService`**

Requirements:
- 读取 yescaptcha key（任务配置优先、全局配置兜底）
- 组装 captcha solver
- 实现 mailbox retry 规则：
  - 固定 `email` 时禁止 mailbox retry
  - 自动邮箱模式下允许多次 mailbox retry
  - 每次 retry 重新申请新的 mailbox account
  - 单次尝试内复用同一个 `mail_acct` 给 `get_current_ids()` 和 `otp_callback`
- 调用现有 `GrokRegister.register`
- 返回 `Account`

- [ ] **Step 2: Implement minimal `GrokCookieService`**

Requirements:
- 封装 `check_valid`
- 保持当前语义：`bool((account.extra or {}).get("sso"))`

- [ ] **Step 3: Implement minimal `GrokSyncService`**

Requirements:
- 封装 `upload_grok2api`
- 仅负责插件 action 路径
- 不迁移 `services.external_sync.py` / `api.integrations.py` / `services.grok2api_runtime.py`
- 返回统一 action result 语义

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
pytest tests/platforms/test_grok_services.py -q
```

Expected:
- PASS

- [ ] **Step 5: Commit**

```bash
git add platforms/grok/services tests/platforms/test_grok_services.py
git commit -m "refactor: extract grok services"
```

### Task 3: 收缩 Grok 插件入口

**Files:**
- Modify: `platforms/grok/plugin.py`
- Test: `tests/platforms/test_grok_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Rewrite `GrokPlatform.register` to delegate to registration service**

Requirements:
- 插件入口不再直接管理 captcha solver / mailbox retry / OTP callback 细节
- 保留当前邮箱日志与重试行为

- [ ] **Step 2: Rewrite `GrokPlatform.check_valid` to delegate to cookie service**

- [ ] **Step 3: Rewrite `GrokPlatform.execute_action` to delegate to sync service**

Requirements:
- `upload_grok2api` 由 sync service 处理
- action id 不变
- 返回形状不变

- [ ] **Step 4: Add plugin-level delegation tests**

Requirements:
- 锁定 `register` delegation + mailbox retry/logging 关键行为
- 锁定 `execute_action("upload_grok2api")` delegation

- [ ] **Step 5: Run focused platform tests**

Run:

```bash
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_grok_services.py -q
```

Expected:
- PASS

- [ ] **Step 6: Run broader regression**

Run:

```bash
pytest tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 7: Commit**

```bash
git add platforms/grok/plugin.py tests/platforms
git commit -m "refactor: slim grok platform plugin"
```

### Task 4: 文档与试点验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-03-31-grok-refactor-design.md`

- [ ] **Step 1: Update plugin guidelines with Grok as the fourth reference implementation**

- [ ] **Step 2: Note deviations discovered during implementation**

- [ ] **Step 3: Record whether the Cursor / Trae / Kiro pattern copied cleanly to Grok**

- [ ] **Step 4: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/grok-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_grok_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 5: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-03-31-grok-refactor-design.md
git commit -m "docs: record grok refactor outcomes"
```

---

## Success Criteria

- `GrokPlatform` 对外接口不变
- `plugin.py` 明显变薄
- 注册编排与外部同步各自有独立边界
- `check_valid` 有明确 service 落点
- 专项测试存在并通过
- `Grok` 成为第四个参考实现
