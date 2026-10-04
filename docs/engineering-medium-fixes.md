# 中高与中优先级工程修复

本分支处理原工程审查的测试与 CI、AI 流式工具调用、前端 mock 与真实接口边界、API 解码、健康检查，以及搜索降级观测。库存 Saga 的设计仍见 `redis-inventory-saga-design.md`，本次没有实现该设计或改变库存一致性协议。

## 1. 测试与 CI

- 后端在 PostgreSQL 的独立 schema 中测试订单并发付款、并发结算与回滚，以及购物车计价、库存限制、评论更新与聚合、订单访问权限和管理员权限。
- Redis 测试使用独立 UUID 键，验证多商品预占失败不发生部分扣减、并发预占不超量、重复释放不增加库存。测试不会清空 Redis 实例；仍应使用专用测试实例。
- 前端使用 Vitest、Testing Library 和 jsdom。测试覆盖按端点解码、空分页、无效响应、并发刷新凭证、Store 同步、跨会话响应隔离、SSE 分片与取消，以及聊天失败的回退条件。
- GitHub Actions 分别运行后端格式检查、`go vet`、真实 PostgreSQL/Redis 集成测试与构建，以及前端冻结锁文件安装、lint、测试和构建。测试中不调用真实 AI、Pinecone 或 Qwen 服务。

本地运行：

```sh
cd backend
export TEST_DATABASE_DSN='postgres://postgres:test@127.0.0.1:5432/mini_store_test?sslmode=disable'
export TEST_REDIS_ADDR='127.0.0.1:6379'
make test-integration
go vet ./...
go build ./cmd/api
```

`make test-integration` 要求两项环境变量均已设置。普通 `go test ./...` 在未配置依赖时会跳过对应集成测试。PostgreSQL 测试账号必须能创建和删除 schema。

```sh
cd frontend
npx --yes pnpm@10.28.2 install --frozen-lockfile
npx --yes pnpm@10.28.2 lint
npx --yes pnpm@10.28.2 test
npx --yes pnpm@10.28.2 build
```

前端使用 Node.js 22 与 package.json 指定的 pnpm 版本。已正确配置 Corepack 或 pnpm 10.28.2 的环境可直接使用 `pnpm`。

## 2. AI 流式工具调用

同步与流式接口共用最多八轮的 agent 循环。流式接口收齐一轮的消息，使用 Eino 的 `ConcatMessages` 合并工具参数碎片，再执行工具，将原 assistant 调用与 tool 结果加入同一对话，继续下一轮流式生成。移除了识别工具意图后重新调用同步 Chat 的路径。

`ai.timeout` 是整次对话的预算，包含上下文查询、工具执行与所有模型轮次。请求取消时停止后续执行并关闭流；模型 SDK 和工具必须遵守传入的 context。工具名未知或结构化参数非法时显式报错。

事件顺序可以是：

```text
thinking → partial*
         → tool_call → tool_result → thinking → partial*
         → complete / navigation → [DONE]
```

服务端在分片边界暂存控制标签，不把思考内容和工具参数直接显示给用户。日志记录请求 ID、耗时、工具数量和响应字节数，不记录用户完整对话和模型内部思考。

前端检查 SSE 类型与事件结构，要求收到结果终态和 `[DONE]`；中断、无终态或错误事件均报错。组件卸载会取消请求并释放 reader。只在流式端点返回 404/405 时使用同步接口；5xx、读取中断和已经开始输出的请求不会再次生成。

## 3. 前端数据边界

Store 从空业务状态启动，仅接收 API 返回的业务结果；移除了 mock 数据模块、本地业务写入方法，以及整个业务状态的 localStorage 持久化。启动时尽力删除旧 mock 缓存，存储不可用不会阻止启动。

同步商品和评论使用稳定回调，数据在 reducer 中按 ID 合并，页面 effect 显式声明这些依赖。异步账户操作使用会话版本检查；登录切换和退出登录时清理账户、订单及购物车，旧会话的响应不能覆盖新会话状态。新增购物车商品不再要求商品预先存在于前端缓存。

后端退出登录会轮换购物车 cookie，随后前端读取新的游客购物车。退出失败会显示错误，保留当前账户状态。首页与搜索页显示接口失败信息，不再把请求失败完全隐藏为正常空列表。

## 4. API 解码

每个端点显式传入对应的 Zod schema；先校验 envelope 与字段类型，再转换 snake_case 到页面使用的 camelCase。不再依据对象是否存在几个字段或空数组推断返回类型。

空商品、用户和订单分页由各自端点合同解释。数值字段收到字符串、缺少必要字段或收到其他资源类型时返回失败结果。商品 `images: null` 按现有后端输出兼容为空数组；未提供的账户创建时间不再伪造为当前时间。并发 401 共用一次凭证刷新，各原请求最多重试一次。

目前 schema 与 Go presenter 手工对应，新增或修改接口时需同步 schema 与合同测试；这不是从 OpenAPI 自动生成客户端。

## 5. 健康检查

- `GET /healthz`：进程存活，返回 200，不因外部依赖失效而失败。
- `GET /readyz`：并行执行 PostgreSQL `PingContext` 与配置的 Redis `PING`，总等待上限两秒。依赖不可用返回 503 和逐项状态。没有配置 Redis 时标记 disabled，不阻止就绪。

部署时将 liveness 指向 `/healthz`，readiness 指向 `/readyz`。该探测验证连接可用性，不验证业务表结构、上游模型服务或完整下单路径。

## 6. 搜索降级观测

向量检索与全文检索并行执行，每个检索阶段独立使用 `search.timeout`。向量服务失败时保留全文结果；重排失败时保留 RRF 排序；两个检索都没有命中时进入 repository 查询。

`search degraded` 告警日志记录阶段、原因和耗时；`search fallback` 记录 repository 降级耗时、命中数和是否成功。正常关闭或未配置的向量服务不记为故障。用户取消请求会返回 context 错误，不被当作普通空结果隐藏。上游 HTTP 错误响应正文不进入错误或日志。

这里提供结构化日志，尚未引入 Prometheus 指标或告警平台。可按 stage/reason 汇总故障率与延迟，并监控 repository 降级比例。
