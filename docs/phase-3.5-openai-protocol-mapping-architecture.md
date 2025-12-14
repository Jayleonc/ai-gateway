# Error / Governance Code 归属与对外协议映射设计决策（定稿）

（Phase 3.5 — OpenAI 协议映射层 / Protocol Mapping）

本文档用于回答并固定一类长期决策问题：

> 类似 `quota_exceeded` 这种“业务错误 code / 治理语义”，
> 到底应该定义在哪一层？谁是唯一权威？如何避免未来漂移？

该文档不是代码说明，而是架构级决策记录（事实文档），用于未来新增功能（新 provider / 新 endpoint / 新治理能力 / 新对外协议）时直接复用判断。

## 一、问题背景（为何必须做出该决策）

在 AI Gateway / Control Plane 这种系统中，存在三类“看起来像 error code 的东西”：

1. 治理决策结果（例如：配额不足、限流）
2. 内部错误归因（例如：provider 断流、客户端断开、内部异常）
3. 对外协议字段（HTTP status / OpenAI error.type / error.code / message）

如果不明确分层，这三者会在代码中混用字符串，导致：

- code 定义重复
- handler 写 if-else
- streaming / non-streaming 行为分叉
- 新增错误类型时无人知道该改哪里

结论：必须明确“谁负责表达什么”，并建立单一事实来源。

## 二、总体设计原则（核心结论）

### 核心原则一句话版

内部“发生了什么” 与 对外“怎么表现”，必须分离。

对应到工程结构：

| 层级               | 职责                     | 是否知道 OpenAI |
| ------------------ | ------------------------ | --------------- |
| policy / streaming | 记录事实、给出内部原因   | 否              |
| core/errors        | 内部错误分类（taxonomy） | 否              |
| transport/openai   | 对外协议映射（唯一权威） | 是              |

## 三、关于“业务 code（如 quota_exceeded）”的归属

### 决策结论

业务 code 不应该在多个层“各自定义一份字符串”。应当拆分为不同语义层级的“强类型原因”，由协议映射层统一输出字符串。

### 推荐做法

#### 1) Policy 层：治理原因（DenyReason）

- 含义：为什么被拒 / 不允许继续
- 面向：治理逻辑
- 示例：
  - `policy.DenyReasonQuotaExceeded`
  - `policy.DenyReasonRateLimited`

#### 2) Streaming 层：结束归因（EndReason）

- 含义：流为什么结束
- 面向：事实账本 / 计量 / 可观测性
- 示例：
  - `streaming.EndReasonQuotaExceeded`
  - `streaming.EndReasonProviderError`
  - `streaming.EndReasonClientDisconnect`
  - `streaming.EndReasonStop`

#### 3) 对外协议层：OpenAI Error Code

- 含义：协议字段
- 面向：客户端兼容性
- 示例：
  - `"quota_exceeded"`
  - `"rate_limited"`
  - `"provider_error"`

三者可以使用相同字符串值，但必须是不同类型、不同包、不同职责。

### 如何避免重复 / 漂移？

- policy / streaming 不输出 string
- 只输出枚举（typed reason）
- 只有 `internal/transport/openai` 层负责把枚举 → 字符串 code
- 新增枚举必须补 mapping，否则测试失败

## 四、对外 OpenAI 兼容错误的“唯一权威来源”

### 决策结论

HTTP status / error.type / error.code / message 的唯一权威来源：`internal/transport/openai`。

### Handler 的职责边界

Handler 只关心：

1. 是否 streaming
2. streaming 是否已发生 first chunk

Handler 不关心：

- http status
- OpenAI error.type
- error.code
- message 文案
- streaming 下是否该断流还是写终止 chunk

这些都属于协议映射层的责任。

### 推荐抽象：Plan 模型

协议映射层返回一个“执行计划（Plan）”，handler 只执行，不做判断。

- `ErrorPlan`（非 streaming）
- `StreamEndPlan`（streaming）

Plan 至少包含：

- 是否写 JSON error
- HTTP status
- OpenAI error payload
- 是否终止流 / 断流

## 五、Streaming 的 EndReason 与对外 error code 的关系

### 决策结论

必须分开。

原因：

- EndReason 是内部归因
- OpenAI error code 是协议字段

它们经常一一对应，但不是同一个概念，未来一定会分叉。

### 推荐命名与映射策略

- EndReason：

  - `quota_exceeded`
  - `provider_error`
  - `internal_error`
  - `client_disconnect`
  - `stop` / `length`

- 对外协议：
  - OpenAI error.code（字符串）
  - OpenAI error.type（rate_limit_error / server_error / ...）

映射发生在：

```
(EndReason + streamingStarted + err)
→ StreamEndPlan
```

而不是在 handler / runtime / observer 中。

## 六、推荐的目录结构与包边界（可演进）

```text
internal/
  policy/
    ...                 # 只输出治理决策与内部原因（typed）
  gateway/
    handler/
      ...               # 不做错误映射，只执行 Plan
    streaming/
      ...               # EndReason / Runtime / Observer（只记录事实）
  core/
    errors/
      ...               # 内部错误分类（taxonomy），不含 OpenAI JSON
  transport/
    openai/
      mapper.go         # 唯一权威：内部 → OpenAI 行为
      codes.go          # OpenAI error.code / error.type
      plans.go          # ResponsePlan / StreamPlan
      mapper_test.go    # 表驱动测试，覆盖所有映射
  provider/
    ...                 # provider 适配器，返回内部错误分类
```

## 七、Streaming 约束下的对外行为映射（可测试）

### 关键约束（来自 Phase 3.5）

- 首 chunk 前：可以返回 JSON error
- 首 chunk 后：HTTP 200 已写，不保证能返回结构化 error

### 推荐做法

- 在 `transport/openai` 中定义显式映射表
- 映射维度：
  - 是否 streaming
  - 是否 started（first chunk）
  - 内部原因（DenyReason / EndReason / ErrorCategory）

### 测试策略（强制）

- 表驱动测试覆盖所有组合：
  - started = true / false
  - quota / provider / internal / client_disconnect / stop
- 新增原因必须补映射，否则测试失败

## 八、最小重构执行清单（不改架构即可落地）

### Step 1

引入 `internal/transport/openai` 包（空壳即可）。

### Step 2

把 `WriteOpenAIError` 改为执行 `ErrorPlan`。

### Step 3

把 streaming 的 EndReason → OpenAI error 映射逻辑移动到 mapper。

### Step 4

逐步把 policy / streaming 输出从 string 改为 enum。

### Step 5

补齐 Phase 3.5 的表驱动映射测试。

## 九、当前代码落地点（本仓库现状）

为确保“事实文档”可落地且可验证，当前仓库已完成最小版本落地：

- 协议映射层：

  - `internal/transport/openai/plans.go`
  - `internal/transport/openai/mapper.go`（`MapError` / `MapStreamEnd`）

- handler 执行 Plan：
  - `internal/gateway/handler/error.go`（`WriteOpenAIError` 调用 `transport/openai.MapError`）
  - `internal/gateway/handler/chat.go`（streaming 结束调用 `transport/openai.MapStreamEnd`）

## 十、最终总结（一句话）

policy / streaming 只负责“发生了什么”，core/errors 只负责“是什么错误”，transport/openai 才负责“对外怎么表现”。

当未来出现任何“这个 code 应该放哪”的问题，优先回到这份文档，而不是回到具体实现。
