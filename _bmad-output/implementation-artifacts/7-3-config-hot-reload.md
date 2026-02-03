# Story 7.3: 配置热更新

Status: ready-for-dev

## Story

As a 系统,
I want 配置修改后立即生效,
so that 无需重启服务。

## Acceptance Criteria

1. **AC1**: 管理员修改配置保存后清除 Redis 缓存
2. **AC2**: 下次业务读取时从数据库加载最新值
3. **AC3**: 缓存 TTL 5 分钟作为兜底

## Dependencies

- **Depends On**: Story 7.1 (配置缓存策略), Story 7.2 (配置更新 API)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 缓存失效逻辑 (AC: #1, #2, #3)
  - [ ] 1.1 在 `affiliate_config_service.go` 实现 `InvalidateCache(key)` 方法
  - [ ] 1.2 在配置更新 API 中调用缓存失效
  - [ ] 1.3 业务逻辑从 ConfigService 读取配置（带缓存）

## Dev Notes

```go
func (s *AffiliateConfigService) InvalidateCache(ctx context.Context, key string) {
    s.redis.Del(ctx, fmt.Sprintf("aff:config:%s", key))
}

func (s *AffiliateConfigService) Get(ctx context.Context, key string) (interface{}, error) {
    cacheKey := fmt.Sprintf("aff:config:%s", key)
    // 先查缓存
    cached, err := s.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var val interface{}
        json.Unmarshal([]byte(cached), &val)
        return val, nil
    }
    // 查数据库
    config, err := s.configRepo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }
    // 回填缓存
    data, _ := json.Marshal(config.ConfigValue)
    s.redis.Set(ctx, cacheKey, string(data), 5*time.Minute)
    return config.ConfigValue, nil
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestInvalidateCache | 删除指定 key 缓存 | Redis DEL 调用成功 |
| TestGet_AfterInvalidate | 缓存失效后读取 | 从 DB 加载并重建缓存 |
| TestGet_CacheTTL | 缓存自然过期 | 5分钟后从 DB 重新加载 |
| TestGet_CacheCorrupt | 缓存 JSON 损坏 | 降级查 DB |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 配置更新后热生效 | 修改配置 → 缓存失效 → 业务读取新值 |

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#5.3] - 缓存失效策略

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
