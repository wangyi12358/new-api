# 用户数据迁移文档：MySQL 到 PostgreSQL

## 适用范围

这份文档用于指导将 `new-api` 项目的用户相关数据，从 MySQL 实例迁移到 PostgreSQL 实例。

迁移目标：

- 保留用户账号本身
- 保留用户 Token / API Key 可继续使用
- 保留 Passkey / 2FA / OAuth 绑定等安全登录信息
- 保留订阅、充值和支付状态
- 按需保留日志、任务、签到等历史数据

本文档基于当前项目代码中的模型与迁移逻辑整理，主要参考：

- [model/main.go](/Users/wangyi/solowin/code/kovar-v2/new-api/model/main.go:258)
- [model/user.go](/Users/wangyi/solowin/code/kovar-v2/new-api/model/user.go:24)
- [model/token.go](/Users/wangyi/solowin/code/kovar-v2/new-api/model/token.go:14)

## 迁移原则

1. 尽量保留原始主键 ID。
2. 有依赖关系的表，必须先迁父表/引用表，再迁子表。
3. 先让 PostgreSQL 目标实例启动一次，由应用自动建表。
4. 迁移期间应冻结写入；至少要冻结被迁移用户的登录、充值、消费和创建 Token 操作。
5. 导入完成后，必须重置 PostgreSQL sequence。

## 建议迁移的表

### 必迁表

这些表是用户迁移后能继续正常使用系统的最小集合。

1. `users`
2. `tokens`
3. `passkey_credentials`
4. `two_fas`
5. `two_fa_backup_codes`
6. `user_oauth_bindings`
7. `user_subscriptions`

### 强烈建议一起迁移

这些表关系到支付、订阅和财务历史的一致性。

1. `subscription_orders`
2. `subscription_pre_consume_records`
3. `top_ups`
4. `redemptions`

### 可选历史表

这些表用于保留用户历史记录、管理台数据和任务数据。

1. `logs`
2. `quota_data`
3. `midjourneys`
4. `tasks`
5. `checkins`

## 依赖表

这些表不完全属于用户私有数据，但如果你要导入用户关联数据，它们必须先存在，而且 `id` 必须兼容。

1. `subscription_plans`
2. `custom_oauth_providers`

如果目标 PostgreSQL 库里这两张表已经有数据，必须先确认它们和 MySQL 源库中的 ID 是否一致，否则用户关联数据不能直接导入。

## 推荐迁移顺序

1. 启动一次 PostgreSQL 目标实例，让应用自动执行 `AutoMigrate` 建表。
2. 停止源库写入，或至少冻结本次要迁移的用户。
3. 先迁依赖/引用表：
   - `subscription_plans`
   - `custom_oauth_providers`
4. 再迁用户主表：
   - `users`
5. 再迁登录和安全相关表：
   - `passkey_credentials`
   - `two_fas`
   - `two_fa_backup_codes`
   - `user_oauth_bindings`
6. 再迁 Token 表：
   - `tokens`
7. 再迁订阅和支付相关表：
   - `user_subscriptions`
   - `subscription_pre_consume_records`
   - `subscription_orders`
   - `top_ups`
   - `redemptions`
8. 最后按需迁历史表：
   - `checkins`
   - `midjourneys`
   - `tasks`
   - `logs`
   - `quota_data`

## 关键注意事项

### 1. 必须保留 `users.id`

几乎所有用户相关表都是通过 `user_id` 关联到 `users`。  
如果目标库中的 `users.id` 变了，所有关联表都会错位。

### 2. 这些表建议保留原始 `id`

导入时，至少建议保留下列表的原始主键：

- `users`
- `tokens`
- `top_ups`
- `subscription_orders`
- `user_subscriptions`
- `subscription_pre_consume_records`
- `tasks`
- `logs`
- `midjourneys`
- `checkins`

### 3. 导入后要重置 PostgreSQL sequence

如果你手工插入了带显式 `id` 的数据，而没有重置 sequence，后续新增数据时很容易报主键冲突。

### 4. PostgreSQL 中的保留字列名

本项目里有一些字段名在 PostgreSQL 中写 SQL 时需要特别注意，例如：

- `"group"`
- `"key"`

在 PostgreSQL 手写 SQL 时，记得用双引号包起来。

### 5. 布尔值转换

MySQL 中很多布尔值通常表现为 `0/1`，而 PostgreSQL 使用 `false/true`。

迁移时重点关注这些字段：

- `enabled`
- `is_used`
- `is_enabled`
- `clone_warning`
- `user_present`
- `user_verified`
- `backup_eligible`
- `backup_state`
- `unlimited_quota`
- `model_limits_enabled`
- `cross_group_retry`
- `is_stream`

### 6. `logs` 可能不在主库

如果部署里设置了 `LOG_SQL_DSN`，那么 `logs` 可能存储在单独的日志数据库，而不是主数据库。迁移前先确认部署配置。

## 部分用户迁移策略

如果你不是全量迁移，而是只迁一部分用户，建议先在 MySQL 中准备一个待迁移用户 ID 清单。

示例：

```sql
CREATE TEMPORARY TABLE migration_user_ids (
  id INT PRIMARY KEY
);

INSERT INTO migration_user_ids (id) VALUES
  (101),
  (205),
  (309);
```

后续所有用户关联表，都按这张临时表来筛选。

## MySQL 导出 SQL 示例

### 用户核心表

```sql
SELECT * FROM users
WHERE id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM tokens
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM passkey_credentials
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM two_fas
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM two_fa_backup_codes
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM user_oauth_bindings
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

### 订阅 / 支付相关表

```sql
SELECT * FROM user_subscriptions
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM subscription_orders
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM top_ups
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM subscription_pre_consume_records
WHERE user_id IN (SELECT id FROM migration_user_ids)
   OR user_subscription_id IN (
     SELECT id
     FROM user_subscriptions
     WHERE user_id IN (SELECT id FROM migration_user_ids)
   );
```

```sql
SELECT * FROM redemptions
WHERE user_id IN (SELECT id FROM migration_user_ids)
   OR used_user_id IN (SELECT id FROM migration_user_ids);
```

### 可选历史表

```sql
SELECT * FROM checkins
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM midjourneys
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM tasks
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM logs
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

```sql
SELECT * FROM quota_data
WHERE user_id IN (SELECT id FROM migration_user_ids);
```

## 依赖表检查 SQL

在导入用户关联表之前，先确认依赖表的 ID 是否兼容。

### 订阅计划表

```sql
SELECT id, title, price_amount, duration_unit, duration_value
FROM subscription_plans
ORDER BY id;
```

### 自定义 OAuth Provider 表

```sql
SELECT id, name, slug, enabled
FROM custom_oauth_providers
ORDER BY id;
```

如果源库和目标库的 ID 不一致，下面这些表不能直接导：

- `user_subscriptions`
- `subscription_orders`
- `subscription_pre_consume_records`
- `user_oauth_bindings`

## PostgreSQL 导入注意事项

### 1. 按顺序导入

请按本文推荐的顺序导入，不要先导子表再导父表。

### 2. 保留显式 ID

对所有存在关联关系的表，导入时不要让 PostgreSQL 自动分配新 ID。

### 3. 导入前检查唯一字段冲突

重点检查这些唯一字段是否和目标库已有数据冲突：

- `users.username`
- `users.email`
- `users.access_token`
- `tokens.key`
- `top_ups.trade_no`
- `subscription_orders.trade_no`
- `passkey_credentials.credential_id`
- `custom_oauth_providers.slug`

## PostgreSQL Sequence 重置示例

导入完成后，执行类似下面的 SQL：

```sql
SELECT setval(
  pg_get_serial_sequence('users', 'id'),
  COALESCE((SELECT MAX(id) FROM users), 1),
  true
);
```

建议对这些带整数主键的表都执行一次同样模式的 sequence 重置：

- `users`
- `tokens`
- `passkey_credentials`
- `two_fas`
- `two_fa_backup_codes`
- `user_oauth_bindings`
- `subscription_plans`
- `subscription_orders`
- `user_subscriptions`
- `subscription_pre_consume_records`
- `top_ups`
- `redemptions`
- `checkins`
- `midjourneys`
- `tasks`
- `logs`
- `quota_data`
- `models`
- `vendors`
- `prefill_groups`
- `custom_oauth_providers`
- `channels`

## 迁移后校验清单

### 账号一致性

1. `users` 的迁移数量是否和预期一致。
2. 每个用户的以下字段是否一致：
   - `id`
   - `username`
   - `email`
   - `group`
   - `quota`
   - `used_quota`
   - `status`

### Token 一致性

1. 每个用户的 Token 数量是否一致。
2. Token 状态是否一致。
3. 抽样测试几个 Token，确认可以正常鉴权。

### 安全信息一致性

1. Passkey 记录是否存在。
2. 2FA 记录是否存在。
3. 备用码数量是否合理。
4. OAuth 绑定中的 `provider_id` 是否都能在 `custom_oauth_providers` 中找到。

### 订阅 / 支付一致性

1. `user_subscriptions.plan_id` 是否都存在于 `subscription_plans`。
2. `subscription_pre_consume_records.user_subscription_id` 是否都能在 `user_subscriptions` 中找到。
3. `subscription_orders.user_id` 是否都能对应到用户。
4. `top_ups.user_id` 是否都能对应到用户。

### 历史数据一致性

1. `logs` 每个用户的数量是否基本一致。
2. `tasks` 每个用户的数量是否一致。
3. `midjourneys` 每个用户的数量是否一致。
4. `quota_data` 是否存在预期数据。

### 新增数据安全性

1. 在 PostgreSQL 中新建一个测试用户。
2. 为该用户创建一个新 Token。
3. 确认不会报 sequence 或主键冲突错误。

## 建议先做一次小规模演练

正式迁移前，建议先在测试环境做一轮小规模试迁：

1. 选几个真实用户。
2. 迁到 PostgreSQL 测试实例。
3. 验证登录、Token、订阅展示、后台查看是否都正常。
4. 确认无误后再执行正式迁移。

## 实操建议

如果你要做的是：

- 全量实例迁移：更适合用成熟迁移工具或按表全量导出导入。
- 部分用户迁移：更适合按表筛选导出，且必须保留 ID。

对这个项目来说，部分用户迁移通常比全量迁移更容易出错，因为关联关系分散在多个字段上：

- `user_id`
- `used_user_id`
- `plan_id`
- `provider_id`
- `user_subscription_id`

所以正式操作前，建议先把“待迁移用户清单”和“依赖表 ID 映射”确认好。
