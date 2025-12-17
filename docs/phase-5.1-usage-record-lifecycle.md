# Phase 5.1 — UsageRecord 与 Streaming 生命周期绑定（Explainability 基础）

目标：不是把治理做精确，而是让系统能解释清楚自己现在在做什么。

Phase 5.1 的交付是：将 streaming 生命周期（start / first chunk / end）沉淀为可查询记录，使系统具备“事后解释源”。

---

## 做了什么

- 引入 `UsageRecord` 作为统一的“事后解释记录”
- 将 streaming 生命周期关键时刻写入 UsageRecord：
  - `start_at`
  - `first_chunk_at`
  - `end_at`
  - `duration`
- 记录归属信息与调用信息：
  - `request_id`
  - `api_key_id`
  - `provider` / `model`
- 记录可观测的交付规模：
  - `confirmed_tokens`
  - `chunk_count`
- 记录最终结果：
  - `status`（completed / partial）
  - `end_reason`（stop / quota_exceeded / ...）
  - `error_msg`（如有）

---

## 语义定位

UsageRecord 的定位是：**Accounting / Usage（事后解释）**。

- 它回答的问题是：
  - “这次请求最终发生了什么？”
- 它不回答的问题是：
  - “该不该继续生成？”（实时治理问题）

因此：

- UsageRecord 是对外解释的权威来源
- UsageRecord 不参与实时决策

---

## 关键语义与不变量

### 1) first_chunk_at 的语义

- `first_chunk_at != nil` 表示：
  - 请求已经进入了“开始交付内容”的阶段
  - 客户端很可能已进入 streaming 消费路径

它是区分以下两类错误的关键证据：

- 首 chunk 前失败：可能返回 JSON error
- 首 chunk 后失败：必须以 SSE 结束语义收口

### 2) status 与 end_reason 的组合解释

- `status=completed` + `end_reason=stop`

  - 表示正常完成

- `status=partial` + `end_reason=quota_exceeded`
  - 表示“已交付部分内容，但因预算限制中断”

> 说明：Phase 5 的目标是“可解释”，不是“精确计量”。因此 confirmed_tokens 的估算可能很粗，但语义上仍作为统一口径。

---

## 可验证点（回归测试关心什么）

- streaming 正常结束时，会写入 UsageRecord：

  - status=completed
  - end_reason=stop

- streaming 事中被中断时，会写入 UsageRecord：

  - status=partial
  - end_reason=quota_exceeded（或其他）

- UsageRecord 字段应包含足够证据用于解释：
  - 请求是否开始交付（first_chunk_at）
  - 交付规模（chunk_count / confirmed_tokens）
  - 结束原因（end_reason）
