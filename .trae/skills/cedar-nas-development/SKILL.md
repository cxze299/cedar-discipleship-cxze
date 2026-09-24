---
name: cedar-nas-development
description: Develops, diagnoses, deploys, and verifies Cedar Discipleship on its Synology NAS. Use for NAS debugging, database checks, releases, rollbacks, or prebuilt image transfer.
---

# Cedar NAS Development

用于 Cedar Discipleship 的本地开发、NAS 生产排查、发布和验收。

## 固定环境

```text
本地仓库: /Users/bytedance/program/agp
NAS 主机: mouss.synology.me
SSH 端口: 7
SSH 用户: yimaneili
NAS 仓库: /volume2/docker/cedar-discipleship
公开地址: https://mouss.synology.me:7399
Compose 文件: deploy/docker-compose.separated.yml
Compose 项目: cedar
容器: cedar-mysql / cedar-backend / cedar-frontend
NAS Git: /usr/local/bin/git
NAS Docker: /usr/local/bin/docker
```

## 安全规则

- 密码和 Token 仅从当前会话输入或环境变量读取。
- 禁止把凭据写入仓库、Skill、命令脚本、日志和提交。
- 禁止输出 `.env`、容器完整环境变量或登录响应中的 Token。
- 数据库排查默认只执行 `SELECT`、`SHOW`、`DESCRIBE` 和 `EXPLAIN`。
- 数据修复、迁移、删除和回滚必须先确认影响范围并保留审计依据。
- 保留 NAS 工作区已有的未跟踪资料文件，不执行 `git clean`、`reset --hard` 或 `checkout --`。
- 生产发布首选本地交叉编译并仅向 NAS 传输增量制品，不传输完整基础镜像，也不在 NAS 上编译源码。
- 首选发布在 5 分钟内无有效进展或明确失败时，才降级到 GHCR 预构建镜像；仅在用户同意应急源码发布时，才在 NAS 构建源码。

## 标准流程

### 1. 确认问题和成功标准

记录以下内容：

```text
复现账号或角色
业务日期和时区
当前小组
预期状态
实际状态
涉及页面和接口
```

先追踪完整数据链：

```text
数据库事实
-> Repository 查询
-> Service 业务判定
-> HTTP DTO
-> 前端状态合并
-> 页面展示与交互
```

同一业务状态只能有一个权威判定。统计页不得用原始记录自行重建首页已有的完成语义。

### 2. 本地预检

```bash
cd /Users/bytedance/program/agp
git status --short --branch
git log -3 --oneline
git diff --check
```

先阅读相关代码和测试，再写失败用例。修改范围保持在问题链路内。

### 3. 浏览器和 API 复现

页面问题使用 `TRAE-browseruse`：

1. 列出标签页。
2. 打开公开地址。
3. 登录由用户完成，不自动填写凭据。
4. 用快照检查文本、状态、按钮和无障碍属性。
5. 用网络请求和控制台确认接口响应及前端错误。

无需页面交互时，使用环境变量执行认证 API 验证：

```bash
payload="$(jq -n \
  --arg username "$AGP_TEST_USERNAME" \
  --arg password "$AGP_TEST_PASSWORD" \
  '{username:$username,password:$password}')"
response="$(curl -ksS \
  -X POST 'https://mouss.synology.me:7399/api/auth/login' \
  -H 'Content-Type: application/json' \
  --data "$payload")"
token="$(printf '%s' "$response" | jq -r '.token // empty')"
test -n "$token"
curl -ksS 'https://mouss.synology.me:7399/api/today?date=YYYY-MM-DD' \
  -H "Authorization: Bearer $token" |
  jq '{date,progress,tasks}'
unset token response payload
```

只输出验证所需字段。

### 4. 连接 NAS

优先使用 SSH Key：

```bash
ssh -o StrictHostKeyChecking=accept-new \
  -p 7 yimaneili@mouss.synology.me
```

密码认证时通过当前会话的安全输入或 `NAS_PASSWORD` 环境变量提供，禁止把密码写进命令正文。自动化环境无法分配终端时可使用 `expect`，密码仍从环境变量读取：

```tcl
set password $env(NAS_PASSWORD)
spawn ssh -p 7 yimaneili@mouss.synology.me {REMOTE_COMMAND}
expect "password:"
send -- "$password\r"
expect eof
```

Synology 非交互 Shell 中使用完整路径：

```bash
/usr/local/bin/git
sudo -n /usr/local/bin/docker
```

### 5. 核对部署状态

```bash
cd /volume2/docker/cedar-discipleship
/usr/local/bin/git status --short --branch
/usr/local/bin/git log -3 --oneline
sudo -n /usr/local/bin/docker ps \
  --filter name=cedar- \
  --format '{{.Names}} {{.Status}} {{.Image}}'
```

确认本地与 NAS 基线提交一致。NAS 上的未跟踪资料目录保持原状。

### 6. 只读数据库排查

通过 MySQL 容器已有环境变量连接，避免读取或打印数据库密码：

```bash
sudo -n /usr/local/bin/docker exec cedar-mysql sh -lc \
  'MYSQL_PWD="$MYSQL_PASSWORD" mysql \
    --default-character-set=utf8mb4 \
    -u"$MYSQL_USER" "$MYSQL_DATABASE" \
    --batch --raw -e "SELECT 1;"'
```

查询必须显式限定 `group_id`、用户、日期范围和 `deleted_at`。排查周任务时同时核对：

```text
study_weeks
study_tasks
task_assets
assets
checkin_records
```

视频身份以 `asset_id` 为准；`task_id` 表示一次任务配置，`week_id` 表示安排周期，`logical_date` 表示打卡发生日期。

### 7. 实现和测试

Go 改动：

```bash
cd /Users/bytedance/program/agp/backend
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go test -race ./internal/learning ./internal/checkin ./internal/statistics ./internal/server
```

前端改动：

```bash
cd /Users/bytedance/program/agp/frontend
npm test -- --run
npm run typecheck
npm run build
```

至少覆盖：

- 正常完成。
- 同资源跨周继承。
- 同周不同资源隔离。
- 已删除记录不参与统计。
- 跨组、跨用户隔离。
- 当前记录可取消，继承记录不可误删。
- 打卡后立即刷新。
- 统计页停留期间定时刷新。

### 8. 提交和推送

提交前执行：

```bash
cd /Users/bytedance/program/agp
git diff --check
git status --short
```

Commit Subject 与 Description 均使用中文：

```bash
git add <changed-files>
git commit -m "<中文类型>：<中文主题>" -m "<中文说明>"
git push origin master
```

每次提交都在 `CHANGELOG.md` 文件最前面的现有标题之后新增一段中文记录，按时间倒序排列。记录日期、关联提交或“本次提交”、影响范围及用户可感知的变更；不要只向长期累积的 `[Unreleased]` 分类追加一行。同一提交无法在自身内容中稳定记录自身哈希，使用“本次提交”，已存在的关联提交则写短提交 ID。

不提交本地私有脚本、凭据、运行日志、数据文件和构建产物。

### 9. NAS 发布

#### 首选：本地构建并传输增量制品

默认执行：

```bash
cd /Users/bytedance/program/agp
./scripts/nas-deploy-local-artifacts.sh
```

该脚本：

- 在本机构建 Linux amd64 后端二进制和前端静态文件。
- 仅传输压缩后的业务制品，复用 NAS 当前容器的基础镜像层。
- 检查当前基础镜像对应提交到目标提交之间的 Dockerfile、Nginx 和 Compose 契约；契约有变化时拒绝增量封装。
- 在 NAS 上快速封装不可变镜像，再用临时镜像环境文件执行 `up -d --no-build`。
- 不拉取 NAS 脏工作区，不覆盖 `.env`，不修改数据挂载，不留下临时构建目录。

本机和 NAS 不在同一可直连网络时，不启动临时 Web 服务；先做连通性探测，无法连接就立即放弃该方案。不要用完整镜像的 `docker save | ssh docker load` 作为默认路径，因为它会重复传输 NAS 已有的基础层。

首选流程超过 5 分钟仍无有效构建、传输或镜像封装进度时，终止残留进程、确认部署锁清理完毕，再使用下一节的 GHCR 降级方案。

#### 降级：GitHub Actions 预构建镜像

`master` 推送后，先在本机等待 GHCR manifest 可用：

```bash
cd /Users/bytedance/program/agp
sha="$(git rev-parse HEAD)"
for service in backend frontend; do
  until docker manifest inspect "ghcr.io/wangz5940/cedar-discipleship-${service}:$sha" >/dev/null 2>&1; do
    sleep 15
  done
done
```

NAS 工作区干净时可先快进同步并运行：

```bash
cd /volume2/docker/cedar-discipleship
/usr/local/bin/git pull --ff-only origin master
./scripts/nas-deploy-prebuilt.sh
```

NAS 工作区存在用户改动时，不执行 pull、reset、clean 或覆盖文件。改用目标提交对应的显式镜像标签和临时环境文件执行 `docker compose up -d --no-build`。

GHCR 拉取超过 15 分钟或连续 5 分钟没有字节进展时停止。先结束残留 `docker pull` / `docker compose pull` 客户端并等待 Docker daemon 恢复响应，不要并发启动多个拉取。

先确认 NAS 没有残留发布进程和部署锁：

```bash
ssh -p 7 yimaneili@mouss.synology.me \
  'ps -ef | grep -E "docker pull|docker compose|nas-deploy-prebuilt" | grep -v grep || true; test -d /tmp/cedar-prebuilt-deploy.lock && echo lock-present || echo lock-clear'
```

#### 应急源码构建

仅当本地增量制品和 GHCR 两种路径均超时或失败，并且用户明确同意后，才在 NAS 上源码构建受影响服务。

仅后端：

```bash
sudo -n /usr/local/bin/docker compose \
  --env-file .env \
  -p cedar \
  -f deploy/docker-compose.separated.yml \
  up -d --build backend
```

仅前端：

```bash
sudo -n /usr/local/bin/docker compose \
  --env-file .env \
  -p cedar \
  -f deploy/docker-compose.separated.yml \
  up -d --build frontend
```

前后端：

```bash
sudo -n /usr/local/bin/docker compose \
  --env-file .env \
  -p cedar \
  -f deploy/docker-compose.separated.yml \
  up -d --build backend frontend
```

发布命令保持单实例运行，持续等待构建和容器替换完成。

### 10. 上线验收

```bash
curl -ksS https://mouss.synology.me:7399/api/health
curl -ksS https://mouss.synology.me:7399/ |
  rg -o 'assets/index-[A-Za-z0-9_-]+\.js'

cd /volume2/docker/cedar-discipleship
/usr/local/bin/git rev-parse --short HEAD
sudo -n /usr/local/bin/docker ps \
  --filter name=cedar- \
  --format '{{.Names}} {{.Status}}'
sudo -n /usr/local/bin/docker logs --since 5m cedar-backend
```

认证验收需对比：

```text
/api/today
/api/dashboard/task-completions
/api/dashboard/monthly-ranking
```

检查同一账号、日期、任务在三个接口中的完成语义一致，并确认前端加载的是本次构建生成的新哈希资源。

## 发布失败处理

1. 保留失败输出并确认失败阶段。
2. 检查容器状态和最近日志。
3. 修复本地代码并重新走测试、提交、推送、拉取、构建流程。
4. 需要回退时创建反向提交：

```bash
git revert <bad-commit>
git push origin master
```

5. 在 NAS 拉取反向提交并重建受影响服务。

## 完成标准

- 本地与 NAS 均指向已推送提交。
- 受影响容器运行正常。
- `/api/health` 返回 `{"ok":true}`。
- 业务接口返回预期数据。
- 页面使用最新静态资源并展示预期状态。
- 工作区无本次任务遗留改动。
