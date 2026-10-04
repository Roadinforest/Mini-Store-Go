# 迁移后数据库核验

检查日期：2026-10-04（Asia/Shanghai）。通过项目 config.Load() 加载本地配置，连接 neondb/public；全部查询位于 REPEATABLE READ、READ ONLY 事务，没有执行 AutoMigrate 或修改数据库。

## 结论

0001 commerce 迁移的核心结构及历史数据处理符合预期。另有既有商品评分缓存不一致，以及尚未完成 GORM 全部类型和辅助索引对齐的问题；不能将本次迁移成功等同于整个数据库与模型完全一致。

## 结构

- Product.images 已移除；ProductImage 存在，主键 (productId, position)，商品外键 ON DELETE CASCADE，position >= 0。
- Cart.items 及旧金额字段已移除；CartItem 存在，主键 (cartId, productId)，Cart/Product 外键已验证，qty > 0、price >= 0 检查约束有效。
- cart_session_idx 唯一索引、cart_user_idx 非空用户唯一索引均存在。
- Review 的 review_user_product_idx 唯一索引存在；isVerifiedPurchase 默认 false。
- Order.itemsPrice/totalPrice/isPaid/isDelivered 已移除；paidAt/deliveredAt 保留；两个 legacy adjustment 字段存在，为 numeric(12,2)、NOT NULL、默认 0。
- 所有相关金额字段为 numeric(12,2)，Product.rating 为 numeric(3,2)。

## 数据

| 表 | 记录数 |
| --- | ---: |
| Product | 1500 |
| ProductImage | 8806 |
| Cart | 8 |
| CartItem | 7 |
| Order | 9 |
| OrderItem | 10 |
| Review | 2390 |
| ReviewDuplicateArchive | 2630 |
| CartMigrationArchive | 8 |
| CartItemMigrationArchive | 1 |
| OrderMigrationArchive | 9 |

- 重复用户/商品评价为 0；1059 个原重复组保留的评价均符合 newest createdAt、id DESC 策略，全部保留者仍存在，移出的评价均不在活动表。
- 购物项、商品图片、订单项、评价均无孤立引用；购物项及订单项的数量/价格无非法值，评价 rating 均在 1 至 5。
- 商品图片 position 全部从 0 连续排列。
- 缺失商品购物项归档 1 条，reason=missing_product。
- 9 笔订单按新公式计算并与 OrderMigrationArchive 原金额比较，差异为 0。
- 订单 b021c31a-ae0a-4964-a05d-2872b9afc021 保留 legacyItemsAdjustment=17.94、legacyTotalAdjustment=0，原小计 217.92、总额 250.61 保持一致。

## 剩余差异

1. 商品评价统计：729 个 Product 的 rating 或 numReviews 与当前 Review 聚合不同，其中 rating 不同 693 个、numReviews 不同 728 个。这些商品均不属于本次重复评价清理影响的商品；所有受重复评价清理影响的商品统计均正确。当前 SQL 只重算重复组关联商品，因此未修正其他商品的既有缓存不一致。示例：商品 3886f8ea-8174-47e2-81d5-ce2cf4848a23 存储 rating=3、numReviews=1，实际评价为 0 条，聚合评分应为 0。
2. 旧时间列多数仍为 timestamp without time zone，而新 CartItem.createdAt 为 timestamptz；Go/GORM 对齐通常采用 timestamptz。Order.shippingAddress/paymentResult、User.address 仍为 json，而当前模型指定 jsonb。
3. 模型中声明的 Order.userId 和 Review.userId/productId 辅助索引尚不存在；本次迁移要求的唯一索引已存在。这影响模型完整对齐及潜在查询性能，不影响已验证的业务唯一性。

在当前业务规则下，ProductImage 和 CartItem 已建立正确的复合主键，评价的业务候选键得到唯一性保护，Cart/Order 的可计算冗余字段已移除。购物项和订单项的名称、价格等属于历史快照，不应套用当前 Product 字段之间的函数依赖。评分缓存不一致属于数据一致性问题，并不能单凭它判定表违反 BCNF；全库 BCNF 证明仍依赖完整业务函数依赖。
