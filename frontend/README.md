# mini-store-go frontend

前端使用 Vite、React、React Router 和 Tailwind，业务数据来自 Go API。Store 不再加载 mock 数据或持久化账户、订单和库存；启动时会删除旧的 `mini-store-go-mock-state` 缓存。

## 启动

需要 Node.js 22、pnpm 10.28.2 和正在运行的后端：

```sh
npx --yes pnpm@10.28.2 install --frozen-lockfile
npx --yes pnpm@10.28.2 dev
```

开发环境默认 API 地址是 `http://localhost:8080/api/v1`，可通过 `VITE_API_BASE_URL` 覆盖。后端 CORS 必须允许前端地址并允许携带 cookie。生产构建使用同域 `/api/v1`，由部署的反向代理转发到后端。

账号通过真实 API 注册，管理员角色由后端管理。

## 验证

```sh
npx --yes pnpm@10.28.2 lint
npx --yes pnpm@10.28.2 test
npx --yes pnpm@10.28.2 build
```

已配置对应版本 pnpm 的环境可直接使用 `pnpm`。测试通过 Vitest 与 Testing Library 运行，不需要真实后端；业务与权限集成测试在 backend 中使用 PostgreSQL 和 Redis 执行。实现说明见 [工程修复说明](../docs/engineering-medium-fixes.md)。
