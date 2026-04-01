# Integration Path Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `ChatGPT` 与 `Grok` 的关键旁路调用链复用已有 service，而不是继续直接调用旧底层模块，同时不改变现有对外 API 协议。

**Architecture:** 本轮不深改 `core.py / switch.py / legacy module`。`ChatGPT` 侧主要收口 `api/chatgpt.py` 到已有 `token / billing / external_sync` services；`Grok` 侧主要收口 `services.external_sync.py` 与 `api/integrations.py` 到 `GrokSyncService`。runtime readiness (`services.grok2api_runtime.py`) 继续保留在调用方一侧，配置 fallback 语义保持当前差异化来源。

**Tech Stack:** Python, FastAPI, sqlmodel, existing platform service modules, pytest

---

## File Structure

### Modified Files

- `api/chatgpt.py`
  - 改为走 ChatGPT services，而不是直连 legacy 模块
- `services/external_sync.py`
  - Grok 分支改为走 `GrokSyncService.upload_grok2api_raw(...)`
- `api/integrations.py`
  - Grok backfill 分支改为走 `GrokSyncService.upload_grok2api_raw(...)`
- `platforms/chatgpt/services/token.py`
  - 增加 raw side-path API
- `platforms/chatgpt/services/billing.py`
  - 增加 raw side-path API
- `platforms/chatgpt/services/external_sync.py`
  - 增加 raw side-path API
- `platforms/grok/services/sync.py`
  - 增加 raw side-path API
- `tests/test_integration_path_unification.py`
  - 新增旁路调用链统一测试
- `docs/platform-plugin-guidelines.md`
  - 回填实现结果
- `docs/superpowers/specs/2026-04-01-integration-path-unification-design.md`
  - 回填实现结果

### Files Explicitly Not Refactored This Round

- `platforms/chatgpt/register_v2.py`
- `platforms/chatgpt/token_refresh.py`
- `platforms/chatgpt/payment.py`
- `platforms/chatgpt/cpa_upload.py`
- `platforms/grok/core.py`
- `services/grok2api_runtime.py`
- 任何 Go 控制面代码

---

### Task 1: 建立旁路调用链测试骨架

**Files:**
- Create: `tests/test_integration_path_unification.py`

- [ ] **Step 1: Write failing ChatGPT side-path tests**

至少覆盖：

```python
def test_chatgpt_api_refresh_token_uses_token_service_raw_method():
    ...

def test_chatgpt_api_payment_link_uses_billing_service_raw_method():
    ...

def test_chatgpt_api_subscription_uses_token_service_raw_method():
    ...

def test_chatgpt_api_upload_cpa_uses_external_sync_service_raw_method():
    ...
```

- [ ] **Step 2: Write failing Grok side-path tests**

至少覆盖：

```python
def test_external_sync_routes_grok_through_grok_sync_service_raw_method():
    ...

def test_integrations_backfill_routes_grok_through_grok_sync_service_raw_method():
    ...
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
cd /root/any-auto-register/.worktrees/integration-paths-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/test_integration_path_unification.py -q
```

Expected:
- FAIL because the side paths have not been rewired yet

- [ ] **Step 4: Commit**

```bash
git add tests/test_integration_path_unification.py
git commit -m "test: add integration path unification coverage"
```

### Task 2: 扩展 service 层 raw side-path API

**Files:**
- Modify: `platforms/chatgpt/services/token.py`
- Modify: `platforms/chatgpt/services/billing.py`
- Modify: `platforms/chatgpt/services/external_sync.py`
- Modify: `platforms/grok/services/sync.py`
- Test: `tests/test_integration_path_unification.py`

- [ ] **Step 1: Add ChatGPT raw token APIs**

Requirements:
- `refresh_account_raw(account, proxy=None)`
- `get_subscription_status_raw(account, proxy=None)`
- 保持当前 API 旁路 proxy 语义

- [ ] **Step 2: Add ChatGPT raw billing API**

Requirements:
- `generate_payment_link_raw(account, plan, country, proxy=None, workspace_name="MyTeam", seat_quantity=5, price_interval="month")`
- 保留当前 Team 路径参数

- [ ] **Step 3: Add ChatGPT raw external sync API**

Requirements:
- `upload_cpa_raw(...) -> tuple[bool, str]`
- 仅覆盖当前 `/upload-cpa` API 路径

- [ ] **Step 4: Add Grok raw sync API**

Requirements:
- `upload_grok2api_raw(account, api_url=None, app_key=None) -> tuple[bool, str]`
- 传入显式参数时使用显式参数
- 不传参数时继续依赖下层现有 config fallback

- [ ] **Step 5: Run focused tests**

Run:

```bash
pytest tests/test_integration_path_unification.py -q
pytest tests/platforms/test_grok_services.py -q
pytest tests/platforms/test_chatgpt_services.py -q
pytest tests/platforms/test_platform_contracts.py -q
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add platforms/chatgpt/services/token.py platforms/chatgpt/services/billing.py platforms/chatgpt/services/external_sync.py platforms/grok/services/sync.py tests/test_integration_path_unification.py
 git commit -m "refactor: add raw service APIs for side paths"
```

### Task 3: 收口 ChatGPT 与 Grok 的旁路调用链

**Files:**
- Modify: `api/chatgpt.py`
- Modify: `services/external_sync.py`
- Modify: `api/integrations.py`
- Test: `tests/test_integration_path_unification.py`

- [ ] **Step 1: Rewire ChatGPT API paths**

Requirements:
- `/refresh-token` 走 `ChatGPTTokenService.refresh_account_raw(...)`
- `/subscription` 走 `ChatGPTTokenService.get_subscription_status_raw(...)`
- `/payment-link` 走 `ChatGPTBillingService.generate_payment_link_raw(...)`
- `/upload-cpa` 走 `ChatGPTExternalSyncService.upload_cpa_raw(...)`
- 保持当前 response shape 与 DB 更新语义

- [ ] **Step 2: Rewire Grok auto-sync path**

Requirements:
- `services.external_sync.py` 的 Grok 分支继续：
  - 先 `ensure_grok2api_ready()`
  - 再 `GrokSyncService.upload_grok2api_raw(account)`
- 不显式传 `api_url/app_key`

- [ ] **Step 3: Rewire Grok backfill path**

Requirements:
- `api/integrations.py` 的 Grok 分支继续：
  - 先 `ensure_grok2api_ready()`
  - 再把显式 fallback 解析出的 `api_url/app_key` 传给 `GrokSyncService.upload_grok2api_raw(...)`
- 保持当前 backfill summary shape

- [ ] **Step 4: Run focused tests**

Run:

```bash
pytest tests/test_integration_path_unification.py -q
```

Expected:
- PASS

- [ ] **Step 5: Run broader regression**

Run:

```bash
pytest tests/platforms/test_grok_services.py -q
pytest tests/platforms/test_chatgpt_services.py -q
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/test_risk_hardening.py -q
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add api/chatgpt.py services/external_sync.py api/integrations.py tests/test_integration_path_unification.py
 git commit -m "refactor: unify chatgpt and grok side paths"
```

### Task 4: 文档与最终验收

**Files:**
- Modify: `docs/platform-plugin-guidelines.md`
- Modify: `docs/superpowers/specs/2026-04-01-integration-path-unification-design.md`

- [ ] **Step 1: Update docs with final side-path boundary rules**

至少写清：
- ChatGPT 侧哪些 API 已统一到 service
- Grok 侧哪些旁路已统一到 service
- runtime readiness 与 service 边界如何分工

- [ ] **Step 2: Record fallback ownership explicitly**

尤其写清：
- Grok auto-sync 继续走下层 config fallback
- Grok backfill 继续走 API 层显式 fallback

- [ ] **Step 3: Final verification**

Run:

```bash
cd /root/any-auto-register/.worktrees/integration-paths-spec
source /root/any-auto-register/.venv/bin/activate
pytest tests/test_integration_path_unification.py -q
pytest tests/platforms/test_grok_services.py -q
pytest tests/platforms/test_chatgpt_services.py -q
pytest tests/platforms/test_platform_contracts.py -q
pytest tests/test_risk_hardening.py -q
cd go-control-plane && go test ./...
cd ../frontend && npm run build
```

Expected:
- all green

- [ ] **Step 4: Commit**

```bash
git add docs/platform-plugin-guidelines.md docs/superpowers/specs/2026-04-01-integration-path-unification-design.md
 git commit -m "docs: record integration path unification outcomes"
```

---

## Success Criteria

- ChatGPT `/refresh-token` / `/payment-link` / `/subscription` / `/upload-cpa` 旁路路径复用 ChatGPT services
- Grok 自动同步 / backfill 路径复用 `GrokSyncService`
- runtime readiness 与 sync service 边界清晰
- 现有对外 API 形状不变
- 不引入新的平台行为漂移
