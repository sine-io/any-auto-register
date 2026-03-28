# Risk Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 收掉任务持久化、数据库路径、平台元数据、计数查询和平台可用性相关的结构性风险，同时保持当前应用可运行。

**Architecture:** 使用 SQLite 扩展现有持久化层，新增任务运行与任务事件表作为任务状态真相源；后端向前端暴露统一平台元数据，前端动态渲染平台与执行器；数据库路径通过环境变量控制，实现本地兼容与 Docker 持久化并存。

**Tech Stack:** FastAPI, SQLModel, SQLite, React, TypeScript, Vite, pytest

---

### Task 1: 建立测试基础设施

**Files:**
- Create: `tests/conftest.py`
- Create: `tests/test_risk_hardening.py`
- Modify: `requirements.txt`

- [ ] Step 1: 添加 `pytest` 到后端测试依赖
- [ ] Step 2: 写数据库隔离 fixture，使用临时 SQLite 文件和环境变量重载
- [ ] Step 3: 写第一个失败测试，断言 `trial_end_time` 会被持久化
- [ ] Step 4: 运行测试并确认失败原因是当前实现未保存该字段

### Task 2: 修正数据库配置和账号持久化

**Files:**
- Modify: `core/db.py`
- Modify: `api/accounts.py`
- Modify: `core/scheduler.py`

- [ ] Step 1: 将数据库 URL 改为环境变量优先
- [ ] Step 2: 在 `save_account` 的创建/更新路径写入 `trial_end_time`
- [ ] Step 3: 为账号创建/更新请求补充 `trial_end_time` 字段
- [ ] Step 4: 跑 `trial_end_time` 相关测试并确认通过

### Task 3: 引入任务运行与事件持久化

**Files:**
- Modify: `core/db.py`
- Modify: `api/tasks.py`

- [ ] Step 1: 写失败测试，断言创建任务后数据库里存在 `task_runs`
- [ ] Step 2: 写失败测试，断言日志写入后能在 `task_events` 中读取
- [ ] Step 3: 新增 `TaskRunModel` 和 `TaskEventModel`
- [ ] Step 4: 用数据库替代 `_tasks` 作为任务状态源
- [ ] Step 5: 保持任务接口返回字段兼容
- [ ] Step 6: 跑任务持久化测试并确认通过

### Task 4: 平台元数据和可用性收口

**Files:**
- Modify: `core/base_platform.py`
- Modify: `core/registry.py`
- Modify: `api/platforms.py`
- Modify: `platforms/kiro/plugin.py`
- Modify: `platforms/grok/plugin.py`
- Modify: `platforms/cursor/plugin.py`
- Modify: `platforms/trae/plugin.py`
- Modify: `platforms/chatgpt/plugin.py`
- Modify: `platforms/openblocklabs/plugin.py`
- Modify: `platforms/tavily/plugin.py`

- [ ] Step 1: 写失败测试，断言 `/api/platforms` 返回 `supported_executors`
- [ ] Step 2: 给平台基类添加可用性接口
- [ ] Step 3: 在明确依赖 Windows 的平台里声明可用性
- [ ] Step 4: 扩展平台列表接口元数据
- [ ] Step 5: 跑平台元数据测试并确认通过

### Task 5: 修正列表接口计数查询

**Files:**
- Modify: `api/accounts.py`
- Modify: `api/tasks.py`
- Test: `tests/test_risk_hardening.py`

- [ ] Step 1: 写失败测试，断言账号列表和任务历史返回的 `total` 正确
- [ ] Step 2: 将 `len(all())` 替换为数据库 `COUNT(*)`
- [ ] Step 3: 跑相关测试并确认通过

### Task 6: 前端动态化平台与执行器

**Files:**
- Modify: `frontend/src/lib/registerOptions.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/pages/Register.tsx`
- Modify: `frontend/src/pages/TaskHistory.tsx`

- [ ] Step 1: 将执行器逻辑改为基于后端平台元数据
- [ ] Step 2: 注册页平台列表改为动态加载
- [ ] Step 3: 任务历史平台筛选改为动态加载
- [ ] Step 4: 保留后端未返回扩展字段时的降级行为
- [ ] Step 5: 跑前端构建确认通过

### Task 7: Docker 持久化收口

**Files:**
- Modify: `docker-compose.yml`
- Modify: `README.md`

- [ ] Step 1: 为 Docker 显式设置 `APP_DB_URL`
- [ ] Step 2: 更新 README 中数据库路径和运行说明
- [ ] Step 3: 复核本地默认启动方式未被破坏

### Task 8: 最终验证

**Files:**
- No code changes required

- [ ] Step 1: 运行后端测试
- [ ] Step 2: 运行 `python -m compileall`
- [ ] Step 3: 运行前端构建
- [ ] Step 4: 汇总未覆盖风险和剩余限制
