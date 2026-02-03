# Story 2.4: 佣金计算失败重试

Status: ready-for-dev

## Story

As a 系统,
I want 佣金计算失败时记录并重试,
so that 保证佣金不丢失。

## Acceptance Criteria

1. **AC1**: 佣金计算异常时记录到 `commission_retry` 表
2. **AC2**: 不影响充值主流程
3. **AC3**: 定时任务每 10 分钟重试
4. **AC4**: 最多重试 3 次，超过标记 failed
5. **AC5**: 重试队列堆积 > 100 条时记录告警日志

## Dependencies

- **Depends On**: Story 2.3 (佣金计算失败时写入 commission_retry 表)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 创建 commission_retry 表 + Schema (AC: #1)
  - [ ] 1.1 创建 Ent Schema 和迁移
- [ ] Task 2: 重试记录逻辑 (AC: #1, #2)
  - [ ] 2.1 在佣金计算失败时调用 `SaveRetryRecord(order, err)`
- [ ] Task 3: 重试定时任务 (AC: #3, #4, #5)
  - [ ] 3.1 创建 `commission_retry_scheduler.go`
  - [ ] 3.2 每 10 分钟查询 pending 且 retry_count < 3 的记录
  - [ ] 3.3 重试成功标记 success，失败 retry_count++
  - [ ] 3.4 超过 3 次标记 failed

## Dev Notes

### commission_retry 表

```sql
CREATE TABLE commission_retry (
    id          BIGSERIAL PRIMARY KEY,
    order_id    VARCHAR(64) NOT NULL,
    user_id     BIGINT NOT NULL,
    amount      DECIMAL(10,2) NOT NULL,
    retry_count INT DEFAULT 0,
    last_error  TEXT,
    status      VARCHAR(20) DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_retry_status ON commission_retry(status, retry_count);
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestSaveRetryRecord | 保存失败记录 | 正确写入 order_id、user_id、amount、last_error |
| TestRetryJob_Success | 重试成功 | status 更新为 success |
| TestRetryJob_Fail | 重试仍失败 | retry_count+1，last_error 更新 |
| TestRetryJob_MaxRetries | retry_count=3 | status 标记为 failed，不再重试 |
| TestRetryJob_AlertThreshold | 队列堆积 > 100 条 | 记录告警日志 |
| TestRetryJob_EmptyQueue | 无待重试记录 | 正常返回，不报错 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整重试流程 | 写入失败记录 → 定时任务拉取 → 重试成功 → 状态更新 |
| 最大重试后标记失败 | 连续失败3次 → status=failed → 不再被拉取 |

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.7] - 失败重试流程
- [Source: _bmad-output/planning-artifacts/epics.md#Story-2.4] - NFR16, NFR17

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
