-- Migration 078: 为 usage_logs 表添加请求阶段耗时字段
--
-- 背景：当前 usage_logs 只存储 duration_ms（端到端总耗时）和 first_token_ms（首字节时间）。
-- 本迁移增加 4 个阶段耗时字段，用于精确定位请求各环节的性能瓶颈：
--
--   auth_latency_ms      : 认证鉴权阶段耗时（从请求进入到鉴权完成）
--   routing_latency_ms   : 路由选择阶段耗时（账号选择 + 并发槽位等待）
--   upstream_latency_ms  : 上游请求阶段耗时（从发出请求到收到首字节/首块响应）
--   response_latency_ms  : 响应处理阶段耗时（流式传输或同步读取响应体的时间）
--
-- 所有字段均为 NULL（历史数据无值），新请求写入时填充。
-- 字段类型为 integer（毫秒），与 duration_ms / first_token_ms 保持一致。

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS auth_latency_ms     integer,
  ADD COLUMN IF NOT EXISTS routing_latency_ms  integer,
  ADD COLUMN IF NOT EXISTS upstream_latency_ms integer,
  ADD COLUMN IF NOT EXISTS response_latency_ms integer;

COMMENT ON COLUMN usage_logs.auth_latency_ms     IS '认证鉴权阶段耗时（ms），从请求进入到鉴权完成';
COMMENT ON COLUMN usage_logs.routing_latency_ms  IS '路由选择阶段耗时（ms），账号选择 + 并发槽位等待';
COMMENT ON COLUMN usage_logs.upstream_latency_ms IS '上游请求阶段耗时（ms），从发出请求到收到首字节';
COMMENT ON COLUMN usage_logs.response_latency_ms IS '响应处理阶段耗时（ms），流式传输或读取响应体';
