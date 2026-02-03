# Story 3.6: 规则说明页面

Status: ready-for-dev

## Story

As a 普通用户,
I want 查看推广规则详情,
so that 了解奖励机制和提现规则。

## Acceptance Criteria

1. **AC1**: 显示邀请奖励规则（双方各得金额）
2. **AC2**: 显示阶梯佣金比例表（档位、人数范围、比例）
3. **AC3**: 显示分成门槛条件
4. **AC4**: 显示提现规则（门槛、周期、方式）
5. **AC5**: 显示常见问题 FAQ（可折叠 Accordion）

## Dependencies

- **Depends On**: Story 3.1 (推广中心路由)
- **Depended By**: 无（静态页面，无后端依赖）

## Tasks / Subtasks

- [ ] Task 1: 创建规则页面 (AC: #1-#5)
  - [ ] 1.1 创建 `src/views/user/affiliate/AffiliateRulesView.vue`
  - [ ] 1.2 规则内容硬编码（后续可从配置获取）
  - [ ] 1.3 FAQ 折叠组件
  - [ ] 1.4 注册路由 `/user/affiliate/rules`

## Dev Notes

### 页面结构

```
AffiliateRulesView.vue
├── 邀请奖励说明
├── 阶梯佣金比例表
├── 分成门槛说明
├── 提现规则说明
└── FAQ (Accordion)
```

规则内容初期硬编码在前端，Epic 7 实现后从 API 获取配置值。

### 组件框架

```vue
<!-- src/views/user/affiliate/AffiliateRulesView.vue -->
<script setup lang="ts">
// 静态页面，无需 composable
// 规则数据硬编码，Epic 7 后改为 API 获取

const faqItems = ref([
  { q: '什么是有效邀请？', a: '被邀请人完成首次充值即为有效邀请', open: false },
  { q: '佣金多久到账？', a: '佣金创建后有 7 天确认期，确认后可提现', open: false },
  { q: '提现门槛是多少？', a: '累计可提现金额达到 $100 即可申请提现', open: false },
])

function toggleFaq(index: number) {
  faqItems.value[index].open = !faqItems.value[index].open
}
</script>
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_all_sections | 挂载组件 | 渲染邀请奖励、阶梯表、门槛、提现规则、FAQ 5个区块 |
| faq_toggle | 点击 FAQ 问题 | 展开/收起答案 |
| tier_table_render | 挂载组件 | 显示阶梯比例表（3档） |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.6] - FR46
- [Source: _bmad-output/affiliate/UX-affiliate-system.md] - 规则页面设计

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
