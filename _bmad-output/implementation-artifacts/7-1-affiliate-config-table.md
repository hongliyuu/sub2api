# Story 7.1: 分销配置数据表

Status: ready-for-dev

## Story

As a 系统,
I want 将分销配置存储到数据库,
so that 支持动态调整。

## Acceptance Criteria

1. **AC1**: `affiliate_config` 表存在且包含默认配置项
2. **AC2**: 配置项使用 JSONB 存储值，支持多种数据类型
3. **AC3**: 默认配置：register_bonus、commission_threshold、tier_rules、withdraw_threshold、commission_confirm_days
4. **AC4**: 初始化脚本/迁移插入默认值

## Dependencies

- **Depends On**: 无（基础设施，首个配置故事）
- **Depended By**: Story 7.2 (管理页面读写配置), Story 7.3 (缓存热更新)

## Tasks / Subtasks

- [ ] Task 1: 创建 affiliate_config Ent Schema + 迁移 (AC: #1, #2)
  - [ ] 1.1 创建 `backend/ent/schema/affiliate_config.go`
  - [ ] 1.2 SQL 迁移文件 + 默认数据 INSERT
- [ ] Task 2: Repository (AC: #3, #4)
  - [ ] 2.1 创建 `internal/repository/affiliate_config_repo.go`
  - [ ] 2.2 实现 Get/Set/GetAll 方法
- [ ] Task 3: ConfigService (AC: #3)
  - [ ] 3.1 创建 `internal/service/affiliate_config_service.go`
  - [ ] 3.2 带 Redis 缓存的配置读取

## Dev Notes

### 表结构

```sql
CREATE TABLE affiliate_config (
    id          SERIAL PRIMARY KEY,
    config_key  VARCHAR(64) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    tier_type   VARCHAR(20) DEFAULT 'global',
    description VARCHAR(256),
    updated_by  BIGINT,
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 默认配置
INSERT INTO affiliate_config (config_key, config_value, description) VALUES
('register_bonus', '{"inviter": 1.00, "invitee": 1.00}', '注册奖励金额'),
('commission_threshold', '{"type": "or", "monthly_plan": true, "min_spend": 10.00}', '分成门槛'),
('tier_rules', '[{"level":1,"min":0,"max":10,"rate":0.05},{"level":2,"min":11,"max":30,"rate":0.08},{"level":3,"min":31,"max":null,"rate":0.12}]', '阶梯规则'),
('max_commission_rate', '0.15', '佣金比例上限'),
('withdraw_threshold', '100.00', '提现门槛'),
('commission_confirm_days', '7', '佣金确认天数');
```

### Ent Schema 框架

```go
// backend/ent/schema/affiliate_config.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type AffiliateConfig struct {
    ent.Schema
}

func (AffiliateConfig) Fields() []ent.Field {
    return []ent.Field{
        field.String("config_key").Unique(),
        field.JSON("config_value", map[string]interface{}{}),
        field.String("tier_type").Default("global"),
        field.String("description").Optional(),
        field.Int64("updated_by").Optional().Nillable(),
        field.Time("updated_at").Default(time.Now),
    }
}

func (AffiliateConfig) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("config_key").Unique(),
    }
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestConfigGet_CacheHit | Redis 缓存存在 | 直接返回缓存数据 |
| TestConfigGet_CacheMiss | Redis 缓存不存在 | 查 DB 并回填缓存，TTL=5min |
| TestConfigGet_NotFound | 数据库无此 key | 返回 error |
| TestConfigSet_Normal | 写入新配置值 | DB 更新 + Redis 缓存删除 |
| TestConfigGetAll | 获取所有配置 | 返回完整配置列表 |
| TestConfigDefaultValues | 初始化后 | 包含 6 个默认配置项 |

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.5] - affiliate_config 表
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#5.1] - 缓存策略

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
