# mini-store-go frontend


当前状态：

- 使用 `Vite + React + React Router + Tailwind`
- 不依赖后端，全部基于本地 mock 数据和 `localStorage`
- 已覆盖首页、搜索、商品详情、购物车、登录注册、结算、订单、用户区、后台页

## 启动

```bash
pnpm install
pnpm dev
```

构建验证：

```bash
pnpm build
```

## 测试账号

- `user@example.com / 123456`
- `admin@example.com / 123456`

## 说明

- mock 状态保存在浏览器 `localStorage`
- 删除 `localStorage` 中的 `mini-store-go-mock-state` 可重置数据
