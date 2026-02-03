# Story 1.3: 生成分享海报

Status: ready-for-dev

## Story

As a 普通用户,
I want 生成包含二维码的分享海报,
so that 在社交平台上更有吸引力地推广。

## Acceptance Criteria

1. **AC1**: 提供 3 套海报模板（简约风、活力风、节日风）
2. **AC2**: 海报包含用户二维码和利益点文案
3. **AC3**: 海报图片支持保存到相册/下载
4. **AC4**: 二维码动态嵌入用户专属邀请链接

## Dependencies

- **Depends On**: Story 1.2 (二维码生成, qrcode_data_url), Story 3.1 (推广中心页面入口)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 海报生成组件 (AC: #1, #2, #3, #4)
  - [ ] 1.1 创建 `src/components/user/affiliate/PosterGenerator.vue`
  - [ ] 1.2 3 套模板 JSON 配置（背景、文字位置、样式）
  - [ ] 1.3 使用 Canvas API 或 html2canvas 生成图片
  - [ ] 1.4 动态嵌入二维码（从 affiliate info API 获取）
- [ ] Task 2: 海报页面入口 (AC: #1)
  - [ ] 2.1 在推广中心首页添加「生成海报」按钮
  - [ ] 2.2 创建 `src/views/user/affiliate/InviteToolsView.vue`（集成海报、二维码、链接）

## Dev Notes

### 技术方案

- 使用 `html2canvas` 将 DOM 渲染为图片
- 3 套模板：每个模板一个 Vue 组件 slot 或 JSON 配置
- 二维码使用 Story 1.2 返回的 `qrcode_data_url`
- 下载：`<a download>` + canvas.toDataURL()

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface PosterTemplate {
  id: string
  name: string
  bg: string
  textColor: string
  accent: string
}
```

### 组件框架

```vue
<!-- src/components/user/affiliate/PosterGenerator.vue -->
<script setup lang="ts">
import type { PosterTemplate } from '@/types/affiliate'
import html2canvas from 'html2canvas'

const props = defineProps<{
  referralLink: string
  qrcodeDataUrl: string
}>()

const templates: PosterTemplate[] = [
  { id: 'minimal', name: '简约风', bg: '#fff', textColor: '#333', accent: '#d97757' },
  { id: 'vibrant', name: '活力风', bg: 'gradient', textColor: '#fff', accent: '#ff6b35' },
  { id: 'festive', name: '节日风', bg: 'pattern', textColor: '#fff', accent: '#e74c3c' },
]

const selectedTemplate = ref<PosterTemplate>(templates[0])
const posterRef = ref<HTMLElement>()

async function downloadPoster() {
  if (!posterRef.value) return
  const canvas = await html2canvas(posterRef.value)
  const link = document.createElement('a')
  link.download = 'invite-poster.png'
  link.href = canvas.toDataURL()
  link.click()
}
</script>
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_templates | 挂载组件 | 显示 3 套模板选项 |
| template_switch | 点击"活力风" | 切换模板样式 |
| qrcode_embedded | 传入 qrcodeDataUrl | 海报中显示二维码图片 |
| download_trigger | 点击下载按钮 | 调用 html2canvas 生成图片 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-1.3] - FR5
- [Source: _bmad-output/affiliate/UX-affiliate-system.md] - 海报模板设计

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
