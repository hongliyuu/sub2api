# Story 7.2: 后台配置管理页面

Status: ready-for-dev

## Story

As a 管理员,
I want 在后台配置分销参数,
so that 灵活调整运营策略。

## Acceptance Criteria

1. **AC1**: 管理员配置页面显示所有配置项当前值
2. **AC2**: 可编辑：注册奖励、分成门槛、阶梯档位、提现门槛、确认周期
3. **AC3**: 保存后配置存入数据库并清除缓存
4. **AC4**: 显示保存成功提示

## Dependencies

- **Depends On**: Story 7.1 (affiliate_config 表和 ConfigService)
- **Depended By**: Story 7.3 (配置保存后触发缓存失效)

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3)
  - [ ] 1.1 创建 `internal/handler/admin/affiliate_config_handler.go`
  - [ ] 1.2 `GET /api/admin/affiliate/config` - 获取所有配置
  - [ ] 1.3 `PUT /api/admin/affiliate/config/:key` - 更新配置
  - [ ] 1.4 注册 admin 路由
- [ ] Task 2: 前端管理页面 (AC: #1, #2, #4)
  - [ ] 2.1 创建 `src/views/admin/affiliate/AffiliateConfigView.vue`
  - [ ] 2.2 配置编辑表单

## Dev Notes

### API

```
GET /api/admin/affiliate/config
PUT /api/admin/affiliate/config/:key
Body: { "config_value": {...} }
```

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface AffiliateConfig {
  config_key: string
  config_value: unknown
  tier_type: string
  description: string
  updated_at: string
}

export interface AffiliateConfigListResponse {
  list: AffiliateConfig[]
}
```

### 组件框架

```vue
<!-- src/views/admin/affiliate/AffiliateConfigView.vue -->
<script setup lang="ts">
import type { AffiliateConfig } from '@/types/affiliate'
import { ref, onMounted } from 'vue'
import { getAffiliateConfigs, updateAffiliateConfig } from '@/api/admin/affiliate'

const configs = ref<AffiliateConfig[]>([])
const saving = ref(false)

onMounted(async () => {
  const res = await getAffiliateConfigs()
  configs.value = res.data.list
})

async function handleSave(key: string, value: unknown) {
  saving.value = true
  try {
    await updateAffiliateConfig(key, { config_value: value })
    // 成功提示
  } finally {
    saving.value = false
  }
}
</script>
```

### API 客户端

```typescript
// src/api/admin/affiliate.ts
export function getAffiliateConfigs() {
  return request.get<AffiliateConfigListResponse>('/api/admin/affiliate/config')
}

export function updateAffiliateConfig(key: string, data: { config_value: unknown }) {
  return request.put(`/api/admin/affiliate/config/${key}`, data)
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_all_configs | 加载配置列表 | 显示所有配置项 |
| edit_config_value | 修改注册奖励值 | 正确更新表单 |
| save_success | 点击保存 | 调用 API 并显示成功提示 |
| save_loading | 保存中 | 按钮禁用 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetAllConfigs | 获取全部配置 | 返回所有配置项 |
| TestUpdateConfig_Normal | 更新配置值 | DB 更新 + 缓存清除 |
| TestUpdateConfig_NotFound | key 不存在 | 返回 404 |

- [Source: _bmad-output/planning-artifacts/epics.md#Story-7.2] - FR47-52
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.2.1] - Admin API

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
