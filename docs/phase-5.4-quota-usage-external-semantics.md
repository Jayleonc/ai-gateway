# Phase 5.4 — Quota / Usage 对外语义与预算模型（非实现）

本阶段目标：让系统能够“解释清楚自己现在在做什么”，而不是“把它做得更精确”。

本文面向：

- 对外沟通（Admin / Internal API / 运维观察）的统一口径
- 对内协作（后续 Phase 6+ 的建模与实现）

本文不改变任何现有行为，仅定义“现状语义”。

本文写作原则：

- 我们用“对外可解释”为目标，而不是“更精确”
- 我们描述“系统已经在做什么”，而不是“系统应该怎么做”
- 我们将所有预算消耗统一口径解释为 token，即使内部估算很粗

---

## 术语与对象（对齐口径）

为了避免“同一个词在不同语境含义不同”，本文先定义几个对外解释时会使用的对象：

- **Quota（配额 / 预算）**

  - 一个上限预算，概念单位为 token
  - 对外仅解释为“剩余 token 预算”，不解释为“请求次数”

- **remaining**

  - Quota 的剩余值（同一个 bucket 的剩余预算）

- **estimated tokens**

  - 估算 token：用于事前准入，允许非常粗

- **confirmed tokens**

  - 确认 token：用于事中治理与事后对账，语义上更接近“实际交付内容的规模”

- **UsageRecord**

  - 请求结束后的可解释记录，用于“对账、解释、定位”，不用于实时决策

- **Admission / Governance / Accounting**
  - 三个阶段性语义角色（下文 2.2 详述），它们可能共享同一份 remaining，但含义必须分层解释

## 0️⃣ 背景与动机（阶段性现象说明）

当前系统存在如下现象（已知、且被允许）：

- non-streaming 请求：

  - quota 看起来像「调用次数」

- streaming 请求：
  - quota 会在生成过程中按 token 消耗并中断

这并非 bug，而是以下事实共同作用的结果：

- QuotaStore 当前被同时用于：

  - 事前准入（Admission）
  - 事中治理（Streaming Governance）

- 但 Quota 的“单位语义尚未被显式建模”
  - 同一个 remaining 在不同阶段被不同粒度地解释与消耗

本阶段任务不是修复该现象，而是将其明确为“阶段性语义”，并对外解释清楚：

- 这是一个一致但粗糙的中间态
- streaming 与 non-streaming 的差异来自估算粒度不同，而非规则冲突
- 更细的 quota 单位模型在后续阶段引入

换句话说：当前系统的 quota 语义处于“能工作、能解释、能治理”的中间态。

- 如果你从**用户体验**看，它可能表现得“不够直觉”（例如 non-streaming 像次数、streaming 像 token）
- 如果你从**系统治理**看，它是合理的：
  - 请求开始前有一层最小成本的准入
  - 请求进行中有一层可中断的防失控治理
  - 请求结束后有一份可查询的解释记录

---

## 1️⃣ Phase 5.4 的边界（明确“不做什么”）

### 本阶段不做

- 不引入 money / price / billing
- 不区分 request quota / token quota 的多桶模型
- 不改变现有 quota 行为
- 不追求精确性或公平性

### 本阶段要做

- 定义 Quota 的对外语义
- 定义 Quota 与 UsageRecord 的关系
- 定义 quota_exceeded 在 Admin / API 视角的解释口径

> 说明：本文不要求 quota 变“更准”。相反，我们接受它目前很粗，并把这种粗糙明确写进语义。

---

## 2️⃣ Quota 的“对外语义模型”（Phase 5.4 核心）

### 2.1 Quota 是什么（当前阶段）

Quota = 一个“预算上限”，单位为 token（概念上）。

- 所有消耗最终都以 token 解释
- non-streaming 的 token 估算是极粗粒度的
- streaming 的 token 估算是交付驱动的

当前系统只有一个 QuotaBucket（即一个 remaining），被多个治理阶段复用。

> 说明：这里的 token 是“概念单位”。当前系统对 token 的估算可能非常粗糙，但对外解释仍以 token 为统一口径。

**对外解释的核心不变量：**

- remaining 的减少，永远解释为“预算（token 口径）被消耗”
- 是否能完整生成，取决于 remaining 是否足以支撑“生成过程的持续交付”
- UsageRecord 是对外解释的权威来源（最终发生了什么，以 UsageRecord 为准）

### 2.1.1 请求生命周期视图（推荐对外解释用）

可以用下面这个时间线向外解释（这是语义视图，不是实现视图）：

1. **Admission（事前）**：请求是否允许开始？
2. **Streaming Governance（事中）**：生成过程中是否仍被允许继续？
3. **Accounting（事后）**：最终发生了什么？系统用 UsageRecord 解释给你看。

### 2.2 Quota 的三种角色（语义区分，不是实现区分）

⚠️ 注意：这是语义分层，不是代码结构分层。

#### A. Admission Quota（事前）

- 发生在：Policy.Evaluate
- 单位：估算 token（最小为 1）
- 目的：
  - 防止无限请求
  - 防止明显滥用
- 特点：
  - 精度低
  - 允许“看起来像按次数”

补充说明：

- Admission 阶段的核心是“用最小成本阻止明显不合理的请求继续占用资源”。
- 当 estimatedTokens 缺失或过低时，Admission 的扣减会退化为最小扣减（例如 1）。
- 因此 non-streaming 场景会自然呈现“像按次数扣”的现象。

#### B. Streaming Governance Quota（事中）

- 发生在：Streaming Runtime（QuotaObserver）
- 单位：confirmed tokens（以交付为驱动的估算与确认）
- 目的：
  - 防 runaway generation
  - 保证预算不会被单次请求耗尽
- 特点：
  - 精度中
  - 可中断

补充说明：

- Governance 阶段的核心是“防 runaway generation”。
- 即使 Admission 允许开始，只要生成过程中预算不足，系统允许中断。
- 这种“允许开始但不保证完整生成”的语义，是本阶段明确允许的设计。

#### C. Accounting / Usage（事后）

- 发生在：OnEnd → UsageRecord
- 单位：最终 confirmed tokens
- 目的：
  - 对账
  - 解释
- 特点：
  - 精度高
  - 不参与实时决策

补充说明：

- Accounting 阶段不解决“该不该继续生成”的问题，只解决“发生了什么”的问题。
- 对外解释时，优先引用 UsageRecord，而不是引用 runtime 日志。

**Phase 5.4 的结论：**

- 当前系统已完整实现 B + C
- A 仍是最小可行形态（以“能解释、可控”为目标，而非精确）

### 2.3 Quota 与 UsageRecord 的关系（一句话版本）

Quota 负责“限制还能消耗多少预算”，UsageRecord 负责“解释本次请求最终消耗了多少以及为什么结束”。

---

## 3️⃣ quota_exceeded 的对外解释模型

### 3.1 在 UsageRecord 中的表现

当发生 quota_exceeded（事中中断）时：

- `status = partial`
- `end_reason = quota_exceeded`

对外解释为：

> “该请求已开始生成，并交付了部分内容，但因预算限制被中断。”

可验证字段（用于解释与定位）：

- `first_chunk_at != nil`
- `confirmed_tokens > 0`（在多数“已交付内容”的场景成立）
- `end_reason = quota_exceeded`

> 补充：在极端情况下，如果系统在极早期中断（例如第一帧不含内容、或估算粒度极粗导致立即中断），可能出现 `confirmed_tokens` 接近 0 的记录。这仍属于“阶段性语义可接受范围”，解释口径不变。

### 3.1.1 对调用方（API 使用者）的体验解释

在 streaming 场景，quota_exceeded 通常表现为：

- HTTP 仍为 200（因为流已经开始）
- 客户端收到部分 `data: ...` chunk
- 随后收到 `data: [DONE]`
- UsageRecord 显示 `status=partial` 且 `end_reason=quota_exceeded`

对外口径建议：

> “你的请求已经开始产出，但预算不足以完成整个生成，因此提前结束。你收到的内容是已交付部分。”

### 3.2 在 Admin / Internal API 视角的解释

当用户看到：

- `remaining` 很小
- streaming 请求很快被中断

正确解释是：

> “系统允许你开始请求（Admission 通过），但预算不足以支撑完整生成，因此在生成过程中被治理逻辑中断。”

这是设计允许的行为，不是异常。

### 3.3 可观测字段对照（用于排障但不改变语义）

当你需要解释“为什么会 quota_exceeded”时，推荐按以下字段顺序对外说明：

1. **remaining（事前与事中共享的预算余额）**
2. **UsageRecord.confirmed_tokens / chunk_count（已交付规模）**
3. **UsageRecord.first_chunk_at / duration（是否进入生成阶段，以及持续多久）**
4. **UsageRecord.end_reason（最终结束原因）**

其中 UsageRecord 是对外解释的主要证据。

---

## 4️⃣ remaining 的当前语义（必须清楚）

### remaining 表示什么？

`remaining` = 还能允许多少“token 级预算消耗”。

它不是：

- 剩余调用次数
- 剩余完整请求数

### 为什么 non-streaming 下看起来像“次数”？

因为 Admission 阶段的估算非常粗：

- `estimatedTokens` 可能为 0 或未提供
- 系统会将其下限提升为 1

结果就是：每次 non-streaming 请求在 Admission 处扣减 1，看起来像“调用次数”。

**这不是 quota 的单位变了，而是估算退化成了最小扣减。**

### 4.1 示例：为什么 streaming 看起来“更像 token”？

假设：remaining=10。

- non-streaming：

  - Admission 估算退化为 1
  - 你可能感觉“还能调用 10 次”

- streaming：
  - Admission 先扣 1（仍然可能退化）
  - 生成过程中每次交付内容都会按 confirmed/estimated tokens 扣减
  - 因此你会感觉“能生成多少内容”与 remaining 更直接相关

这两者的差异不是 quota 规则不一致，而是两种场景对 token 的可观测性不同：

- non-streaming：只有一次性返回，事前只能极粗估算
- streaming：每次交付都是一个“可计量点”，因此治理更细

### 4.2 重要澄清：remaining 与“完整请求数”的关系

在本阶段语义中：

- remaining 不能被解释为“还允许多少个完整请求”
- remaining 只能被解释为“还允许消耗多少 token 级预算”

因此：

- remaining 很小但仍允许开始生成，是允许的
- remaining 足够大但仍可能中断（例如估算误差、治理策略）也是允许的

---

## 5️⃣ 本阶段的稳定结论（可作为文档结尾）

- Quota 的行为在当前阶段是一致但粗糙的
- streaming 与 non-streaming 的差异是估算精度差异，而不是规则差异
- 该状态是 Phase 3–5 的合理中间态
- 更精细的 quota 单位模型将在 Phase 6 引入

---

## FAQ（常见误解与统一回答）

### Q1：为什么 non-streaming 像“按次数扣”？

因为 Admission 阶段用 estimatedTokens 做最小成本准入，当估算缺失/为 0 时会退化为最小扣减（例如 1）。对外仍应解释为 token 预算，只是估算很粗。

### Q2：为什么 streaming 会“生成到一半被掐断”，但 HTTP 还是 200？

因为流已经开始交付。对外语义是“已交付部分内容，随后因预算限制中断”，对应 UsageRecord：`status=partial`、`end_reason=quota_exceeded`。

### Q3：quota_exceeded 是异常吗？要不要报警？

不是异常，它是治理策略的一种正常结果。是否报警取决于你是否将其视为产品体验问题或滥用信号，但在本阶段语义里它是允许发生的。

### Q4：为什么 UsageRecord 里 confirmed_tokens 可能为 0，但 end_reason 是 quota_exceeded？

在极早期中断、首帧无内容、或估算粒度极粗的情况下，可能出现“进入 streaming 流程但实际可确认交付规模非常小”的记录。它仍表达同一个事实：请求尝试生成，但因预算限制被中断。

---

## Phase 6 方向（仅语义展望，不是本阶段交付）

Phase 6 将引入更显式的单位建模与更细的预算桶语义，使得：

- Admission 与 Governance 使用的预算单位与桶更加清晰
- remaining 可以在对外解释上区分“请求级预算”与“token 级预算”（多桶）
- 估算与确认的口径更一致，减少“看起来像次数”的错觉

本文到此为止：Phase 5.4 仅确保现状语义可解释。
