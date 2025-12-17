# Allowance / Usage / Quota（对外解释口径）

结论：Snapshot ≠ Allowance ≠ Usage。

## 1. 什么是 Allowance（使用许可）

Allowance 来自 API Key 的配置，表示“你被允许怎么用”。

它包括：

- 可用模型（AllowedModels）：这个 key 被允许访问哪些模型
- 最大预算（QuotaLimit）：这个 key 被授予的最大使用预算（许可上限）
- 使用速率（RateLimit）：这个 key 被授予的速率许可（可能尚未被强制执行）

Allowance 是语义模型，不等价于计费（Billing）。

---

## 2. 什么是 Usage（已发生使用）

Usage 来自 UsageRecord，表示“你实际上发生了什么”。

它是事后事实：

- 用于解释一次请求最终发生了什么
- 不参与准入（Admission）或实时治理（Enforcement）决策

---

## 3. 什么是 Quota / Remaining（当前状态）

Quota / Remaining 来自 QuotaStore，表示“此刻还能不能继续用”。

它是 runtime 治理状态：

- 用于事前准入（Admission）：请求是否允许进入
- 用于事中治理（Streaming Governance）：生成过程中是否允许继续

---

## 4. 一个请求的完整解释路径（文字版）

- “请求是否允许进入” → 看 Quota / Policy
- “请求过程中是否被中断” → 看 Streaming Governance
- “最终用了多少” → 看 UsageRecord
- “这个 Key 被允许多少” → 看 Allowance
