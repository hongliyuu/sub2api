# Oracle Operations

本文件记录当前 Oracle 生产机 `/home/ubuntu/sub2api` 的最低限度运维约定，目标是先把“能恢复、能自检、减少漂移”补上，而不去冒然改动线上流量路径。

## 当前新增的文件

- `deploy/Caddyfile.oracle-a1-free`
  - 仓库内保存的 Oracle 当前生效 Caddy 配置快照，来源于 `/etc/caddy/Caddyfile`
- `deploy/ops/preflight_oracle.sh`
  - 发布前/巡检前自检脚本
- `deploy/ops/backup_sub2api.sh`
  - 在线备份脚本，不停止服务
- `deploy/ops/verify_backup_sub2api.sh`
  - 备份完整性检查脚本
- `deploy/systemd/sub2api-backup.service`
- `deploy/systemd/sub2api-backup.timer`

## 推荐命令

```bash
cd /home/ubuntu/sub2api
bash deploy/ops/preflight_oracle.sh
bash deploy/ops/backup_sub2api.sh
bash deploy/ops/verify_backup_sub2api.sh latest
sudo install -m 0644 deploy/systemd/sub2api-backup.service /etc/systemd/system/sub2api-backup.service
sudo install -m 0644 deploy/systemd/sub2api-backup.timer /etc/systemd/system/sub2api-backup.timer
sudo systemctl daemon-reload
sudo systemctl enable --now sub2api-backup.timer
```

## 备份内容

每次备份会写入 `/home/ubuntu/backups/sub2api/<UTC timestamp>/`，至少包含：

- `postgres.dump`
- `sub2api_data.tgz`
- `redis_data.tgz`
- `.env`
- `Caddyfile.oracle-live`
- `manifest.txt`
- `SHA256SUMS`（如果系统可用）

## 运维约束

- 不要直接把生产机仓库当开发机长期改代码；先在本地/分支完成，再部署 commit。
- 每次改 `/etc/caddy/Caddyfile` 后，同步回 `deploy/Caddyfile.oracle-a1-free`，避免继续漂移。
- `sub2api-backup.timer` 只负责产出备份，不自动删除旧备份。清理备份前先人工确认。
- 发布前至少运行一次 `deploy/ops/preflight_oracle.sh`。

## 当前未做的事

- 没有引入新依赖。
- 没有修改当前线上 Caddy 路由。
- 没有自动化执行破坏性恢复；恢复仍建议人工确认后进行。
