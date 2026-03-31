# ChatGPT Platform Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `ChatGPT` 平台插件从“厚插件入口”重构为“薄插件 + 明确 service 边界”，同时保持当前对外行为不变。

**Architecture:** 保留 `platforms/chatgpt/register_v2.py`、`token_refresh.py`、`payment.py`、`cpa_upload.py` 作为底层实现层，不深改协议与外部同步细节。新增 `registration / token / billing / external_sync` 四个 service，`plugin.py` 只负责平台入口、service 调度和统一结果包装。此轮仅迁移 `ChatGPTPlatform` 路径；`api/chatgpt.py`、`services/external_sync.py` 等平行调用链保持不动。

**Tech Stack:** Python, pytest, existing ChatGPT registration/payment/token/upload modules, existing platform plugin architecture

---

## File Structure

### New Files

- `platforms/chatgpt/services/__init__.py`
  - service 包入口
- `platforms/chatgpt/services/registration.py`
  - 负责 mailbox adapter 组装、retry 配置、注册编排
- `platforms/chatgpt/services/token.py`
  - 负责 `check_valid` / `refresh_token`
- `platforms/chatgpt/services/billing.py`
  - 负责 `payment_link`
- `platforms/chatgpt/services/external_sync.py`
  - 负责 `upload_cpa / upload_tm`
- `tests/platforms/test_chatgpt_services.py`
  - ChatGPT 专项 service 测试

### Modified Files

- `platforms/chatgpt/plugin.py`
  - 收缩为薄插件入口
- `tests/platforms/test_platform_contracts.py`
  - 保留并在需要时补 ChatGPT 相关断言
- `docs/platform-plugin-guidelines.md`
  - 记录 ChatGPT 作为第五个参考实现的结果
- `docs/superpowers/specs/2026-03-31-chatgpt-refactor-design.md`
  - 记录试点 outcome / deviations / pattern assessment

### Files Explicitly Not Refactored This Round

- `platforms/chatgpt/register_v2.py`
- `platforms/chatgpt/token_refresh.py`
- `platforms/chatgpt/payment.py`
- `platforms/chatgpt/cpa_upload.py`
- `api/chatgpt.py`
- `services/external_sync.py`

这些文件继续保持原样，避免把 ChatGPT 试点扩大成“插件治理 + 其他直连调用链重构”的双重项目。

---

### Task 1: 建立 ChatGPT service 测试骨架

**Files:**
- Create: `tests/platforms/test_chatgpt_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Write the failing registration service tests**

```python
def test_chatgpt_registration_service_uses_generic_mailbox_adapter():
    ...

def test_chatgpt_registration_service_falls_back_to_tempmail_without_mailbox():
    ...
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd /root/any-auto-register/.worktrees/chatgpt-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_chatgpt_services.py::test_chatgpt_registration_service_uses_generic_mailbox_adapter -q
```

Expected:
- FAIL because `ChatGPTRegistrationService` does not exist yet

- [ ] **Step 3: Write the failing token / billing / external sync tests**

```python
def test_chatgpt_token_service_check_valid_uses_subscription_status():
    ...

def test_chatgpt_token_service_refresh_token_wraps_success():
    ...

def test_chatgpt_billing_service_routes_plus_and_team_links():
    ...

def test_chatgpt_external_sync_service_upload_cpa_wraps_success():
    ...

def test_chatgpt_external_sync_service_upload_tm_wraps_success():
    ...
```

- [ ] **Step 4: Run tests to verify they fail**

Run:

```bash
pytest tests/platforms/test_chatgpt_services.py -q
```

Expected:
- FAIL with missing service classes/functions

- [ ] **Step 5: Commit**

```bash
git add tests/platforms/test_chatgpt_services.py
git commit -m "test: add chatgpt service contract coverage"
```

### Task 2: 提取 Registration / Token / Billing / External Sync services

**Files:**
- Create: `platforms/chatgpt/services/__init__.py`
- Create: `platforms/chatgpt/services/registration.py`
- Create: `platforms/chatgpt/services/token.py`
- Create: `platforms/chatgpt/services/billing.py`
- Create: `platforms/chatgpt/services/external_sync.py`
- Test: `tests/platforms/test_chatgpt_services.py`

- [ ] **Step 1: Implement minimal `ChatGPTRegistrationService`**

Requirements:
- 组装 mailbox adapter
- 保留固定 `email` + mailbox 场景
- 保留无 mailbox 时默认 TempMail fallback
- 读取 `register_max_retries`
- 调用 `RegistrationEngineV2`
- 返回 `Account`

- [ ] **Step 2: Implement minimal `ChatGPTTokenService`**

Requirements:
- 封装 `check_valid`
- 封装 `refresh_token`
- 保留当前 duck-typed account adapter 兼容字段

- [ ] **Step 3: Implement minimal `ChatGPTBillingService`**

Requirements:
- 封装 `payment_link`
- 处理 `plus / team` 路由
- 保留 country / plan 参数语义

- [ ] **Step 4: Implement minimal `ChatGPTExternalSyncService`**

Requirements:
- 封装 `upload_cpa`
- 封装 `upload_tm`
- 保留当前 `generate_token_json` / `upload_to_cpa` / `upload_to_team_manager` 路径

- [ ] **Step 5: Run tests to verify they pass**

Run:

```bash
pytest tests/platforms/test_chatgpt_services.py -q
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add platforms/chatgpt/services tests/platforms/test_chatgpt_services.py
git commit -m "refactor: extract chatgpt services"
```

### Task 3: 收缩 ChatGPT 插件入口

**Files:**
- Modify: `platforms/chatgpt/plugin.py`
- Test: `tests/platforms/test_chatgpt_services.py`
- Test: `tests/platforms/test_platform_contracts.py`

- [ ] **Step 1: Rewrite `ChatGPTPlatform.register` to delegate to registration service**

Requirements:
- 插件入口不再直接构造 mailbox adapter
- 保留当前邮箱/密码与返回行为

- [ ] **Step 2: Rewrite `ChatGPTPlatform.check_valid` to delegate to token service**

- [ ] **Step 3: Rewrite `ChatGPTPlatform.execute_action` to delegate to token / billing / external sync services**

Requirements:
- `refresh_token` 由 token service 处理
- `payment_link` 由 billing service 处理
- `upload_cpa / upload_tm` 由 external sync service 处理
- action id 不变
- 返回形状不变

- [ ] **Step 4: Add plugin-level delegation tests**

Requirements:
- 锁定 `register` delegation + adapter selection 关键行为
- 锁定 `execute_action` delegation

- [ ] **Step 5: Run focused platform tests**

Run:

```bash
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/platforms/test_chatgpt_services.py -q
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
git add platforms/chatgpt/plugin.py tests/platforms
git commit -m "refactor: slim chatgpt platform plugin"
```

### Task 4: 文档与试点验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-03-31-chatgpt-refactor-design.md`

- [ ] **Step 1: Update plugin guidelines with ChatGPT as the fifth reference implementation**

- [ ] **Step 2: Note deviations discovered during implementation**

- [ ] **Step 3: Record whether the Cursor / Trae / Kiro / Grok pattern copied cleanly to ChatGPT**

- [ ] **Step 4: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/chatgpt-refactor-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/platforms/test_chatgpt_services.py tests/platforms/test_platform_contracts.py tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 5: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-03-31-chatgpt-refactor-design.md
git commit -m "docs: record chatgpt refactor outcomes"
```

---

## Success Criteria

- `ChatGPTPlatform` 对外接口不变
- `plugin.py` 明显变薄
- register / token / billing / external sync 各自有明确 service 边界
- 专项测试存在并通过
- `ChatGPT` 成为第五个参考实现
