# Phase 3.5 — 语义对外对齐 & 可观测性（草案）

Phase 3.5 的目标一句话版：让“平台内部已经发生的治理事实”，对外变得可解释、可对齐、可验证。

本文档只定义 **对外行为**（HTTP / OpenAI 语义 / Streaming 结束行为 / 可观测性口径），不写实现。

## 范围与术语

- **EndReason**：网关内部对“流为何结束”的归因（见 `StreamingContext.EndReason`）。
- **OpenAI Error**：对齐 OpenAI 兼容 API 的错误响应结构：`{"error": {"message": ..., "type": ..., "code": ...}}`。
- **Streaming 结束行为**：
  - **正常结束**：输出最终的结束 chunk（OpenAI chat.completion.chunk，`finish_reason=stop|length|...`），然后发送 `[DONE]`。
  - **异常结束**：可能返回一个 error 响应（仅在首 chunk 前），或在首 chunk 后断流（无法可靠返回标准 JSON 错误）。

## 3.5.1 —— EndReason → OpenAI Error / Response 对齐

### 核心原则

- **首 chunk 前（StreamingStarted=false）**：允许返回标准 HTTP 错误响应（JSON），完全对齐 OpenAI error 结构。
- **首 chunk 后（StreamingStarted=true）**：HTTP status 已经是 200 且响应体开始写入，此时：
  - 不保证还能返回结构化 JSON 错误。
  - 优先策略是：
    - 能写“终止 chunk”就写终止 chunk（并尽可能给出可解释的 `finish_reason` / 内嵌错误提示）；
    - 否则直接断流，由客户端感知为连接中断。

> 该原则与 Phase 3.1 的“成功起点 = first chunk flush”一致：first chunk 后的失败属于“部分成功”。

### EndReason 映射表（对外口径）

下表定义了“**推荐默认口径**”。其中 OpenAI `error.type` 以 OpenAI 习惯命名为准（语义对齐优先，字段名不做扩展）。

| EndReason           | 场景解释（对外）                                   | StreamingStarted=false（首 chunk 前）                                                                                                              | StreamingStarted=true（首 chunk 后）                                                                    |
| ------------------- | -------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `quota_exceeded`    | 预算/配额不足（平台侧治理）                        | HTTP **429**；OpenAI error：`type=rate_limit_error`；`code=quota_exceeded`                                                                         | **中断流**：推荐写一个“终止 chunk”后 `[DONE]`（或直接断流）；并保证日志/UsageRecord 可解释为 quota 中断 |
| `client_disconnect` | 客户端主动断开连接                                 | 不返回（连接已断）                                                                                                                                 | 不返回（连接已断）；在日志/UsageRecord 中归因                                                           |
| `internal_error`    | 网关内部错误（非上游返回）                         | HTTP **500**；OpenAI error：`type=server_error`；`code=internal_error`                                                                             | **中断流**：优先断流；若能写“终止 chunk”则写（但不保证）                                                |
| `error`             | 上游 provider 返回非预期/中途异常（含 5xx/断流等） | 推荐将 provider 错误映射为 HTTP **502/503**（平台视角）或直接 **500**（OpenAI 兼容视角）；OpenAI error：`type=server_error`；`code=provider_error` | **中断流**：直接断流（或终止 chunk）；日志/UsageRecord 标记为 provider error                            |
| `stop` / `length`   | 正常结束                                           | HTTP 200；正常响应 / streaming 正常收尾                                                                                                            | HTTP 200；正常收尾：终止 chunk + `[DONE]`                                                               |

### 约束（保证一致性）

- 任何 EndReason 映射必须满足：
  - **同一个 EndReason 在非 streaming 与 streaming 的首 chunk 前行为一致**（HTTP code + OpenAI error.type）。
  - **首 chunk 后**不承诺结构化错误，但必须：
    - 在日志/UsageRecord 中可还原原因；
    - 对配额类治理（quota_exceeded）要保证“对账口径一致”（见 3.5.2）。

## 3.5.2 —— UsageRecord 的“对外解释模型”

UsageRecord 是“内部事实账本”。本节定义当它被暴露给管理后台 / 管理 API 时的解释口径（不定义 UI/DB）。

### 字段分层

#### A. 核心指标（Core Metrics）

用于列表页、账单、预算看板、SLA 汇总：

- `request_id`
- `api_key_id`
- `provider`
- `model`
- `status`（`completed` / `partial`）
- `confirmed_tokens`
- `chunk_count`
- `start_at` / `first_chunk_at` / `end_at` / `duration`
- `end_reason`

#### B. 调试字段（Debug Fields）

用于排障详情页：

- `error_code`
- `error_msg`
- （未来可扩展）provider request id / upstream status code / route decision / trace id

### status 的业务解释

- `completed`

  - 业务含义：流正常结束（平台认为“完整交付”）。
  - 常见表现：`end_reason=stop|length`。

- `partial`
  - 业务含义：已发生首 chunk（用户已收到内容），但流未完整结束。
  - 常见表现：`end_reason=quota_exceeded|error|internal_error|client_disconnect`。

> 本阶段约束：**首 chunk 前失败通常不生成 UsageRecord**；如未来为了可观测性生成 failed record，需要额外定义其对外展示与是否入账规则。

### quota_exceeded 的账单解释（对外口径）

当 `end_reason=quota_exceeded`：

- **计费口径**：只对 `confirmed_tokens` 对应的已交付部分计费（不追溯撤销）。
- **为什么是 partial**：因为治理发生在 streaming 过程（首 chunk 后），系统选择“停止继续生成”而不是“否认已交付”。
- **如何让用户验证**：
  - `first_chunk_at` 非空 → 证明已开始交付
  - `confirmed_tokens > 0` → 证明存在已确认消耗
  - 结束原因 `quota_exceeded` → 解释为何中断

### completed/partial 的 SLA 统计建议口径

- **SLA 成功率（完成层）**：`status=completed` 的比例。
- **交付成功率（生成层）**：`first_chunk_at != nil` 的比例。
- **治理触发率**：`end_reason=quota_exceeded` 的比例。

## 3.5.3 —— Streaming Summary 的稳定日志口径

### 目标

在生产环境中：

- 一条日志看懂发生了什么
- 一眼判断：用户问题 / 配额问题 / provider 问题 / 网关问题

### 稳定模板（必须固定字段）

建议固定输出一条 **Streaming 完结日志**（无论成功或失败）。字段以 key=value 形式输出，便于检索/聚合。

推荐字段：

- `request_id`
- `api_key_id`
- `provider`
- `model`
- `status`（completed/partial/failed）
- `end_reason`
- `chunks`
- `confirmed_tokens`
- `duration_ms`
- `first_chunk_latency_ms`（若无 first chunk，则为空或 -1）
- `error`（若有错误则输出简短 message；否则省略）

### 判因速查（人类读法）

- `status=failed` 且 `end_reason=quota_exceeded`：通常是 **事前治理** 或 **首 chunk 前 quota 拦截**（可按实现细分）。
- `status=partial` 且 `end_reason=quota_exceeded`：**Streaming 中途 quota 耗尽**（已交付部分有效）。
- `status=partial` 且 `end_reason=client_disconnect`：**用户侧断开**（平台不回滚已交付）。
- `status=partial` 且 `end_reason=error`：**上游异常/断流**。
- `status=partial` 且 `end_reason=internal_error`：**网关内部异常**。

## 非目标（本阶段不做）

- 不规定 provider 之间的细粒度错误码映射（只定义语义分层：quota/client/provider/internal）。
- 不定义 UI 展示、存储结构、索引、采样策略。
- 不定义 tracing 实现（仅定义稳定字段口径）。
