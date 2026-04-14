# 编译

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o okex-buddy main.go 
```

<br />

# 创建服务

```bash

sudo vim /etc/systemd/system/okex-buddy.service

[Unit]
Description=OKEx Buddy Trading Daemon
After=network.target redis.service  # 确保网络和Redis先启动（如果你用了Redis）
Wants=redis.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/home/go-okex-buddy
ExecStart=/home/go-okex-buddy/okex-buddy
Restart=always
RestartSec=5s
# 标准日志输出（告别 nohup.out）
StandardOutput=journal
StandardError=journal

# 可选：内存与资源限制（防止内存泄漏拖垮服务器）
# MemoryMax=512M
# OOMScoreAdjust=-500

[Install]
WantedBy=multi-user.target
```

# 启动与管理服务

```bash
#重载配置（每次修改 .service 文件后都要执行）
systemctl daemon-reload
#启动服务
systemctl start okex-buddy
#设置开机自启
systemctl enable okex-buddy
#检查状态
systemctl status okex-buddy
```

<br />

# 查看实时日志

```bash
#（类似 tail -f）
journalctl -u okex-buddy -f
#查看最近 100 行日志
sudo journalctl -u okex-buddy -n 100
#查看指定时间段的日志
sudo journalctl -u okex-buddy --since "2026-04-14 09:00:00"
```

<br />

# 修改 journalctl 配置

```bash
vim /etc/systemd/journald.conf

[Journal]
# 最大单个日志文件大小（默认 10% 分区空间，这里设为 100M）
SystemMaxFileSize=100M
# 日志目录总大小上限（建议 1G-2G）
SystemMaxUse=1G
# 日志保留时长（默认不清理，这里设为保留 7 天）
MaxRetentionSec=7day
# 自动压缩过期的日志文件（节省空间）
Compress=yes

systemctl restart systemd-journald
```


