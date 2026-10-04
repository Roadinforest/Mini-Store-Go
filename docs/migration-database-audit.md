# 迁移前实际数据库检查

检查日期：2026-10-04（Asia/Shanghai）。读取 `backend/configs/config.yaml`，未使用 DSN 环境覆盖。连接的是 Neon PostgreSQL 的 `neondb` 数据库、`public` schema；`database.auto_migrate=false`。检查均在 REPEATABLE READ、READ ONLY 事务中完成，没有清理记录或执行迁移。本文不包含连接密码或完整 DSN。

## 当前结构

仍是旧结构：Product.images、Cart.items 及金额列、Order.itemsPrice/totalPrice/isPaid/isDelivered 仍存在。ProductImage、CartItem、ReviewDuplicateArchive 均不存在。

Cart、Review 查询到的索引只有各自主键，尚未建立本次迁移的购物车唯一索引和用户＋商品评价唯一索引。没有证据表明当前库已经完成本次迁移或执行了独立的评价归档清理脚本。

| 表 | 记录数 |
| --- | ---: |
| Product | 1500 |
| Cart | 8 |
| Order | 9 |
| OrderItem | 10 |
| Review | 3961 |

## 评价

- 1059 组 `(userId, productId)` 有多条评价，共涉及 2630 条记录，超过每组一条的数量为 1571 条。
- 每个重复组至少存在两种不同内容。比较字段为 rating、title、description、isVerifiedPurchase，没有将 id 或 createdAt 差异视为内容差异。
- 最大一组有 6 条评价；若执行保留一条规则，活动评价数量将从 3961 降为 2390，影响评分和评价数。
- 所有 3961 条评价分布在 3 个用户、1225 个商品上，时间范围为 2003 至 2023 年。它们可能来自集中导入的历史数据，但仅凭这些信息不能确认原始作者或导入来源。

应先确定业务是否允许同一用户多次评价，再决定保留一条、归档其余记录。不能将这些数据当作相同内容的重复插入直接去重。独立清理脚本默认保留创建时间最新的一条，这是一项数据处理策略，并非已证实的正确业务结论。

## 购物车

Cart.items 的实际类型是 json[]。总共包含 7 个 JSON 对象元素及 398 个 JSON 数字元素：

- 4 个对象使用 product_id，3 个对象使用 productId。
- 1 条购物车包含 398 个数字元素；这些数字可作为字节解码成合法 JSON 数组，数组内有一条商品记录：product_id=B08CR66TK9、qty=1、price=13.99。
- 混用字段名及数字字节数组均未被当前 up 脚本的搬迁逻辑覆盖。
- 购物车 `441d0de0-709b-4fbe-a422-5619068d9992` 中的商品 `aecf8ad6-5391-4a36-96a9-6be11987ace9` 已不在 Product 表中。这会阻止建立 CartItem 的商品外键。
- 未发现 sessionCartId 或非空 userId 重复。

需要在迁移前规范化购物项格式，兼容两种商品 ID 字段并解码数字字节数组；缺失商品的购物项应单独处理并保留原始数据，不应直接忽略或随意伪造商品。

## 订单

订单 `b021c31a-ae0a-4964-a05d-2872b9afc021`：

| 项目 | 金额 |
| --- | ---: |
| 存储的 itemsPrice | 217.92 |
| OrderItem 的 SUM(price * qty) | 199.98 |
| 差额 | 17.94 |
| shippingPrice | 0 |
| taxPrice | 32.69 |
| 存储的 totalPrice | 250.61 |
| 按现有明细＋历史运费税费计算的总价 | 232.67 |

移除旧金额列前，须查明是订单明细缺失、额外费用未建模还是旧金额错误；不能自动选择其中一个数值，改变历史订单金额。

其余订单未发现本次金额一致性查询命中的问题；订单付款/送达布尔标记与时间字段一致，OrderItem 未发现引用不存在的 Product。

## 结论

Duplicate reviews 错误与实际库的唯一性冲突吻合。即使处理这些评价，当前 up 脚本还会遇到购物项格式、缺失商品和历史订单金额问题。现有脚本基于代码模型及构造测试数据编写，未覆盖这些真实历史数据，应先按实际格式与业务策略修订迁移方案，再执行写入。
