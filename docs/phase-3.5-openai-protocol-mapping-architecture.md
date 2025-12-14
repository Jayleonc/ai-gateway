# Phase 3.5 — OpenAI 协议映射层（Protocol Mapping）架构约定（事实文档）

本文档把“对外协议映射层（Protocol/Transport Mapping）作为唯一权威来源”的设计写成团队约定。

本文档只描述架构边界与事实来源，不描述 UI / 管理 API / DB。

## 背景问题（为什么要做）

在 Phase 3.5 之前，对外语义映射的逻辑分散在三处：

- policy：deny code（治理决策结果）
- core/errors：内部 error → OpenAI JSON（code/type/http status）
- handler：streaming EndReason + started → 返回 OpenAI error 或断流

这会导致：同一个对外语义（例如 `quota_exceeded`）在多个层级被重复表达，未来新增错误类型或新增 endpoint 时容易漂移。

## 目标（单一事实来源）

- 对外协议行为（HTTP status / OpenAI error.type / error.code / message 口径 / streaming 的首 chunk 前后差异）必须由 **唯一权威来源** 产出。
- handler 不写映射规则，只执行“计划（Plan）”。

## 三层责任边界（必须遵守）

### 1) Domain / Governance（内部事实与决策）

- policy：只负责决策（allow/deny）以及 deny 原因（内部治理码）。
- streaming：只负责记录事实（EndReason、IsStarted、已确认 token 等）。

这一层不产生 OpenAI JSON，不产生 http status。

### 2) Error Taxonomy（内部错误分类）

- core/errors：只负责定义内部错误分类（sentinel errors / wrapped errors），并提供统一的错误类型常量。
- core/errors 不应散落在业务点里做协议映射规则。

### 3) Protocol Mapping（对外协议映射）

- `internal/transport/openai` 是 **唯一权威来源**：
  - 将 domain facts / internal errors 映射为 OpenAI-compatible 响应计划（Plan）
  - 统一处理 streaming 的“首 chunk 前可返回 JSON error、首 chunk 后只能断流/终止”的约束

## 代码落地（最小版本）

### 核心产物：Plan

- `ErrorPlan`：描述一个 OpenAI error JSON 的输出（HTTPStatus + Body）。
- `StreamEndPlan`：描述 streaming 结束时应采取的动作：
  - `write_json_error`（仅 started=false 时允许）
  - `close_stream`（started=true 时的最小策略）

### handler 的约束

- handler 只做：
  - 识别 non-streaming/streaming
  - 对 streaming 读取 started 状态
  - 执行 Plan

handler 不保存对外 error code/message 表。

## 演进规则（未来新增错误/原因时必须做）

- 新增 EndReason / 新增内部错误分类时：
  - 必须在 `internal/transport/openai` 增补映射
  - 必须补对应表驱动测试（未来补齐）

## 非目标

- 不引入管理 API / UI。
- 不要求全量覆盖所有 HTTP 错误码。
- 不在本阶段暴露 UsageRecord 给用户。
