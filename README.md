# QuickFeishu 使用说明

QuickFeishu 用于定时采集 QuickRouter 账号与令牌用量，保存历史快照，并通过飞书机器人发送日报。程序为单个 Go 服务，内置 Web 管理页面，数据保存在本地 SQLite。

生产环境建议使用 Docker Compose 部署，完整步骤参见 [容器化部署文档](docs/docker-deployment.md)。

## 1. 准备信息

使用前准备：

- QuickRouter 账号 ID（`user_id`）
- QuickRouter 系统令牌（`system_token`）
- 飞书群自定义机器人 Webhook 地址

## 2. 启动与配置

### 从源码运行

首次运行先安装前端依赖并构建页面：

```bash
cd frontend
npm ci
npm run build
cd ..
go run . serve
```

后续如果没有修改前端，可直接运行：

```bash
go run . serve
```

终端显示 `QuickFeishu running at http://localhost:<实际端口>/` 后即可访问。前台运行时按 `Ctrl+C` 结束进程。

### 使用已构建程序

选择与系统匹配的文件，Linux/macOS 先增加执行权限：

```bash
chmod +x quick-feishu-linux-amd64
./quick-feishu-linux-amd64 serve
```

不带参数启动时也默认执行 `serve`。服务默认监听 `0.0.0.0:8080`；如果端口被占用，会依次尝试后续端口。控制台会输出实际访问地址。

前台运行时按 `Ctrl+C` 停止。如果需要后台运行并保留日志：

```bash
nohup ./quick-feishu-linux-amd64 serve > server.log 2>&1 &
echo $! > quick-feishu.pid
```

使用保存的 PID 停止后台进程：

```bash
kill "$(cat quick-feishu.pid)"
rm quick-feishu.pid
```

首次启动会在程序同目录生成：

- `config.yaml`：配置文件
- `data/`：SQLite 数据目录

打开 `http://服务器地址:8080`，进入「设置」，填写账号 ID、系统令牌、API 地址和飞书 Webhook，然后：

1. 点击「测试飞书连接」。
2. 设置每日快照和日报时间。
3. 点击「保存并应用」，程序会切换到对应账号数据库并采集一次快照。

定时任务当前按北京时间（UTC+8）执行。

### 直接编辑配置文件

也可在程序同目录创建 `config.yaml`：

```yaml
app:
  port: 8080
  timezone: Asia/Shanghai

account:
  user_id: "你的账号 ID"
  system_token: "你的系统令牌"
  api_base: "https://api.quickrouter.ai"

feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
  retry_times: 3

schedule:
  snapshot_time: "00:00"
  report_time: "10:30"
```

敏感项可改用环境变量，环境变量优先级高于配置文件，且对应字段会在页面中锁定：

| 环境变量 | 对应配置 |
| --- | --- |
| `QR_USER_ID` | `account.user_id` |
| `QR_SYSTEM_TOKEN` | `account.system_token` |
| `QR_FEISHU_WEBHOOK` | `feishu.webhook_url` |
| `QF_CONFIG_FILE` | 配置文件路径 |
| `QF_DATA_DIR` | SQLite 数据目录 |

## 3. 日常使用

Web 页面包含：

- **仪表盘**：查看最新用量、立即采集快照、立即发送日报、查看发送记录。
- **历史快照**：查询采集历史和令牌明细。
- **日报模板**：调整卡片标题、分区、字段及差值显示。
- **设置**：修改账号、Webhook、采集时间和日报时间。

命令行也支持手动执行：

```bash
# 启动 Web 服务和定时任务
./quick-feishu serve

# 立即采集并保存一次快照
./quick-feishu snapshot

# 使用已有的最新快照生成并发送日报
./quick-feishu run-report
```

首次发送日报前必须至少成功采集一次快照。切换 `user_id` 后会使用独立数据库：`data/accounts/<user_id>/quick-feishu.db`。

## 4. 打包生成 dist 产物

环境要求：Go 1.26.4、Node.js 和 npm。

```bash
# 在项目根目录执行

# 首次打包先安装前端依赖
cd frontend
npm ci
cd ..

# 打包
./scripts/build.sh
```

脚本会先执行前端构建，将页面写入 Go 的嵌入目录，再交叉编译所有平台。产物位于 `dist/`：

```text
dist/
├── quick-feishu-darwin-arm64
├── quick-feishu-darwin-amd64
├── quick-feishu-linux-amd64
└── quick-feishu-windows-amd64.exe
```

这些文件都已内置前端页面，部署时不需要复制 `frontend/`，也不需要安装 Node.js。将对应平台的一个二进制文件与可写的配置、数据目录一起部署即可。

## 5. Linux 部署

建议将程序、配置和数据分开存放：

```text
/opt/quick-feishu/quick-feishu
/etc/quick-feishu/config.yaml
/var/lib/quick-feishu/
```

创建 `/etc/systemd/system/quick-feishu.service`：

```ini
[Unit]
Description=QuickFeishu
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=quick-feishu
Group=quick-feishu
ExecStart=/opt/quick-feishu/quick-feishu serve
Environment=QF_CONFIG_FILE=/etc/quick-feishu/config.yaml
Environment=QF_DATA_DIR=/var/lib/quick-feishu
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

确保运行用户可读配置文件、可写数据目录，然后启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now quick-feishu
sudo systemctl status quick-feishu
journalctl -u quick-feishu -f
```

停止或重启服务：

```bash
sudo systemctl stop quick-feishu
sudo systemctl restart quick-feishu
```

升级前备份 `config.yaml` 和整个数据目录。服务没有登录鉴权且监听所有网卡，不要直接暴露到公网；建议仅允许内网访问，或在前置反向代理中增加 HTTPS 和身份认证。

## 6. 常见问题

- **页面打不开**：查看启动日志中的实际端口，并检查防火墙；部署时最好确保配置端口未被占用。
- **采集失败**：检查 `user_id`、`system_token`、`api_base` 以及服务器到 QuickRouter API 的网络连接。
- **飞书发送失败**：在设置页测试连接，并确认机器人 Webhook 有效、群策略允许机器人发消息。
- **修改端口未生效**：端口变更需要重启整个进程；「保存并应用」只重启内部定时任务。
- **环境变量配置无法在页面修改**：这是预期行为；需修改服务环境变量并重启进程。
