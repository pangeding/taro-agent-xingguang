# 2026-09-14 技术文档：SQLite 迁移 MySQL（保留双驱动）

## 0. 目标

将 `backend-go` 默认数据库从 SQLite 切换为 MySQL，同时保留 SQLite 作为可回退驱动；并全量迁移现有 SQLite 历史数据。

## 1. 环境与连接

- MySQL：Windows 主机，WSL2 通过网关访问。
  - host: `172.23.96.1`（WSL2 默认网关；重启后可能变化，用 `ip route show default` 复查）
  - port: `3306`
  - database: `tarot`（已创建，`utf8mb4` / `utf8mb4_unicode_ci`）
  - user: `root`
  - password: 见 `backend-go/.env` 的 `MYSQL_PASSWORD`（禁止写入本文档或任何 git 跟踪文件）
- MySQL 版本：8.0.27。

### 1.1 WSL 侧连接注意事项
- Windows MySQL 需 `bind-address=0.0.0.0` 且防火墙放行 3306。
- WSL2 NAT 模式下 Windows 主机 IP = 默认网关：`ip route show default | awk '{print $3}'`。

## 2. 配置约定（internal/config）

新增环境变量（`backend-go/.env`）：

```
# 驱动选择：mysql | sqlite | 空(自动)
DB_DRIVER=              # 留空时：MYSQL_HOST 非空 => mysql，否则 sqlite

DATABASE_URL=data/tarot.db   # sqlite 路径；若以 mysql:// 开头则直接作为 MySQL DSN

MYSQL_HOST=172.23.96.1
MYSQL_PORT=3306
MYSQL_DB=tarot
MYSQL_USER=root
MYSQL_PASSWORD=***
```

驱动判定顺序（`Settings.EffectiveDB()`）：
1. `DATABASE_URL` 以 `mysql://` 开头 => mysql，DSN 用其转换；
2. `DB_DRIVER=mysql` 或（`DB_DRIVER` 为空且 `MYSQL_HOST` 非空）=> mysql，DSN 由 `MYSQL_*` 构造；
3. 其余 => sqlite，DSN 为 `DATABASE_URL`（默认 `data/tarot.db`）。

MySQL DSN 格式：
```
user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=True&loc=Local
```
（go-sql-driver 用最后一个 `@` 定位，密码含 `@` 无需转义。）

## 3. 代码改动

### 3.1 依赖（go.mod）
```
gorm.io/driver/mysql
github.com/joho/godotenv
```

### 3.2 internal/config/config.go
- 新增字段：`DBDriver`、`MySQLHost`、`MySQLPort`、`MySQLDB`、`MySQLUser`、`MySQLPassword`。
- `Load()` 首行调用 `godotenv.Load()`（忽略文件不存在错误），使 server/脚本自动加载 `.env`（同时修复当前 `DATABASE_URL`、`DASHSCOPE_API_KEY` 带引号不生效问题）。
- 新增方法：
  - `EffectiveDB() (driver, dsn string)`
  - `MySQLDSN() string`

### 3.3 internal/db/database.go
- `Init(driver, dsn string) *gorm.DB`：
  - `mysql`：`gorm.Open(mysql.Open(dsn), cfg)`，`cfg.DisableForeignKeyConstraintWhenMigrating=true`，连接池 `SetMaxOpenConns(20)`、`SetMaxIdleConns(10)`、`SetConnMaxLifetime(time.Hour)`。
  - `sqlite`：`gorm.Open(sqlite.Open(dsn), cfg)`，保留 `PRAGMA journal_mode=WAL;`。
- 连接失败 `log.Fatalf`（含 driver/dsn，但不含密码明文——打印前对 DSN 脱敏）。

### 3.4 internal/service/card_service.go
- `GetRandomCard()` 改为 Go 层随机：
  - `Pluck("id", &ids)` 取全部 ID；
  - `math/rand` 选 1 个，`First(&card, id)`；
  - 正逆位用 Go 随机。
- 删除 `Order("RANDOM()")`、`Raw("SELECT ABS(RANDOM()) % 2")`，消除 SQLite/MySQL 方言差异。

### 3.5 cmd/server/main.go
- 用 `driver, dsn := cfg.EffectiveDB()`，`db.Init(driver, dsn)`。

### 3.6 scripts/import_cards/main.go
- 删除自实现 `loadEnv`，改用 `config.Load()` + `cfg.EffectiveDB()`。
- 打印目标驱动与库（不打印密码）。

### 3.7 scripts/migrate_sqlite_to_mysql/main.go（新增）
- 源：SQLite 文件（默认 `data/tarot.db`，可用 `SQLITE_PATH` 覆盖）。
- 目标：由 `config.Load()` 解析出的 MySQL。
- 流程：
  1. 打开源 SQLite（只读）与目标 MySQL；
  2. 目标 `AutoMigrate`；
  3. 按依赖顺序迁移，**显式保留原主键 ID**：
     `tarot_cards → readings → reading_cards → conversations → messages → feedbacks`；
  4. 使用 `clause.OnConflict{DoNothing: true}` 保证幂等；
  5. 每表打印源/目标计数以便核对。
- 注意：先迁移父表再子表；`reading_cards` 依赖 `readings` 与 `tarot_cards`。

## 4. 执行步骤

```bash
cd backend-go
go get gorm.io/driver/mysql github.com/joho/godotenv
go mod tidy
go build ./...

# 1) 建表 + 导入 22 张牌到 MySQL
go run scripts/import_cards/main.go

# 2) 全量迁移 SQLite 历史数据到 MySQL
go run scripts/migrate_sqlite_to_mysql/main.go

# 3) 启动服务
go run cmd/server/main.go
```

## 5. 验证

```bash
# 编译
cd backend-go && go build ./...

# MySQL 表与计数
MYSQL_PWD="$MYSQL_PASSWORD" mysql -h 172.23.96.1 -P 3306 -uroot tarot \
  -e "show tables; select 'tarot_cards',count(*) from tarot_cards union all select 'readings',count(*) from readings union all select 'reading_cards',count(*) from reading_cards union all select 'conversations',count(*) from conversations union all select 'messages',count(*) from messages union all select 'feedbacks',count(*) from feedbacks;"

# 冒烟
curl http://localhost:8000/health
curl http://localhost:8000/api/v1/cards/random/
curl -X POST http://localhost:8000/api/v1/readings/langgraph -H "Content-Type: application/json" \
  -d '{"question":"我最近运气怎么样？","spread_type":"single"}'
```

## 6. 回退

将 `.env` 的 `DB_DRIVER=sqlite`（或清空 `MYSQL_HOST`）即可回到 SQLite，无需改代码。

## 7. 风险与注意

- `172.23.96.1` 随 WSL 重启可能变化；若失败先复查网关 IP。
- MySQL 8 `caching_sha2_password`：`go-sql-driver/mysql` v1.7+ 原生支持。
- `ReadingCard.Card` 关联 tag 为 `foreignKey:ID;references:CardID`，MySQL 下 `AutoMigrate` 会尝试建外键，故禁用外键约束迁移；关联查询本身不依赖该约束。
- 迁移脚本保留原 ID，避免外键/引用错乱；重复执行幂等。
