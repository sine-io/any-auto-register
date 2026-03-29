# Rollback Plan

本文档描述双后端部署出现异常时的回退路径。

## 1. 代码回滚

如果问题来自最近一次发布：

```bash
git log --oneline -n 10
git checkout <known-good-commit>
```

如果你通过镜像发布，优先回滚镜像 tag，而不是在线改工作区。

## 2. 容器回滚

停止当前双后端：

```bash
docker compose -f docker-compose.control-plane.yml down
```

切回单 Python 方案：

```bash
docker compose -f docker-compose.yml up -d --build
```

## 3. 数据回滚

当前默认使用 SQLite：

- `data/account_manager.db`

回滚前先备份：

```bash
cp data/account_manager.db data/account_manager.db.bak.$(date +%Y%m%d%H%M%S)
```

如果只是应用层回滚，优先保留现有数据库，不要直接覆盖。

## 4. 网关回滚

如果问题只在 Go 控制面或 `/api-go` 路由：

- 先停掉双后端 compose
- 切回单 Python compose
- 外部访问继续走原 `docker-compose.yml`

## 5. 最小回滚验收

回滚后至少确认：

- 首页能正常打开
- `/api/platforms` 可访问
- 基础账号列表可打开
- 注册任务能创建

## 6. 不建议的操作

- 不要直接删除 `data/` 再重建
- 不要在未备份 SQLite 的情况下做 destructive reset
- 不要同时保留两套占用同一主机端口的 compose 栈
