# Phase 5.5 — Quota View / Budget Snapshot（最小可观测闭环）

一句话：让用户“看见预算”，而不是“用好预算”。

Phase 5.5 的价值：

- 验证：QuotaStore + UsageRecord 在对外解释上是自洽的
- 为 Phase 6 的 quota model 升级提供数据基础
- 让系统从“只能解释单次请求” → “能解释一段时间的预算状态”

---

## 做了什么

提供一个 internal / admin API：

- `GET /internal/quota/:api_key`

返回一个预算快照：

- `quota_total`
- `quota_used`（来自 UsageRecord 汇总）
- `quota_remaining`（来自 QuotaStore）
- `last_updated_at`

---

## 字段语义（Phase 5.5 版本）

### quota_remaining

- 来源：QuotaStore.Remaining(api_key)
- 语义：当前阶段统一解释为“token 级预算余额”

### quota_used

- 来源：UsageRecord 汇总（confirmed_tokens 累加）
- 语义：在可观测窗口内（取决于当前存储实现）已交付内容的规模

### quota_total

Phase 5.5 的阶段性定义：

- `quota_total = quota_used + quota_remaining`

它是一个“解释性总量”，用于让对外口径自洽：

- remaining（实时） + used（事后记录） → 一个可解释的预算快照

### last_updated_at

- 来源：UsageRecord 中最近一次的 CompletedAt（否则退回 RequestedAt）
- 语义：快照数据最后一次发生变化的时间点

---

## 边界（本阶段刻意不做）

- 不支持充值
- 不支持时间窗口
- 不支持多维度 quota
- 不支持 money

也就是说：Phase 5.5 不是产品化预算系统，它只是一个“可观测闭环”。

---

## 可验证点

- quota_used 与 /internal/usage 的 confirmed_tokens 汇总一致
- quota_remaining 与 QuotaStore 一致
- quota_total 在 Phase 5.5 语义下自洽（used + remaining）
