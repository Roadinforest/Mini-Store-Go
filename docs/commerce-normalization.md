# 电商数据结构规范化与历史数据迁移

## 当前脚本的处理策略

`0001_normalize_commerce.up.sql` 已按实际 Neon 数据库检查结果调整。它是一份包含 BEGIN/COMMIT 的完整脚本，不需要先执行单独的评价清理脚本。执行前应停止旧版应用写入，并使用含历史金额差额支持的新版代码。

| 情况 | 迁移行为 |
| --- | --- |
| 同一用户对商品有多条评价 | 按 createdAt 最新、同时间 id 最大保留一条活动评价；重复组全部原始记录先完整归档；重算受影响商品评分与评价数 |
| 购物项使用 product_id 或 productId | 统一转成 CartItem.productId；两个字段同时存在但不同则报错 |
| json[] 内是 UTF-8 字节数字 | 校验每个值为 0–255 整数，重组字节并解码 JSON 数组，兼容中文等多字节字符 |
| json[] 内是 JSON 对象字符串 | 解析后按同一规则校验 |
| 购物项商品已不存在 | 完整保存原购物车及不可用购物项，将不可用项移出活动购物车；不创建虚构商品 |
| 购物车旧金额与当前规则不一致 | 先归档原记录；迁移后根据有效购物项及当前规则计算金额 |
| 订单旧商品金额与明细不一致 | 保存原订单快照，并以 legacyItemsAdjustment 记录差额；不改订单明细或历史账单金额 |
| 订单总价与旧费用分项不一致 | 以 legacyTotalAdjustment 保留总价差额，不将未知差额解释为税费或运费 |
| 购物车唯一性冲突、同车重复商品、无效字段/字节或订单状态矛盾 | 报错并整笔回滚，不自动猜测修复方案 |

创建时间不等同于最后编辑时间。“每组保留最新创建的一条评价”是这份脚本采用的数据整理策略。原库有 1059 个重复组，涉及 2630 条评价；按检查时的快照，执行后活动评价从 3961 条变为 2390 条，2630 条原始记录均进入归档。若不接受该策略，不应执行这份 up 脚本。

## 历史金额

订单商品金额 = `SUM(OrderItem.price * qty) + legacyItemsAdjustment`。

订单总价 = 商品金额 + 历史运费 + 历史税费 + legacyTotalAdjustment。

两种差额只表达原有记录之间的数值差异，并不确认它们是实际收费项目。新订单两项差额都为零；需要调查的历史订单可按差额不为零筛选。

实际检查到的订单 `b021c31a-ae0a-4964-a05d-2872b9afc021`，明细合计 199.98、旧商品金额 217.92。迁移记录 legacyItemsAdjustment=17.94，保持响应商品金额 217.92、总价 250.61。管理员销售额 SQL 同步包含差额，避免报表与原有账单不一致。

## 关系模型与接口

- Product.images 拆为 ProductImage，以 `(productId, position)` 为主键；图片顺序、重复 URL 和封面顺序保留。
- Cart.items 拆为 CartItem，以 `(cartId, productId)` 为主键；保留加购时的名称、slug、图片、单价及数量。sessionCartId 唯一，非空 userId 唯一。
- Cart 不存四个派生金额，读取时用 Decimal 计算。税率 15%；商品金额大于 100 免运费，正数且不超过 100 运费 10，空车为零。
- Order 删除 itemsPrice、totalPrice、isPaid、isDelivered，保留历史运费、税费、地址、支付信息及差额。状态从付款/送达时间是否为空计算。
- OrderItem 保留历史商品快照，不根据商品当前价格或名称更新。
- Review 建立 `(userId, productId)` 唯一约束，购买验证默认 false。

购物车修改、结账和评价聚合在事务中锁定父记录；首次创建由唯一约束协调，付款和送达保持幂等。商品查询、搜索回填和评价商品预加载均读取新图片关系。管理员商品修改不会覆盖评价统计。

响应 DTO 位于 `internal/dto/commerce.go`。图片仍是字符串数组，购物车与订单保留四个金额字段和布尔付款/送达状态，金额为 JSON number；空列表为 `[]`。前端商品写入不再提交 rating、num_reviews。

地址和支付结果按整体 JSON 值对象处理；认证表的业务依赖没有扩展。本次变更不宣称认证业务及 JSON 内部字段均已完成 BCNF 验证。

## 归档

| 表 | 保存的数据 |
| --- | --- |
| ReviewDuplicateArchive | 重复评价组全部原始记录，保留评价 ID、归档时间及完整 rowData |
| CartMigrationArchive | 每条购物车迁移前的完整记录，包括原始 JSON 格式与旧金额 |
| CartItemMigrationArchive | 因商品不存在而移出的购物项快照、位置和原因 |
| OrderMigrationArchive | 每条订单迁移前的完整记录，包括原账单金额 |

归档表没有关联用户、商品或购物车的外键，避免后续业务删除使历史备份级联丢失。原始记录保存在 rowData；归档与搬迁、删除、统计更新同属一个事务。失败不会留下半迁移数据。

## 执行

若上次报错后连接还在中止事务中，先执行 `ROLLBACK;`。然后重新加载并一次执行完整脚本。命令行从仓库根目录运行：

```sh
psql "$DATABASE_DSN" -v ON_ERROR_STOP=1 \
  -f backend/internal/infra/database/migrations/0001_normalize_commerce.up.sql
```

脚本执行前锁定相关表，执行末尾输出本次归档评价数量、不可用购物项数量和有历史差额的订单数量。它只适用于旧结构，不应在已经升级的库上重复直接执行。

也可以显式开启 database.auto_migrate。Go 在自身事务里执行去掉 BEGIN/COMMIT 的脚本正文，然后对齐模型；任何一步失败都回滚。已有新结构不会重复搬迁，空库直接创建当前模型。开关默认关闭。

`0000_report_duplicate_reviews.sql` 仍可独立执行，用于只读检查。`0000_reconcile_duplicate_reviews.sql` 仍可单独执行，用于只处理评价；新版 up 已内置相同策略，不要求它作为前置步骤。

## 回滚

停止新版写入后执行：

```sh
psql "$DATABASE_DSN" -v ON_ERROR_STOP=1 \
  -f backend/internal/infra/database/migrations/0001_normalize_commerce.down.sql
```

回滚重建图片和有效购物项数组，并计算购物车当前金额；按历史差额恢复订单金额及状态列，再删除差额列和新业务表。

归档的重复评价仅在保留评价未被修改、且该条原记录 ID 不在活动表时恢复。如果保留评价已经编辑，归档继续保留，等待人工判断。不会自动重新加入不可用购物项，也不会把已结账的购物车恢复成迁移前状态。所有归档表都会保留；原始字节数组、camelCase 格式和不可用商品快照仍可取回。

回滚后使用旧版代码，关闭新版 AutoMigrate，避免再次升级。

## 验证范围

PostgreSQL 测试覆盖原始结构升级、事务失败回滚、直接执行完整脚本、重复评价归档、中文 UTF-8 字节数组、两种 ID 字段、缺失商品、17.94 历史差额、总价差额、down/up 往返，以及原有购物车、订单、评价和销售额逻辑。实际数据库检查记录保存在 `docs/migration-database-audit.md`，该文件描述检查时状态，并不代表已经执行了新版迁移。

## 0002 增量对齐

已经完成 0001 的数据库执行 `0002_align_commerce.up.sql`，不要再次执行 0001。先停止应用写入，执行完整文件：

```sh
psql "$DATABASE_DSN" -v ON_ERROR_STOP=1 \
  -f backend/internal/infra/database/migrations/0002_align_commerce.up.sql
```

0002 在一个事务中完成以下操作：

- 重算所有商品的评价聚合，只更新不一致的商品；无评价的商品归零。原评分和评价数先写入 ProductReviewStatsArchive，便于审计。
- 补齐 Order.userId、Review.userId/productId 的模型辅助索引。
- Order.shippingAddress/paymentResult 和 User.address 转为 jsonb。
- 按数据库所有者确认的 Asia/Shanghai 解释旧 timestamp without time zone，显式 AT TIME ZONE 转成 timestamptz，并保留原时间精度。比如旧值 2026-01-01 12:34:56.123 转换后的时间点是 2026-01-01 04:34:56.123 UTC；这是同一业务时间点。已有 timestamptz、归档数据和 Prisma 迁移元数据不转换。

脚本固定操作 public schema，锁等待超过 10 秒会失败并回滚，失败连接先 ROLLBACK 再重试。可以独立重复执行：已一致的统计不新增归档，已存在的索引不重复创建，已转换的时间不再次解释。执行后返回 corrected_products；以先前只读核验快照为基准应为 729，若期间评价变化则以当时数据库为准。

0002 不改订单金额字段或订单明细。当前 AutoMigrate 没有自动调用 0002，需要显式执行此文件。PostgreSQL 测试验证评分归零/重算、原统计归档、索引、JSON 类型、Asia/Shanghai 转换、毫秒精度、NULL、不同会话时区、订单金额保持以及重复执行。

## 0003 移除未使用的旧认证表

当前后端注册、登录、刷新均采用 User + JWT，未使用 Account、Session、VerificationToken。对应模型、User.Accounts/User.Sessions 关联、模型迁移注册已经移除，AutoMigrate 不会重新创建三张表。Cart.sessionCartId 仍用于匿名购物车，继续保留。

执行清理脚本：

```sh
psql "$DATABASE_DSN" -v ON_ERROR_STOP=1 \
  -f backend/internal/infra/database/migrations/0003_remove_unused_auth_tables.up.sql
```

0003 先锁定现有的旧认证表，将完整原始记录写入 LegacyAuthArchive，再删除三张活动表；归档没有关联 User 的外键。脚本不使用 DROP CASCADE，遇到额外依赖会整体回滚。可以重复执行，也支持旧认证表已经不存在的数据库。0002 已调整为只对齐当前业务表，清理后重复执行不会依赖这三张旧表。

2026-10-04 使用本地配置检查实际库时，三张旧认证表均为 0 行。清理前业务表数量：User 4、Product 1500、Cart 8、CartItem 7、Order 10、OrderItem 11、Review 2390。清理脚本不修改这些表的数据。

测试覆盖有数据的旧表完整归档、重复执行、额外依赖失败回滚、新库建表及清理后 AutoMigrate 均不创建旧表，并执行了数据库和认证路由的 race 测试、go vet 与编译检查。

同日已执行 0003 并只读核验：Account、Session、VerificationToken 均不存在，LegacyAuthArchive 为 0 行；上述业务表记录数全部保持一致。
