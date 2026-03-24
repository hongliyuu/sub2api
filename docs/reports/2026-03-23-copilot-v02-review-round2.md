# Copilot v0.2 文档二次评审报告

- 评审对象:
  - `docs/copilot-improvements/v0.2-implementation-plan.md`
  - `docs/reports/2026-03-23-copilot-plan-v01-v02-review-response.md`
- 评审日期: 2026-03-23
- 评审结论: v0.2 相比 v0.1 明显改进，H1/H2/M1/M3 处理方向正确；仍有 1 个中风险一致性问题和 1 个低风险交付完整性问题需要修订。

---

## 主要发现（按严重级别）

### 中风险

#### M1. M2 处置结论与方案细节存在冲突，容易导致执行偏差

- 证据 1（v0.2 摘要）:
  - `v0.2-implementation-plan.md` 写明“无缓存时保留原有 5xx”。
  - 位置: `docs/copilot-improvements/v0.2-implementation-plan.md:10`
- 证据 2（v0.2 详细策略）:
  - 表格明确写“缓存过期 + 上游失败 + 无任何缓存 → 200，静态默认列表”。
  - 位置: `docs/copilot-improvements/v0.2-implementation-plan.md:208`
- 证据 3（回应报告总结）:
  - 汇总行写“无缓存时保留 5xx”，与上面策略不一致。
  - 位置: `docs/reports/2026-03-23-copilot-plan-v01-v02-review-response.md:123`

- 影响:
  - 实施人员会不确定“无缓存 + 上游失败”到底该返回 200 还是 5xx，最终行为可能与预期不一致。

- 建议修订:
  - 二选一并全篇统一：
    1. 如果目标是高可用优先：保留“无缓存 + 上游失败 → 200 默认模型”，则把摘要/回应中的“无缓存时保留 5xx”改成“仅无账号且无缓存返回 503”。
    2. 如果目标是语义优先：改成“无缓存 + 上游失败 → 502”，仅 stale 场景返回 200。

### 低风险

#### L1. P3a 交付物清单包含数据文件，但当前仓库尚未提供该文件

- 文档声明:
  - P3a 交付包含 `docs/copilot-improvements/supported-endpoints-data.md`。
  - 位置: `docs/copilot-improvements/v0.2-implementation-plan.md:458`
- 当前状态:
  - 文件尚不存在（本次检查）。

- 影响:
  - 不影响方案本身，但会影响“P3a 是否已可执行/可验收”的判断。

- 建议修订:
  - 在 v0.2 文档中明确该文件是“后续实施产物”，或者先落一个模板文件（含表头与采集时间字段）以消除歧义。

---

## 已确认修正到位的项

- H1: 已从“签名冒泡方案”切换为 `ForwardMessages` 独立扫描（方案 B），方向正确。
- H2: 已把“图片在 merge 中丢失”拆成独立子任务 P1-A，方向正确。
- M1: 缓存已改为按 `groupID` 设计，方向正确。
- M3: 已明确“先采集真实 SupportedEndpoints，再写入 DefaultModels”，方向正确。

---

## 审阅结论

v0.2 已达到“可实施草案”质量。建议先修正文档中的 M2 语义冲突（中风险）后再进入编码阶段，避免实现和验收标准不一致。
