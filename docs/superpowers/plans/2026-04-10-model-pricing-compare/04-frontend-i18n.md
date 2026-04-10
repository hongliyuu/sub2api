# Task 4 — 前端 i18n

> 在 zh.ts 和 en.ts 中新增三层价格对比相关的 i18n key。

**文件：**
- 修改：`frontend/src/i18n/locales/zh.ts`（第 3109 行附近 `modelPricings` 对象）
- 修改：`frontend/src/i18n/locales/en.ts`（第 3009 行附近 `modelPricings` 对象）

---

## 步骤

- [ ] **Step 1: 在 zh.ts 的 modelPricings 对象中添加新 key**

找到 `zh.ts` 第 3139 行 `calculationLogic: '计费逻辑说明',` 这一行，在其后（`usdPerMillion` 之后、`},` 之前）插入：

```typescript
      // 三层对比
      litellmPrice: 'LiteLLM 价格',
      dbPrice: '数据库配置',
      fallbackPrice: '内置 Fallback',
      activeSource: '生效层',
      activeSourceLitellm: 'LiteLLM（最高优先）',
      activeSourceDatabase: '数据库配置',
      activeSourceFallback: '内置 Fallback',
      activeSourceNone: '无价格数据',
      noLitellmData: '无 LiteLLM 数据',
      litellmCompare: 'LiteLLM 对比',
      priceDiffHigher: '你的配置高于 LiteLLM',
      priceDiffLower: '你的配置低于 LiteLLM',
      priceDiffEqual: '与 LiteLLM 一致',
      priceDiffNoData: 'LiteLLM 无数据',
      // 查询区域
      lookupTitle: '查询任意模型价格',
      lookupPlaceholder: '输入模型名，如 gpt-4o、claude-sonnet-4',
      lookupButton: '查询',
      lookupLoading: '查询中...',
      lookupNoResult: '未找到该模型的任何价格数据',
      lookupResultTitle: '价格对比',
      tierLabel: '来源层',
      inputOutput: '输入 / 输出（/百万 Token）',
      cacheReadCreation: '缓存读取 / 创建',
      priorityPrice: 'Priority 服务层',
      activeTag: '生效中',
```

- [ ] **Step 2: 在 en.ts 的 modelPricings 对象中添加新 key**

找到 `en.ts` 第 3040 行 `usdPerMillion: 'USD / 1M Tokens',` 之后（`},` 之前）插入：

```typescript
      // Three-tier compare
      litellmPrice: 'LiteLLM Price',
      dbPrice: 'DB Config',
      fallbackPrice: 'Built-in Fallback',
      activeSource: 'Active Source',
      activeSourceLitellm: 'LiteLLM (highest priority)',
      activeSourceDatabase: 'DB Config',
      activeSourceFallback: 'Built-in Fallback',
      activeSourceNone: 'No pricing data',
      noLitellmData: 'No LiteLLM data',
      litellmCompare: 'LiteLLM Compare',
      priceDiffHigher: 'Your config is higher than LiteLLM',
      priceDiffLower: 'Your config is lower than LiteLLM',
      priceDiffEqual: 'Matches LiteLLM',
      priceDiffNoData: 'No LiteLLM data',
      // Lookup area
      lookupTitle: 'Query Any Model Price',
      lookupPlaceholder: 'Model name, e.g. gpt-4o, claude-sonnet-4',
      lookupButton: 'Query',
      lookupLoading: 'Querying...',
      lookupNoResult: 'No pricing data found for this model',
      lookupResultTitle: 'Price Comparison',
      tierLabel: 'Source',
      inputOutput: 'Input / Output (per 1M tokens)',
      cacheReadCreation: 'Cache Read / Creation',
      priorityPrice: 'Priority Tier',
      activeTag: 'Active',
```

- [ ] **Step 3: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit
```

预期：无类型错误。

- [ ] **Step 4: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "Feature: i18n 新增三层价格对比相关文案"
```
