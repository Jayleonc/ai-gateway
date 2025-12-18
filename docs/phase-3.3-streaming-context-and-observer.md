# Phase 3.3 — StreamingContext / StreamingObserver 语义说明（对齐代码）

本文只解释三件事（对外/对内都可复用）：

- `StreamingContext` 是什么、承载什么事实
- `StreamObserver` 是什么、为什么要存在
- 订阅者（observer）分别负责什么（quota / metering / logging / tracing）

---

## 1) StreamingContext（per-request 状态）

`StreamingContext` 表示一次 streaming 请求的 **per-request 运行态事实容器**。

它承载的不是“策略/决策”，而是“事实/状态”。这一点在代码注释中也被明确：

- Context 只做事实容器
- 决策交给订阅者（observer）

### 它承载哪些信息（事实字段）

`StreamingContext` 承载以下关键事实（用于解释、治理、观测）：

- 身份与归属：

  - `request_id`（`RequestID`）
  - `api_key_id`（`APIKeyID`）
  - `provider`（`Provider`）
  - `model`（`Model`）

- 生命周期时间点：

  - `start_at`（`StartAt`）
  - `first_chunk_at`（`FirstChunkAt`）
  - `end_at`（`EndAt`）

- 计量/交付规模：

  - `chunks`（`ChunkCount`）
  - `confirmed_tokens`（`ConfirmedTokens`）

- 结束状态：
  - `end_reason`（`EndReason`，例如 `stop` / `quota_exceeded` / `error` / `client_disconnect` / `internal_error`）
  - `err`（`Err`）

### “首 chunk”为什么重要

`StreamingContext.IsStarted()` 以 `FirstChunkAt != nil` 作为判断：

- 首 chunk 前：允许返回结构化 JSON error（协议对齐）
- 首 chunk 后：响应体已开始写入，错误只能以“流式结束语义”收口（例如 DONE 或断流），并依赖日志/UsageRecord 解释

### 现状对齐（代码事实）

- 当前代码中已存在 `StreamingContext`
- 它在 runtime 与 observer 之间传递 per-request 状态
- `Runtime` 负责：
  - 标记 `FirstChunkAt`
  - 增加 `ChunkCount`
  - 设置 `EndReason`
  - 保证 `OnEnd` 只执行一次（通过 `Ended` / `MarkEnd()`）

---

## 2) StreamingObserver（OnFirstChunk / OnChunk / OnEnd）

`StreamObserver` 是 streaming 生命周期的观察者接口，用于把“治理/观测/输出”从 runtime 编排中解耦。

### 为什么要有 Observer

`Runtime` 的职责是过程编排：

- 读流
- 驱动生命周期（first chunk / chunk / end）
- 保证 `OnEnd` 一定被调用

但 `Runtime` 不应该承担：

- quota 怎么算
- usage 怎么记
- 日志怎么打
- tracing 怎么注入
- SSE 怎么写

Observer 的存在就是为了把这些“扩展性行为”外置为订阅者。

### 生命周期钩子语义

- `OnFirstChunk(ctx)`

  - 在首个 chunk **已成功写出并 flush** 后调用
  - 语义上代表：请求已经“开始交付”

- `OnChunk(ctx, delta)`

  - 每个 chunk 交付后调用
  - 允许 observer 返回 `stop=true` 来触发中断（治理点）

- `OnEnd(ctx)`
  - 流结束时调用（无论正常/异常）
  - runtime 保证它被调用（且只调用一次）

### 现状对齐（代码事实）

- 当前代码中已存在 `StreamObserver` 接口
- 已存在 `ObserverGroup` 用于组合多个 observer：
  - 顺序调用 `OnFirstChunk`
  - 顺序调用 `OnChunk`，任意 observer `stop=true` 即停止
  - `OnEnd` 会尽量调用全部 observer，即使某个出错

---

## 3) 订阅者（observer）清单与定位（示例，不固定）

下面列的是“典型订阅者分类”，用于解释系统分层与职责边界。具体有哪些订阅者，以当前实现为准。

### quota（治理类）

定位：Streaming Governance（事中治理）。

- 作用：在生成过程中根据预算/配额状态决定是否中断
- 典型结果：将 `end_reason` 归因为 `quota_exceeded`

现状对齐：当前代码中已存在 quota observer（基于 `QuotaStore`）。

### metering（观测/对账类）

定位：Usage / Accounting（事后事实）。

- 作用：在 `OnEnd` 生成 `UsageRecord`
- 用途：用于解释“最终发生了什么”，不参与实时决策

现状对齐：当前代码中已存在 metering observer，并在 `OnEnd` 写入 recorder。

### logging（稳定日志口径）

定位：可观测性与排障。

- 作用：输出稳定的 streaming summary（key=value），用于检索与聚合
- 对齐：日志字段口径见 `phase-3.5-external-alignment-observability.md` 的 3.5.3

现状对齐：当前代码中 `Runtime.LogSummary()` 会输出摘要日志（无论成功或失败）。

### tracing（追踪/链路观测）

定位：可观测性扩展点。

- 作用：注入/传播 trace id、span 等（例如请求维度的 trace）
- 本阶段原则：只定义口径，不定义实现

现状对齐：当前文档阶段只声明 tracing 的语义位置与边界，不要求具体实现。
