# Phase 3.2 — Streaming 生命周期与事件边界设计

本文档把 Phase 3.1 的“治理语义”（成功/确认消耗/结算/不计费失败/旁路）压缩为 Gateway 内部**一条可实现、可观测**的 Streaming 生命周期，并明确“事件边界点”（事件 ≠ MQ）。

## 目标

- 定义 Streaming 请求在 Gateway 内的标准生命周期（线性模型）。
- 定义每个阶段的**可观测边界点**（Hook/Checkpoint/State transition）。
- 明确：
  - 哪些点允许挂治理逻辑（quota/metering/log/tracing）
  - 哪些点永远不应做业务判断

## 核心澄清：事件 ≠ 消息队列

Phase 3.2 的“事件”指：

- 在同一个请求上下文内
- 在同一个 handler 生命周期内
- Gateway 能稳定感知、并对外解释的**状态跃迁点**

它可以被实现为：

- Hook 点
- Callback
- Observer
- 内存中的状态机

而不是：

- 跨服务异步消息
- 必达投递
- 分布式消费

> 是否引入 MQ 是未来实现选择；是否存在事件边界点是系统可治理性的生死线。

## 标准生命周期（Gateway 视角）

```
┌──────────────┐
│ RequestStart │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ PolicyCheck  │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ ProviderConnect  │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ FirstChunkWrite  │  ★ Streaming 成功起点（Phase 3.1）
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ ChunkDelivered * │  (0..N 次)
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ StreamEnd        │  ★ 结算点
└──────────────────┘
```

> 这是 Gateway 的视角，不是 Provider 的视角。

## 事件边界点定义（允许做什么 / 禁止做什么）

### 1) RequestStart

**发生时机**

- 请求进入 handler，完成基础解析/鉴权前准备（例如 request id、trace id）。

**允许**

- 初始化 StreamingContext（token=0/chunk=0/started=false）。
- 绑定身份信息（APIKeyID / ProjectID / TenantID）。
- 初始化观测（trace span、日志字段）。

**禁止**

- 扣 quota
- 写 metering
- 做任何 provider 相关判断

**原因**

- RequestStart ≠ 用户消耗开始。

### 2) PolicyCheck

**发生时机**

- 事前静态治理：key 校验、项目状态、模型/路由规则。

**允许**

- 作为唯一硬门禁：deny/allow。
- 决定目标 provider/model。

**禁止**

- token 级别动态决策
- streaming 过程中才知道的信息反推到这里

**原因**

- Policy = 静态治理
- Streaming = 动态过程

### 3) ProviderConnect

**发生时机**

- 已向 provider 发起请求，且成功拿到可读 stream（response body reader）。

**允许**

- 打点：connect latency。
- 记录“尝试调用 provider”的事件（用于可观测性/审计）。

**禁止**

- 扣 quota
- 写 metering

**原因**

- ProviderConnect ≠ 成功起点（用户尚未收到任何可感知交付）。

### 4) FirstChunkWrite（Phase 3.2 核心锚点）

**发生时机（必须精确定义）**

- Gateway 向客户端：
  - 写出至少一个有效 chunk
  - 并 flush 成功

**它意味着**

- StreamingStarted = true
- Phase 3.1 的“成功起点”成立
- 后续失败不能整体回滚

**允许**

- 标记 streaming started。
- 开始确认 token 消耗（进入“消耗态”）。
- 允许 quota 从“允许”进入“扣减态”（但不建议在此一次性扣完）。

**禁止**

- 以 ProviderConnect 作为成功起点。
- 在没有交付的情况下写入 metering 作为“已发生消耗”。

### 5) ChunkDelivered（0..N 次）

**发生时机**

- 每次一个 chunk 成功写出并 flush 后（或等价的“交付确认点”）。

**允许**

- 估算 token 增量，并累加 confirmedTokens。
- 尝试消耗 quota（TryConsume / 或按 chunk 聚合扣减）。
- 写入 metering buffer（内存态），等待结算。

**关键原则：可失败但不可回滚**

- 如果在第 N 个 chunk 后 quota 耗尽：
  - 可以选择中断流（立即/或发完当前 chunk 再中断）
- 但不能：
  - 回滚 N-1 个 chunk 已确认的消耗
  - 改口说“这次不算数”

**与 Phase 3.1 对齐**

- 平台只为“成功交付给客户端的内容”负责计量，而不为“尝试生成的内容”负责。

### 6) StreamEnd（结算点）

**发生时机**

- 流结束：
  - provider 正常结束
  - gateway 主动中断
  - client disconnect
  - provider 中途异常

**允许**

- 写最终 metering（confirmedTokens）。
- 记录结束原因（finish_reason / error_reason）。
- 关闭 trace span，输出最终日志。

**禁止**

- 把 StreamEnd 当成“治理起点”。

**原因**

- StreamEnd 是结算点，不是开始点。

## 特殊情况：Client Disconnect / Provider 异常

### Client Disconnect

- 不需要独立事件类型，它是 StreamEnd 的一种原因。
- 处理口径：
  - 若 `StreamingStarted=false`（FirstChunkWrite 前断开）：视为未成功，不计费/不扣配额。
  - 若 `StreamingStarted=true`：进入 StreamEnd，已确认部分计费/扣配额。

### Provider 中途异常

- FirstChunk 前异常：ProviderConnect → StreamEnd（失败，不计费）。
- FirstChunk 后异常：ChunkDelivered → StreamEnd（部分成功，已确认部分计费）。

## Phase 3.2 的一句工程口径

Streaming 治理只关心三类边界点：

1. 是否开始交付（FirstChunkWrite）
2. 已交付多少（ChunkDelivered）
3. 何时结束（StreamEnd）

其余都是实现细节。

## 下一步（Phase 3.3 预告）

- 设计 `StreamingContext`（per-request 状态）
- 设计 `StreamingObserver`（OnFirstChunk/OnChunk/OnEnd）
- 订阅者包含：quota、metering、logging、tracing

> 仍然全部在内存、同步路径内完成，不引入 MQ。
