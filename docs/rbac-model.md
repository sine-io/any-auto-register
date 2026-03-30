# RBAC Model

本文档定义如果项目未来进入多人模式，推荐的最小 RBAC（Role-Based Access Control）模型。

## Goal

RBAC 的目标不是做复杂权限系统，而是先解决三个问题：

1. 谁可以看数据
2. 谁可以改配置
3. 谁可以执行高风险动作

## Recommended Roles

### 1. Owner

权限：

- 管理 workspace
- 管理成员
- 修改全局配置
- 执行所有任务与动作
- 删除账号、任务、代理

### 2. Admin

权限：

- 管理账号、任务、代理
- 修改业务配置
- 执行大部分动作

限制：

- 不能删除 workspace
- 不能管理 owner

### 3. Operator

权限：

- 创建任务
- 查看账号/任务
- 执行低风险动作

限制：

- 不能改关键配置
- 不能批量删除重要资源

### 4. Viewer

权限：

- 只读查看

限制：

- 不能创建任务
- 不能执行动作
- 不能修改配置

## Permission Groups

建议不要直接给每个接口配独立权限，而是先分组：

- `config.read`
- `config.write`
- `account.read`
- `account.write`
- `account.delete`
- `task.read`
- `task.write`
- `task.delete`
- `proxy.read`
- `proxy.write`
- `integration.manage`
- `action.execute`
- `workspace.manage`
- `membership.manage`

## High-Risk Operations

这些操作建议默认只给 `owner/admin`：

- 修改全局配置
- 删除账号
- 批量删除任务历史
- 重启 solver
- 启停 integrations
- 执行桌面切换类 action

## Why Action-Level Policy Matters

当前平台动作差异很大：

- 有些只是读用户信息
- 有些会重启本地桌面客户端
- 有些会写外部系统

因此未来最好区分：

- `action.read_only`
- `action.external_sync`
- `action.desktop_switch`

## Minimal Enforcement Order

如果以后开始落 RBAC，推荐顺序：

1. 先做路由级角色判断
2. 再做资源级 workspace 过滤
3. 最后做 action/category 细粒度控制

## Recommendation

当前阶段先记住两点：

- `viewer / operator / admin / owner` 这四级已经足够
- 不要现在就设计过细的 permission matrix
