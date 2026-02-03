# Story 5.3: 提现方式选择

Status: ready-for-dev

## Story

As a 普通用户,
I want 选择微信或支付宝作为提现方式,
so that 将佣金转入我常用的账户。

## Acceptance Criteria

1. **AC1**: 提现页面可选择「微信」或「支付宝」
2. **AC2**: 根据方式填写对应账户信息
3. **AC3**: 账户信息脱敏显示

## Dependencies

- **Depends On**: Story 5.2 (提现 API 接口), Story 5.1 (提现资格检查)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 前端提现页面 (AC: #1, #2, #3)
  - [ ] 1.1 创建 `src/views/user/affiliate/WithdrawView.vue`
  - [ ] 1.2 方式选择 Radio 组件
  - [ ] 1.3 账户输入表单
  - [ ] 1.4 金额输入和校验

## Dev Notes

纯前端页面，调用 Story 5.2 的 `POST /api/v1/affiliate/withdraw` 接口。

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface WithdrawRequest {
  amount: number
  payment_method: 'wechat' | 'alipay'
  payment_account: string
}
```

### 组件框架

```vue
<!-- src/views/user/affiliate/WithdrawView.vue -->
<script setup lang="ts">
import type { WithdrawRequest } from '@/types/affiliate'
import { ref, computed } from 'vue'
import { submitWithdraw } from '@/api/affiliate'

const form = ref<WithdrawRequest>({
  amount: 0,
  payment_method: 'alipay',
  payment_account: '',
})

const submitting = ref(false)

// 表单校验
const isValid = computed(() =>
  form.value.amount >= 100 &&
  form.value.payment_account.length > 0
)

async function handleSubmit() {
  if (!isValid.value || submitting.value) return
  submitting.value = true
  try {
    await submitWithdraw(form.value)
    // 成功提示并跳转
  } finally {
    submitting.value = false
  }
}
</script>
```

### API 客户端

```typescript
// src/api/affiliate.ts
export function submitWithdraw(data: WithdrawRequest) {
  return request.post('/api/v1/affiliate/withdraw', data)
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| method_switch | 点击"微信" | payment_method 切换为 wechat |
| account_input | 输入账户 | 正确绑定 payment_account |
| amount_validation | 输入金额 50 | 提交按钮禁用 |
| submit_success | 表单合法并提交 | 调用 API 并显示成功提示 |
| submit_loading | 提交中 | 按钮禁用，显示 loading |
| account_mask | 显示已保存账户 | 脱敏显示 138****8888 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.3] - FR28

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
