# Worker Protocol

当前 Python Worker 由主后端进程暴露，路由前缀为：

```text
/api/worker
```

## 1. Register

### Request

`POST /api/worker/register`

```json
{
  "platform": "trae",
  "email": "",
  "password": "",
  "count": 1,
  "concurrency": 1,
  "register_delay_seconds": 0,
  "proxy": "",
  "executor_type": "protocol",
  "captcha_solver": "yescaptcha",
  "extra": {}
}
```

### Response

```json
{
  "ok": true,
  "success_count": 1,
  "error_count": 0,
  "errors": [],
  "cashier_urls": [],
  "logs": [
    "开始注册第 1/1 个账号",
    "✓ 注册成功: user@example.com",
    "完成: 成功 1 个, 失败 0 个"
  ],
  "error": ""
}
```

语义：

- `ok=true` 表示整个批次没有失败项
- `logs` 为同步返回的执行日志，供 Go 控制面落到 `task_events`
- `cashier_urls` 为执行过程中收集到的升级链接

## 2. Check Account

### Request

`POST /api/worker/check-account`

```json
{
  "platform": "trae",
  "account_id": 1
}
```

### Response

```json
{
  "ok": true,
  "valid": true,
  "error": ""
}
```

## 3. Execute Action

### Request

`POST /api/worker/execute-action`

```json
{
  "platform": "trae",
  "account_id": 1,
  "action_id": "switch_account",
  "params": {}
}
```

### Response

```json
{
  "ok": true,
  "data": {
    "message": "done"
  },
  "error": ""
}
```

## 4. List Actions

### Request

`GET /api/worker/actions/{platform}`

### Response

```json
{
  "actions": [
    {
      "id": "sync_external",
      "label": "同步外部系统",
      "available": true,
      "availability_reason": ""
    }
  ]
}
```

## Notes

- 当前 `register`、`check-account`、`execute-action`、`actions/{platform}` 都已经可供 Go 控制面调用
- 前端不应直接调用这些 worker 接口；它们属于 Go 控制面和 Python Worker 之间的内部协议
