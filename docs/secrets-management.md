# Secrets Management

本文档描述如果系统进入多人或长期在线阶段，凭据与敏感信息应如何管理。

## Current State

当前系统里这些数据仍属于高敏感：

- 账号密码
- token / refresh token
- client secret
- API keys
- 管理口令

而且很多信息当前仍会：

- 存 SQLite
- 被控制面读取
- 被前端展示或编辑

## Current Risk

在单机工具阶段，这种方式还能接受；
在多人或长期部署阶段，风险会迅速上升。

## Secret Categories

建议至少分三类：

### 1. System Secrets

例如：

- `AAR_INTERNAL_CALLBACK_TOKEN`
- 第三方平台 API key
- CPA / Team Manager / 管理面 token

特点：

- 属于系统级
- 不应明文返回到前端

### 2. User Secrets

例如：

- 平台账号密码
- access token / refresh token

特点：

- 属于资源级敏感数据
- 未来需要 workspace 边界

### 3. Runtime Secrets

例如：

- Worker 间调用 token
- 临时验证码
- 临时 OAuth state

特点：

- 不一定需要持久化
- 更适合短生命周期内存态

## Recommended Direction

### Short Term

继续当前做法，但至少保证：

- 前端读取 secret 时默认掩码
- 内部 callback token 不明文暴露
- 审计日志不写 secret 值

### Mid Term

引入更明确的 secret 分层：

- 系统配置 secret 与普通配置分离
- 用户凭据与普通资源字段分离

### Long Term

如果进入多人或生产长期在线：

- 数据库中的高敏感字段应加密存储
- 密钥不应与业务库放在一起
- 最终可以接外部 secret store

## Encryption Guidance

如果以后开始做加密，建议：

- 只加密真正高敏感字段
- 不要全表全字段盲目加密
- 保留必要的可查询字段明文索引

## UI Guidance

前端不应再把 secret 当普通字段处理。

建议原则：

- 默认掩码展示
- 修改时只提交变更值
- 不提供“读出原 secret”能力

## Logging Guidance

永远不要把这些内容打进日志：

- access token
- refresh token
- password
- client secret
- 管理口令

日志里只保留：

- action name
- resource id
- result
- actor

## Recommendation

当前阶段不要直接做全面 secrets 重构。

更合理的路线是：

1. 先保持掩码和最小审计
2. 未来做用户/权限模型时再引入加密边界
3. 最终再考虑专门的 secret store
