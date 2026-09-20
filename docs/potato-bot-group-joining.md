# Potato 机器人加入群聊能力

核对日期：2026-09-20

官方文档：

- [Potato Bot API](https://m.potatochat.app/api/bot)
- [Potato Bots introduction](https://m.potatochat.app/bots)

## 结论

Potato Bot API 没有提供让机器人主动加入指定群聊的 `joinChat`、`joinGroup` 或邀请链接导入接口。

官方提供的 `setJoinGroups` 只控制机器人是否允许被用户添加到群聊。实际加入动作仍需由有邀请权限的群成员或管理员在 Potato 客户端中完成。加入后，服务端可以通过 `getGroups` 发现群聊，再在本系统中把该群聊分配给学习小组。

## 允许加入群聊

### 请求

```text
POST https://api.rct2008.com:8443/<bot_token>/setJoinGroups
Content-Type: application/json
```

```json
{
  "enable": true
}
```

参数：

| 参数 | 必填 | 类型 | 含义 |
| --- | --- | --- | --- |
| `enable` | 是 | Boolean | `true` 允许用户把机器人添加到群聊；`false` 禁止 |

认证方式：

- Bot Token 由 Potato `@BotFather` 创建机器人时签发。
- Token 作为 HTTPS URL 路径的一部分传递，不使用额外的 Authorization Header。
- Token 必须只保存在服务端密钥配置中，不得写入前端、日志或 API 响应。

成功响应：

```json
{
  "ok": true
}
```

失败响应遵循 Potato Bot API 通用格式：

```json
{
  "ok": false,
  "description": "human-readable error",
  "error_code": 1002
}
```

`error_code` 的具体值可能变化，调用方不应把它作为永久协议常量。

## 查询已加入群聊

### 请求

```text
GET https://api.rct2008.com:8443/<bot_token>/getGroups
```

无请求参数，认证方式与 `setJoinGroups` 相同。

成功响应：

```json
{
  "ok": true,
  "result": {
    "Groups": [
      {"PeerID": 10945523, "PeerName": "普通群"}
    ],
    "SuperGroups": [
      {"PeerID": 11705614, "PeerName": "超级群"}
    ],
    "Channels": []
  }
}
```

本系统仅接受普通群和超级群，分别映射为 `chat_type=2` 和 `chat_type=3`。

## 支持的接入流程

1. 使用 `@BotFather` 创建机器人并取得 Token。
2. 调用 `setJoinGroups`，设置 `enable=true`。
3. 群成员或管理员在 Potato 客户端中搜索机器人用户名并将其加入目标群。
4. 本系统调用 `getMe` 验证 Token，调用 `getGroups` 获取已加入群聊。
5. 超级管理员在“管理后台 → 机器人管理”中把群聊分配给学习小组。

普通文本发送不要求机器人拥有群管理员权限，但机器人必须仍在群内且具备发言能力。置顶、踢人、修改群信息等管理接口要求相应的群管理员权限；本系统不使用这些权限。

## 限流

Potato 官方文档没有公布 `setJoinGroups`、`getMe` 或 `getGroups` 的数值限流配额。调用方应：

- 避免轮询；仅在管理页面刷新或必要的绑定校验时调用。
- 对 HTTP `429` 遵守 `Retry-After`。
- 对网络错误和 `5xx` 使用有上限的退避重试。
- 不对认证或权限错误自动重试。

本系统的消息队列对 `sendTextMessage` 使用单队列每秒最多一段、最多五次重试，并隔离每个机器人的队列和失败状态。

## 无直接加入 API 时的替代方案

推荐方案是保留人工授权步骤：通过机器人用户名或 `potato.im/<bot_username>` 链接打开机器人资料，由目标群管理员完成添加。这确保群成员关系由群管理员明确授权。

不建议用个人账号客户端协议模拟加群。该方案需要保存个人账号凭据、处理验证码和风控，并扩大账号封禁与越权风险。如果未来必须自动化，应作为独立受控服务设计，使用专用账号、最小权限、审批、审计和速率限制，不应混入当前 Bot Token 服务。

## 本系统的多机器人管理接口

两个接口都要求已登录的超级管理员身份，使用系统现有的 Bearer Token 鉴权。

### 查询机器人状态

```text
GET /api/super-admin/bot-management
Authorization: Bearer <access_token>
```

响应：

```json
{
  "configured": true,
  "robots": [
    {
      "id": "primary",
      "name": "主机器人",
      "state": "healthy",
      "authenticated": true,
      "identity": {
        "id": 10100427,
        "first_name": "Primary Bot",
        "username": "primary_bot"
      },
      "last_checked_at": "2026-09-20T03:30:00Z",
      "queue": {
        "pending": 0,
        "completed": 12,
        "failed": 0
      },
      "chats": [
        {
          "chat_id": 11705614,
          "chat_type": 3,
          "title": "学习群",
          "group_id": 1,
          "joined": true
        }
      ],
      "bindings": [
        {
          "chat_id": 11705614,
          "chat_type": 3,
          "group_id": 1
        }
      ]
    }
  ],
  "study_groups": []
}
```

`state` 为 `healthy`、`degraded` 或 `unavailable`。`error_code` 只返回稳定分类，不返回 Token、上游 URL 或响应正文。

### 分配通知目标

```text
PUT /api/super-admin/bot-bindings
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "robot_id": "primary",
  "chat_id": 11705614,
  "chat_type": 3,
  "group_id": 1
}
```

`group_id=0` 解除绑定。绑定前会使用该机器人的 `getMe` 验证 Token，并使用 `getGroups` 验证机器人仍在目标群中。成功响应为：

```json
{
  "ok": true
}
```

### 新增机器人

```text
POST /api/super-admin/bot-robots
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "id": "primary",
  "name": "主机器人",
  "token": "123:secret"
}
```

`token` 必填；`id` 和 `name` 可省略。省略 `id` 时，服务端会在 `getMe` 验证通过后根据机器人用户名生成稳定 ID；省略 `name` 时使用机器人名称或用户名。

新增时服务端会：

1. 校验 Token 格式。
2. 调用 Potato `getMe` 验证 Token。
3. 拒绝重复机器人 ID、重复 Token 或超过 32 个机器人。
4. 将页面新增的机器人写入 `${AGP_NOTIFICATION_DIR}/robots.json`，文件权限为 `0600`。
5. 立即启动该机器人的独立通知队列。

响应：

```json
{
  "robot": {
    "id": "primary",
    "name": "主机器人",
    "state": "healthy",
    "authenticated": true,
    "identity": {
      "id": 10100427,
      "first_name": "Primary Bot",
      "username": "primary_bot"
    },
    "last_checked_at": "2026-09-20T03:30:00Z",
    "queue": {
      "pending": 0,
      "completed": 0,
      "failed": 0
    },
    "chats": [],
    "bindings": []
  }
}
```

Token 只保存在后端配置文件，不会出现在 API 响应、审计日志或前端状态中。部署环境变量中的机器人仍由部署配置管理；如果环境变量与页面保存的机器人使用同一个 ID，环境变量优先生效。
