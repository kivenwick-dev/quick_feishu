# QuickFeishu 容器化部署

本文使用项目根目录的 `Dockerfile` 与 `compose.yaml` 部署 QuickFeishu。镜像内包含 Vue 前端和 Go 服务，运行时不需要 Node.js 或 Go。

## 1. 部署结构

容器内关键路径：

| 路径 | 用途 | 持久化卷 |
| --- | --- | --- |
| `/app/quick-feishu` | 应用程序 | 镜像内文件 |
| `/app/config/config.yaml` | 页面设置、定时任务与模板配置 | `quick_feishu_config` |
| `/app/data/accounts/<user_id>/quick-feishu.db` | 各账号独立 SQLite 数据库 | `quick_feishu_data` |

容器使用 UID/GID `10001` 的非 root 用户运行，并启用 `no-new-privileges`。镜像内置健康检查，每 30 秒访问一次 `/api/dashboard`。

## 2. 环境要求

- Docker Engine 24 或更高版本
- Docker Compose v2（使用 `docker compose` 命令）
- 服务器能够访问 `api.quickrouter.ai` 和飞书开放平台
- 至少 1 GB 可用内存用于构建镜像

确认环境：

```bash
docker version
docker compose version
```

## 3. 首次部署

克隆项目并准备端口配置：

```bash
git clone git@github.com:kivenwick-dev/quick_feishu.git
cd quick_feishu
cp .env.example .env
```

`.env` 默认只需要配置监听地址和端口：

```dotenv
QF_BIND_ADDRESS=127.0.0.1
QF_PORT=8080
```

账号 ID、系统令牌和飞书 Webhook 不需要写入 `.env`，首次启动后直接在前端「设置」页面配置。页面保存的内容写入持久化卷中的 `/app/config/config.yaml`，重新创建或升级容器不会丢失。

构建并启动：

```bash
docker compose up -d --build
```

查看状态和日志：

```bash
docker compose ps
docker compose logs -f --tail=200 quick-feishu
```

默认访问地址：<http://127.0.0.1:8080>。

首次进入页面后，在「设置」中填写账号 ID、系统令牌、飞书 Webhook，并确认 API 地址、快照时间和日报时间，然后点击「保存并应用」。程序会按账号自动切换独立数据库并开始更新数据。

对于不希望通过页面配置的无人值守部署，也可以在 `.env` 中添加以下可选变量：

```dotenv
QR_USER_ID=822572
QR_SYSTEM_TOKEN=替换为真实系统令牌
QR_FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/替换为真实地址
```

环境变量的优先级高于前端配置，对应字段会在页面中锁定。修改环境变量后需要重新创建容器：

```bash
docker compose up -d --force-recreate
```

`.env` 可能包含敏感信息，不要提交到 Git。项目的 `.gitignore` 已忽略该文件。

## 4. 网络与公网访问

QuickFeishu 当前没有登录鉴权。`compose.yaml` 默认将端口绑定到 `127.0.0.1`，适合通过 Nginx、Caddy 或其他网关增加 HTTPS 和身份认证后访问。

如果只在可信内网直接访问，可在 `.env` 中设置：

```dotenv
QF_BIND_ADDRESS=0.0.0.0
```

不要在没有防火墙、HTTPS 和身份认证的情况下将管理页面直接暴露到公网。

Nginx 反向代理示例：

```nginx
server {
    listen 443 ssl http2;
    server_name quick-feishu.example.com;

    # ssl_certificate 与认证配置按实际环境填写

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 5. 常用运维命令

```bash
# 启动
docker compose up -d

# 停止但保留配置和数据库
docker compose down

# 重启
docker compose restart quick-feishu

# 查看最近日志
docker compose logs --tail=200 quick-feishu

# 在容器中手动采集快照
docker compose exec quick-feishu /app/quick-feishu snapshot

# 在容器中手动发送日报
docker compose exec quick-feishu /app/quick-feishu run-report
```

不要使用 `docker compose down -v`，该命令会删除配置卷和数据库卷。

## 6. 升级

升级前先执行下一节的备份，然后运行：

```bash
git pull --ff-only origin main
docker compose build --pull
docker compose up -d
docker compose ps
```

新容器会继续挂载原有配置卷和数据卷。应用启动时会自动执行数据库结构迁移。

## 7. 备份与恢复

备份配置和所有账号数据库：

```bash
mkdir -p backups
docker compose exec -T quick-feishu \
  sh -c 'tar -C /app -czf - config data' \
  > "backups/quick-feishu-$(date +%Y%m%d-%H%M%S).tar.gz"
```

检查备份内容：

```bash
tar -tzf backups/quick-feishu-YYYYMMDD-HHMMSS.tar.gz
```

恢复会覆盖当前配置和数据库。确认备份文件无误后执行：

```bash
docker compose down
docker compose run --rm --no-deps --entrypoint sh quick-feishu -c \
  'find /app/config -mindepth 1 -delete; find /app/data -mindepth 1 -delete; tar -C /app -xzf -' \
  < backups/quick-feishu-YYYYMMDD-HHMMSS.tar.gz
docker compose up -d
```

## 8. 使用宿主机目录代替命名卷

如需直接查看数据库文件，可把 `compose.yaml` 的卷修改为：

```yaml
volumes:
  - ./runtime/config:/app/config
  - ./runtime/data:/app/data
```

首次启动前创建目录并设置权限：

```bash
mkdir -p runtime/config runtime/data
sudo chown -R 10001:10001 runtime
docker compose up -d --build
```

此时账号数据库位于 `runtime/data/accounts/<user_id>/quick-feishu.db`。

## 9. 故障排查

### 容器反复重启

```bash
docker compose ps
docker compose logs --tail=300 quick-feishu
```

重点检查 `.env` 是否填写、配置卷是否可写，以及 `QF_PORT` 映射的宿主机端口是否被占用。

### 健康检查失败

```bash
docker inspect --format '{{json .State.Health}}' quick-feishu
docker compose exec quick-feishu wget -qO- http://127.0.0.1:8080/api/dashboard
```

### 切换账号后看不到旧历史

每个账号使用独立数据库。确认 `.env` 中的 `QR_USER_ID` 与系统令牌属于同一账号，并检查数据卷中是否存在相应账号目录。程序会拒绝把接口返回账号与配置账号不一致的数据写入数据库。

### 修改 `.env` 后没有生效

`docker compose restart` 不会重新读取 Compose 环境变量。请使用：

```bash
docker compose up -d --force-recreate
```
