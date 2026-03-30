# Platform Plugin Guidelines

本文档定义平台插件的最小契约，目标是让 `platforms/*/plugin.py` 的行为可预测、可测试、可由控制面稳定消费。

## Required Metadata

每个平台插件都应显式定义：

- `name`
- `display_name`
- `version`
- `supported_executors`

## Action Contract

`get_platform_actions()` 返回列表时，每个 action 至少应包含：

- `id`
- `label`
- `params`

其中：

- `id` 必须稳定，可作为 API/前端路由参数
- `label` 用于面向用户展示
- `params` 必须始终存在，即使为空也要返回 `[]`

## Execute Action Contract

`execute_action()` 应遵循统一返回形状：

- 成功：`{"ok": true, "data": ...}`
- 失败：`{"ok": false, "error": "..."}` 

不要在失败时只返回 `data.message`，否则控制面和前端需要为每个平台写特殊分支。

## Token Update Guidance

如果动作执行会刷新 token 或生成新的认证材料：

- 优先放入 `data`
- 字段名尽量使用当前系统已经追踪的名字
- 避免同一平台混用多套命名风格

## Error Handling Guidance

- 缺少关键凭据时，优先返回明确的 `error` 字段，而不是抛异常
- 未知 action 可以继续抛 `NotImplementedError`
- 外部依赖失败时，应尽量转换成可读错误字符串

## Current Priority Targets

这一轮优先治理：

- `Cursor`
- `Trae`
- `Grok`

原因：

- 它们在桌面切换、升级链接、外部上传这类动作上最常被直接调用
- 也是最容易出现返回形状不一致的平台

## Future Cleanup Candidates

后续可以继续拆分的典型问题：

- 插件层同时承担业务决策、外部 IO、桌面客户端控制
- 动作返回值未统一，控制面不得不做兼容处理
- `platforms/*/core.py` 中协议逻辑和副作用逻辑混在一起，不利于单测
