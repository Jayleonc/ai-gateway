# internal/admin

一句话结论：`internal/admin` 是 **AI Gateway 的只读管理/可观测视角**，用于解释系统当前状态（Allowance / Quota / Usage），而不是参与请求执行或业务决策。

## 1) Purpose（为什么存在）

`internal/admin` 提供面向运维与排障的“解释性视图”（explanatory views）：

- 解释系统现在处于什么状态（配额、使用量、Allowance 等）。
- 让人能够观察与核对 runtime 行为的结果。

它的定位是 **read-only 的观测与汇总**，不是 runtime 链路的一部分。

## 2) Non-Goals（不做什么）

为了防止未来把 Admin 写歪，这里明确约束：

- **不参与 runtime 决策**：不做鉴权决策、不做路由决策、不做模型选择。
- **不影响 Streaming / Quota enforcement**：Admin 的调用不应改变任何限流/扣减/中止策略。
- **不是 Gateway 的一个“子业务”**：不承载面向最终用户的业务流程。
- **不是 Billing 系统**：不负责计费对账、结算、出账等面向财务的语义。
- **默认只读**：不引入对核心数据的写入路径（除非未来演进章节中明确触发条件被满足）。

## 3) Layering Model（分层模型）

目录结构对应的职责边界如下：

- `handler/`：HTTP/JSON 边界层，仅做参数解析、调用 service、返回结果。
- `service/`：视图组合层（View Composition），**只读地组合**来自 `identity / metering / quota` 等模块的现有信息，形成可理解的 Admin 视图。
  - 不做数据持久化
  - 不拥有任何表/存储
- `types.go`：Admin API 的返回类型（View Models），用于表达“观测视图”的结构。

关键约束：**Admin Service 不是 Repository，不封装持久化逻辑。**

## 4) Why There Is No Repo (Yet)（为什么现在没有 repo / domain）

在本项目里，`repo` 被定义为 **持久化抽象（persistence abstraction）**：

- 其职责是屏蔽存储细节，并围绕“本域的数据”提供一致的读写接口。

但当前 `internal/admin` 的本质是“视图组合”，它：

- 不拥有自己的业务数据
- 不引入新的持久化需求
- 只是读取并解释其它域已经存在的事实

因此此阶段不引入 `internal/admin/repo` 或 `internal/admin/domain`：

- 避免让“组合视图”伪装成“领域/持久化层”，造成语义混淆。
- 避免在架构上暗示 Admin 对数据有所有权。

## 5) Future Evolution（什么时候会变）

当且仅当 Admin **产生并拥有自己的业务数据** 时，这里的分层才会演进，例如：

- Admin 用户与权限（RBAC）
- 审计日志（Audit Log）
- 管理侧配置与策略（需要被持久化、版本化、可追溯）

一旦出现上述需求，将引入：

- `internal/admin/domain/*`：Admin 自己的领域模型与业务语义
- Admin 自己的 repository（与 Gateway 其它 domain 平级），用于承载“Admin 拥有的数据”的持久化抽象

在那之前，`internal/admin` 将保持为 **只读的 View Composition**，不引入 repo 以维持语义清晰。
