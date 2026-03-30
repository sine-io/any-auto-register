# Worker Scaling Evaluation

本文档评估当前项目是否值得进入多 Worker / 远程 Worker / 队列化执行阶段。

## Current State

当前执行模型是：

- Go 控制面负责：
  - 管理面
  - 查询面
  - 任务创建
  - 状态与事件真相源
- Python Worker 负责：
  - 注册执行
  - 浏览器自动化
  - 邮箱收码
  - 验证码链路

默认仍偏向单机、单 Worker 运行。

## Existing Strengths

当前架构其实已经为多 Worker 做了一些准备：

- Go 与 Python 间通过 HTTP 协议通信
- Worker 回调接口已经存在
- 任务状态在 Go 控制面持久化

这意味着：

- 逻辑边界已经比纯单体强
- 后续切到远程 Worker 不需要从零重写

## Current Constraints

### 1. No Real Queue

现在任务创建后，Go 直接调用 Python Worker。

这适合单机，但不适合：

- 多 Worker 调度
- Worker 忙碌状态管理
- 重试/优先级/并发上限控制

### 2. Worker Is Stateful Around Local Runtime

Python Worker 当前默认依赖本地：

- 浏览器
- Xvfb/noVNC
- Solver
- 本地桌面切换能力

这会导致不同平台天然不适合统一横向扩展。

### 3. Not All Jobs Are Equal

任务其实分成两类：

1. **纯协议/轻执行任务**
2. **重浏览器/重桌面任务**

如果未来扩容，不应把这两类用同一种 Worker 池处理。

## Scaling Options

### Option A: Keep Single Worker, Add Limits

做法：

- 继续单 Worker
- 增加任务并发上限
- 增加队列长度控制
- 增加忙碌/拒绝策略

优点：

- 最小成本
- 不引入额外调度复杂度

缺点：

- 扩展上限有限

### Option B: Multiple Local Workers

做法：

- 在同一台机器上跑多个 Python Worker
- Go 控制面按平台/能力路由

优点：

- 比单 Worker 更强
- 仍然可以依赖本地资源

缺点：

- 浏览器、端口、显示、Solver 资源协调会更复杂

### Option C: Remote Workers + Queue

做法：

- Go 控制面前面加真正队列
- Worker 变成可远程注册的执行节点

优点：

- 真正可扩展
- 更适合未来多用户或更大规模任务

缺点：

- 当前阶段成本最高
- 需要更强的鉴权、调度、重试和可观测性

## Recommendation

当前建议顺序：

1. **短期：Option A**
   - 单 Worker + 并发限制 + 清晰的拒绝/排队策略
2. **中期：Option B**
   - 在单机上做分能力 Worker
3. **长期：Option C**
   - 只有在真实规模需求出现后再做远程 Worker + 队列

## Trigger Conditions

建议只有满足以下条件之一再推进多 Worker：

- 同一时间需要并发执行多个浏览器重任务
- 单 Worker 经常被长时间占满
- 需要把不同平台分配到不同机器能力池
- 要支持远程执行节点

## Required Foundations Before Scaling

在真正扩 Worker 之前，建议先补这些基础：

- Worker 能力标签（协议 / 浏览器 / 桌面）
- 任务并发和容量限制
- 更明确的 worker health / readiness
- 更稳定的日志与事件链路
- 数据库/队列层的并发策略

## Bottom Line

当前项目已经具备“未来可扩”的边界，但还没到“现在就该做远程 Worker”的阶段。

最合理的路径是：

- 先把单 Worker 路径继续打稳
- 再做能力分层
- 最后再谈真正的多节点调度
