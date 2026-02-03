# Story 3.3: 一键复制邀请链接

Status: ready-for-dev

## Story

As a 普通用户,
I want 一键复制我的邀请链接,
so that 快速分享给朋友。

## Acceptance Criteria

1. **AC1**: 点击「复制链接」按钮后邀请链接复制到剪贴板
2. **AC2**: 显示 Toast 提示「复制成功」
3. **AC3**: Toast 2 秒后自动消失
4. **AC4**: 浏览器不支持 Clipboard API 时降级处理

## Dependencies

- **Depends On**: Story 3.1 (AffiliateHomeView 页面和推广信息 API)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 复制功能实现 (AC: #1, #2, #3, #4)
  - [ ] 1.1 在推广中心首页添加「复制链接」按钮
  - [ ] 1.2 使用 Clipboard API (`navigator.clipboard.writeText`)
  - [ ] 1.3 降级方案：`document.execCommand('copy')`
  - [ ] 1.4 Toast 提示组件

## Dev Notes

### 实现代码

```typescript
async function copyInviteLink() {
  const link = affiliateInfo.value?.referral_link
  if (!link) return

  try {
    await navigator.clipboard.writeText(link)
    showToast('复制成功')
  } catch {
    // 降级方案
    const textarea = document.createElement('textarea')
    textarea.value = link
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    showToast('复制成功')
  }
}
```

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.3] - FR40

### composable 用法

```typescript
// src/composables/useClipboard.ts
export function useClipboard() {
  const copied = ref(false)

  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text)
      copied.value = true
    } catch {
      // 降级方案
      const textarea = document.createElement('textarea')
      textarea.value = text
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      copied.value = true
    }
    setTimeout(() => { copied.value = false }, 2000)
  }

  return { copied, copy }
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| copy_success | 点击复制按钮 | 调用 clipboard.writeText，显示 Toast |
| toast_auto_dismiss | 复制成功后 | Toast 2秒后消失 |
| fallback_copy | Clipboard API 不可用 | 使用 execCommand 降级方案 |
| no_link | affiliateInfo.referral_link 为空 | 按钮不执行操作 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
