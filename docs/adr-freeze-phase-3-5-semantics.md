# ADR — Freeze Phase 3–5 Semantics (Streaming / Usage / Quota)

## Status

Accepted (Phase 3–5 baseline semantics)

## Context

Phase 3–5 在系统内建立了 Streaming Governance、Usage 观测与 Quota 的最小闭环。

在当前阶段，我们的目标不是“更精确/更公平/更商业化”，而是让系统：

- 能稳定工作
- 能解释自己的行为
- 能提供可观测证据（UsageRecord / internal APIs）

我们明确接受一些“看起来不直觉”的中间态行为（例如 non-streaming quota 像按次数、streaming quota 按 token 中断），并将更复杂、更精细的模型推迟到 Phase 6。

这份文档的作用：

- 防止未来（3 个月后或新同事）把 Phase 6 的复杂度倒灌回 Phase 3–5
- 给出统一的对外解释口径（Admin / Internal API）
- 给出后续演进的边界与路线

---

## Decision

冻结 Phase 3–5 的语义版本，作为后续迭代的 baseline：

- Streaming Runtime 不承担业务含义，只负责生命周期驱动（OnFirstChunk / OnChunk / OnEnd）
- 治理逻辑通过 Observer 组合实现（QuotaObserver / MeteringObserver / streamWriteObserver 等）
- 对外解释以 UsageRecord 为权威证据
- Quota 统一解释为“token 级预算”，即使 Admission 阶段估算极粗

---

## What Phase 3–5 Solved

### 1) Streaming Governance: 可插拔的生命周期与治理

- **Streaming 生命周期事件**：

  - `OnFirstChunk`：标记流已开始（first_chunk_at），并允许设置 SSE headers
  - `OnChunk`：逐 chunk 治理（可中断）
  - `OnEnd`：统一收口与对账

- **ObserverGroup 组合治理**：
  - 不修改 Runtime 即可叠加治理能力
  - Quota / Metering / SSE 输出等均可通过 observer 注入

### 2) Streaming SSE 输出：由 Observer 写回 ResponseWriter

- `streamWriteObserver` 在 `OnFirstChunk/OnChunk/OnEnd` 写入：
  - SSE headers
  - `data: ...\n\n` chunk
  - `data: [DONE]\n\n`

关键语义：

- **首 chunk 前失败**：仍可返回 JSON error（协议对齐）
- **首 chunk 后失败/中断**：以 streaming end 语义收口（DONE + UsageRecord）

### 3) UsageRecord：对外可解释的权威证据

- 在 streaming 结束时生成 UsageRecord（通过 MeteringObserver）
- 关键字段（Phase 3.5 对齐）：
  - request_id / api_key_id / provider / model
  - status（completed / partial）
  - end_reason（stop / quota_exceeded / ...）
  - confirmed_tokens / chunk_count
  - start_at / first_chunk_at / end_at / duration
  - error_msg（如果有）

### 4) Internal Usage Query：最小可查询闭环

- `GET /internal/usage?api_key_id=...&limit=...`
- `GET /internal/usage/:request_id`

目的：

- 用于解释“发生了什么”
- 用于验证 streaming 治理语义是否可观测

### 5) Quota Snapshot：预算状态的最小可观测闭环

- `GET /internal/quota/:api_key`
- 返回：
  - quota_total（阶段性定义为 used + remaining）
  - quota_used（UsageRecord 汇总）
  - quota_remaining（QuotaStore）
  - last_updated_at

目的：

- 验证 QuotaStore + UsageRecord 在解释口径上自洽
- 让系统从“只能解释单次请求”升级到“能解释一段时间的预算状态”

---

## Allowed Intermediate Behaviors (Intentional)

这些行为在 Phase 3–5 **刻意允许**，不是 bug：

### A) non-streaming quota 看起来像“调用次数”

原因：Admission 阶段 `estimatedTokens` 可能缺失/为 0，会退化为最小扣减（例如 1）。

对外口径：

- remaining 仍然解释为 token 预算
- “像次数”只是估算粒度太粗导致的观感

### B) streaming quota 在生成过程中按 token 消耗并可中断

原因：Streaming Governance 阶段逐 chunk 扣减（confirmed/estimated tokens），用于防 runaway generation。

对外口径：

- 系统允许开始生成，但不保证完整生成
- 预算不足时可中断并生成 UsageRecord（status=partial, end_reason=quota_exceeded）

### C) QuotaStore 单 bucket 被多阶段复用

原因：当前阶段未显式建模 quota 单位与多桶。

对外口径：

- remaining = token 级预算余额
- Admission 与 Governance 都在消耗同一份预算，只是粒度不同

### D) Internal APIs 无权限/无分页（仅最小闭环）

- `/internal/usage`、`/internal/quota` 仅用于语义验证与 admin 观测
- 安全、鉴权、分页、多条件检索属于后续阶段

### E) 存储为 InMemory（演示/开发态）

- recorder/query/quotaStore 都是 InMemory 实现
- 这是为了冻结语义与加速迭代，不是生产化方案

---

## Explicitly Deferred to Phase 6

以下问题被明确推迟到 Phase 6（不要倒灌回 Phase 3–5）：

### 1) Quota 单位显式建模与多桶预算

- request quota vs token quota
- 多维度 quota（按 model/provider/tenant/项目）
- 时间窗口与重置策略

### 2) Billing / money / pricing

- 不引入货币化、计费、价格表

### 3) 更精确的 tokenization / 估算一致性

- 统一 estimated vs confirmed 的口径
- 更可靠的 token 计算器

### 4) Durable storage / retention

- UsageRecord 持久化（DB/OLAP）
- 查询性能、分页、过滤、索引

### 5) 安全与权限

- internal/admin API 的鉴权与审计
- 多租户隔离

### 6) 产品化体验

- 用户侧更直觉的 remaining 解释与提示
- 更细粒度的错误分类与 SLA 指标

---

## Consequences

### Positive

- 当前系统语义可解释、可观测、可回归测试
- 后续 Phase 6 的复杂度有明确边界，不会污染现阶段实现

### Trade-offs

- 估算粗糙导致用户观感可能“不直觉”
- Internal API 暂时仅适用于开发/验证语义，不适合直接暴露
