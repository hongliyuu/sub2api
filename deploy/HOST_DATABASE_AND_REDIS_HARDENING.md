# Host PostgreSQL And Redis Hardening

This guide is for the `docker-compose.standalone.yml` deployment mode where
PostgreSQL and Redis run on the host instead of inside Docker.

## PostgreSQL

Recommended `conf.d` fragment:

```conf
listen_addresses = '127.0.0.1,172.17.0.1'
max_connections = 150
shared_buffers = '512MB'
effective_cache_size = '1536MB'
maintenance_work_mem = '128MB'
work_mem = '8MB'
shared_preload_libraries = 'pg_stat_statements'
pg_stat_statements.max = 10000
pg_stat_statements.track = all
pg_stat_statements.save = on
compute_query_id = on
track_io_timing = on
log_min_duration_statement = '1000ms'
log_lock_waits = on
deadlock_timeout = '1s'
log_line_prefix = '%m [%p] db=%d,user=%u,app=%a,client=%h '
```

After restart, run:

```sql
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

## Redis

Recommended bind scope:

```conf
bind 127.0.0.1 172.17.0.1 -::1
protected-mode yes
```

Keep `requirepass` enabled and avoid binding Redis to the public NIC.
