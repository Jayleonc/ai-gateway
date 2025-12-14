# Phase 3.4 — Metering 语义设计（非实现）

本文档只定义 **UsageRecord 的语义** 与 **结算约束**，不讨论存储介质（内存/DB/异步队列）与实现细节。

## 真实目标

把 Streaming 过程中的“已确认消耗”（例如 `StreamingContext.ConfirmedTokens`）提升为**可结算事实**（UsageRecord），使其满足：

- 一次请求最终形成明确的结算记录
- 有明确的幂等键（防重复入账）
- 有明确的写入时机（最终一致）
- 有明确的失败旁路策略（不影响响应）

## 1) 一次 Streaming 请求，最终要生成几条 UsageRecord？

**答案：1 条（最终结算记录）。**

- UsageRecord 表示“一次 request 的最终结算结果”，不是过程事件流。
- StreamingContext 在过程中累积“已确认事实”（如 confirmed_tokens），到 OnEnd 形成最终结算。

> 后续若要支持“长连接按时间片落账/中途落账防丢”，可以演进为多条（periodic settlement），但不属于本阶段语义。

## 2) 唯一键是什么？

**答案：幂等键使用 `request_id`（全局唯一）。**

- `request_id`：作为 UsageRecord 的 idempotency key。
- `api_key_id`：作为计费归属维度（subject），用于聚合/查询/对账，但**不参与幂等键**。

推荐语义字段（非实现）：

- `usage_record.idempotency_key = request_id`
- `usage_record.api_key_id = api_key_id`

> 若不信任外部传入的 request_id，应明确规定：request_id 必须由网关生成并保证全局唯一。

## 3) OnEnd 时写，还是过程中 buffer？

**答案：过程中 buffer（累积事实），OnEnd 一次性写入（提交结算）。**

- 过程中：不断更新 StreamingContext 的事实字段（confirmed_tokens/chunk_count/first_chunk_at 等）。
- OnEnd：将“事实账本”快照转为 UsageRecord 并提交。

理由：

- Runtime 已保证 OnEnd only-once，是天然的“结算提交点”。
- 避免把 3.4 拉进事件流语义（乱序、重放、增量幂等、聚合）。

## 4) Metering 失败时，是否影响响应？

**答案：不影响响应（旁路 best-effort）。**

- Metering 属于事后记录：失败应降级为日志/指标/告警。
- 失败不应让请求失败，也不应回滚已发生的对外输出。
- 依赖幂等键（request_id）保证后续重试/补偿不会重复入账。

## (可选) UsageRecord 的 status 语义

UsageRecord 可以包含一个 `status` 字段，用于表达“结算完成度”。这是语义层字段，不是数据库设计约束。

- `status=completed`：流正常结束（完整结算）。
- `status=partial`：已发生首 chunk，但因 quota/provider/client 等原因中途终止（部分结算）。
- `status=failed`：首 chunk 前失败（通常不会生成 UsageRecord；若为可观测性需要生成，则应明确标记为 failed）。

> 本文档不讨论如何实现。

## 结论（四句话版本）

- **条数**：每个 streaming 请求 **1 条最终 UsageRecord**。
- **幂等键**：`request_id`。
- **写入时机**：过程中累积事实，**OnEnd 提交结算**。
- **失败策略**：**旁路不影响响应**，失败可观测、可重试且幂等。
