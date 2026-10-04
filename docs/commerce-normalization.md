# 电商数据结构规范化

分支：`codex/normalize-commerce-schema`。

## 持久化结构

| 表 | 变化与业务含义 |
| --- | --- |
| Product | 删除 `images` 数组列；保留商品评分、评价数作为由评价服务维护的缓存 |
| ProductImage | 主键 `(productId, position)`；按 position 保留图片顺序，允许重复 URL，第一张仍为封面；删除商品时级联删除图片 |
| Cart | 删除 `items`、全部四个金额列；sessionCartId 唯一，非空 userId 唯一；一个用户保留一个活动购物车 |
| CartItem | 主键 `(cartId, productId)`；保存加购时的名称、slug、图片、单价及数量；数量必须大于零，单价非负；删除购物车时级联删除明细 |
| Order | 删除 itemsPrice、totalPrice、isPaid、isDelivered；保留历史运费、税费、收货地址和支付信息；付款、送达状态由时间字段是否为空计算 |
| OrderItem | 保留下单商品快照，不从商品当前名称、价格、图片回填 |
| Review | 新增 `(userId, productId)` 唯一索引，购买验证默认 false |

购物车金额按明细读取时计算，税率 15%，商品金额大于 100 免运费，正数且不超过 100 运费 10，空购物车运费为零。全程使用 Decimal，税费按两位小数四舍五入。

订单商品金额按历史明细 `SUM(price * qty)` 计算，总价加上订单已保存的运费、税费。修改商品价格、用户地址或未来计价规则不改变历史订单。管理员销售额查询同步改为按订单明细及历史费用求和，避免引用已删除的 totalPrice 列。

CartItem 与 ProductImage 消除了重复组；Cart 的派生金额、Order 的分项到总价依赖不再出现在基础表内。地址、支付结果仍按整体 JSON 值对象处理；Account、VerificationToken 的业务依赖尚未扩展。本次变更不据此宣称所有嵌套字段和认证业务都已完成 BCNF 验证。

## 业务与接口

购物车增减、清空在事务中锁定父表后读写明细，防止丢失并发修改或在结账后重新写回旧商品。首次创建用唯一索引和 INSERT ON CONFLICT 协调；登录用户先锁定用户记录，防止不同会话并发创建多个购物车。登录后按 userId 解析已有购物车，保留其原 sessionCartId；没有用户购物车时可接管当前会话的游客购物车。保留原来已有用户购物车优先的行为，不自动合并两辆车。不能接管其他用户已拥有的会话购物车。

下单在相同锁顺序下读取明细、计算费用、建立订单快照并删除购物项；任何失败整笔回滚。付款和送达仍幂等，时间字段是状态的唯一存储依据。

评价写入先锁定商品，串行更新该商品的评价及聚合缓存；联合唯一索引兜底禁止重复评价。创建和更新评价时，只有存在该用户该商品的已付款订单才设置 verified_purchase 为 true。旧评价验证标记保留，下一次编辑时按实际购买情况更新。

响应定义移到 `internal/dto/commerce.go`。商品图片仍返回字符串数组，购物车与订单仍返回 items_price、shipping_price、tax_price、total_price 和布尔付款/送达状态，金额仍为 JSON number。空列表返回 `[]`。管理员商品写入不再接受 rating、num_reviews；前端提交类型及请求已同步移除它们，商品编辑不会覆盖评价统计。

## 升级与回滚

迁移文件位于：

- `backend/internal/infra/database/migrations/0001_normalize_commerce.up.sql`
- `backend/internal/infra/database/migrations/0001_normalize_commerce.down.sql`

升级须与新版代码配套，停止旧版应用写入后执行。下面命令从仓库根目录运行，DATABASE_DSN 指向待升级数据库：

```sh
psql "$DATABASE_DSN" --single-transaction -v ON_ERROR_STOP=1 \
  -f backend/internal/infra/database/migrations/0001_normalize_commerce.up.sql
```

up 脚本先锁表并检查重复购物车、重复评价、购物车费用与当前规则、历史订单费用与明细，以及订单状态与时间的一致性；然后搬迁图片与购物项，最后删除旧列。购物项中重复商品、无效数量/单价、缺失字段或无对应商品会被主键、非空、CHECK、外键拒绝。任何错误都回滚整个事务，应按报错先处理旧数据后重试，不会自动删重复记录。

开发环境也可以显式开启 database.auto_migrate：AutoMigrate 在事务中检测旧 Cart.items，先执行该升级脚本再对齐当前模型；已有新结构重复执行不会再次搬迁。空数据库直接创建新版结构。此开关默认关闭，不会在普通启动时自动升级。

回滚先停止新版写入，再执行：

```sh
psql "$DATABASE_DSN" --single-transaction -v ON_ERROR_STOP=1 \
  -f backend/internal/infra/database/migrations/0001_normalize_commerce.down.sql
```

down 根据当前明细重建数组、按当前购物车规则重建费用、恢复订单历史金额及状态列，然后删除新表和新唯一索引。购物车金额缓存恢复的是回滚时计算值，购物项内部 createdAt 没有对应旧格式字段；订单历史快照与费用、图片及购物项显示顺序保留。回滚后应使用旧版代码，并关闭新版 AutoMigrate，以免再次升级。

## 验证

新增测试覆盖旧结构升级、失败后的完整回滚、down/up 往返、新数据库初始化、图片顺序与事务替换、历史订单快照、购物车并发创建和加购、评价并发唯一性、真实购买校验，以及运费临界点和十进制舍入。使用专用 PostgreSQL 测试数据库，每个测试建独立 schema 并清理。
