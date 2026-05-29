# 数据库查询问题：表名不匹配

## 问题描述

使用 sqlite3 直接查询数据库时，使用 `taro` 或 `TarotCard` 作为表名查询失败：

```sql
SELECT count(0) FROM taro;          -- Error: no such table: taro
SELECT count(0) FROM TarotCard;     -- Error: no such table: TarotCard
SELECT count(0) FROM taro.TarotCard; -- Error: no such table: taro.TarotCard
```

## 原因分析

项目使用 Peewee ORM，模型类名与实际数据库表名不一致：

| 模型类名 | 实际表名 | 定义位置 |
|---------|---------|---------|
| `TarotCard` | `tarot_cards` | `backend/app/db/models.py:42` |
| `Reading` | `readings` | `backend/app/db/models.py:57` |
| `ReadingCard` | `reading_cards` | `backend/app/db/models.py:71` |
| `Feedback` | `feedbacks` | `backend/app/db/models.py:84` |

表名通过 `class Meta: table_name = "xxx"` 显式指定，采用复数形式、小写下划线命名。

## 正确查询方式

### 基础查询

```sql
-- 查询各表数据量
SELECT count(*) FROM tarot_cards;   -- 22 条
SELECT count(*) FROM readings;      -- 7 条
SELECT count(*) FROM reading_cards; -- 11 条
SELECT count(*) FROM feedbacks;     -- 0 条

-- 查看表结构
.schema tarot_cards
.schema readings
.schema reading_cards
.schema feedbacks
```

### tarot_cards 表查询

```sql
-- 字段: id, name, arcana_type, suit, number, meaning_upright, meaning_reversed, keywords, element, zodiac_sign, image_url, description

-- 查看所有大阿尔卡纳牌
SELECT id, name, number, keywords FROM tarot_cards WHERE arcana_type = 'major';

-- 查看某张牌的详细信息
SELECT name, meaning_upright, meaning_reversed FROM tarot_cards WHERE name = '愚人';

-- 按花色查询小阿尔卡纳牌
SELECT name, suit, number FROM tarot_cards WHERE arcana_type = 'minor' AND suit = 'wands';

-- 查看牌的数量统计
SELECT arcana_type, count(*) FROM tarot_cards GROUP BY arcana_type;
```

### readings 表查询

```sql
-- 字段: id, session_id, question, spread_type, created_at

-- 查看所有占卜记录
SELECT id, question, spread_type, created_at FROM readings ORDER BY created_at DESC;

-- 按牌阵类型统计
SELECT spread_type, count(*) FROM readings GROUP BY spread_type;

-- 查询某天的占卜记录
SELECT * FROM readings WHERE date(created_at) = '2026-04-27';

-- 查询某个会话的占卜记录
SELECT * FROM readings WHERE session_id = '9abd865e-dfa5-4f5b-91a2-fe7f73fab378';
```

### reading_cards 表查询

```sql
-- 字段: id, reading_id, card_id, position, is_reversed, interpretation

-- 查看某次占卜抽到的牌
SELECT rc.position, tc.name, rc.is_reversed
FROM reading_cards rc
JOIN tarot_cards tc ON rc.card_id = tc.id
WHERE rc.reading_id = 1
ORDER BY rc.position;

-- 查看逆位牌的记录
SELECT rc.id, tc.name, rc.is_reversed
FROM reading_cards rc
JOIN tarot_cards tc ON rc.card_id = tc.id
WHERE rc.is_reversed = 1;

-- 统计各牌出现次数
SELECT tc.name, count(*) as times
FROM reading_cards rc
JOIN tarot_cards tc ON rc.card_id = tc.id
GROUP BY tc.name
ORDER BY times DESC;
```

### 多表关联查询

```sql
-- 查看占卜记录及对应的牌
SELECT r.question, r.created_at, tc.name, rc.position, rc.is_reversed
FROM readings r
JOIN reading_cards rc ON r.id = rc.reading_id
JOIN tarot_cards tc ON rc.card_id = tc.id
ORDER BY r.created_at DESC;

-- 查看某个问题抽到的所有牌及解读
SELECT tc.name, rc.is_reversed, rc.interpretation
FROM readings r
JOIN reading_cards rc ON r.id = rc.reading_id
JOIN tarot_cards tc ON rc.card_id = tc.id
WHERE r.question LIKE '%工作运势%'
ORDER BY rc.position;
```

### feedbacks 表查询

```sql
-- 字段: id, reading_id, rating, comment, created_at

-- 查看反馈记录（当前为空）
SELECT * FROM feedbacks;

-- 按评分统计
SELECT rating, count(*) FROM feedbacks GROUP BY rating;

-- 查看某次占卜的反馈
SELECT f.rating, f.comment, r.question
FROM feedbacks f
JOIN readings r ON f.reading_id = r.id;
```

## 数据库文件路径

- 路径：`backend/app/db/taro.db`
- 配置文件：`backend/.env` 中 `DATABASE_URL="sqlite:///app/db/taro.db"`
- 初始化脚本：`backend/scripts/init_db.py`

## 相关代码

- 数据库连接：`backend/app/db/base.py`
- 模型定义：`backend/app/db/models.py`
- 表创建函数：`backend/app/db/models.py:88` `create_tables()`
