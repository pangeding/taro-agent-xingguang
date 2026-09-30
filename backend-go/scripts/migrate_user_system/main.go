// 用户系统数据迁移脚本。
//
// 幂等，可重复执行。支持 DRY_RUN 预览。
//
//	DRY_RUN=1 go run scripts/migrate_user_system/main.go              # 只打印不写
//	go run scripts/migrate_user_system/main.go                        # 执行
//	go run scripts/migrate_user_system/main.go --drop-legacy          # 附加删除老列 user_id
//
// 目标：把历史数据全部归入引导管理员账号下，并为 conversations / readings / feedbacks
// 补上 owner_id 归属列。详见 doc/2026-09-28-用户系统与数据隔离技术文档.md §5。
//
// 重要：本脚本必须在部署带 Auth 中间件的新版后端之前跑完。
// 否则新代码按 owner_id 过滤，而列还没回填，所有用户会看到 0 条历史。
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"backend-go/internal/bootstrap"
	"backend-go/internal/config"
	"backend-go/internal/db"

	"gorm.io/gorm"
)

// convSessionRe 匹配 readings.session_id 中由会话流程写入的 "conv_<id>" 形式。
//
// 必须锚定首尾（^...$）：若只写 LIKE 'conv\_%'，
// 后面 CAST 到整数时会把 'conv_abc' 之类的值截断成 0，产生脏行。
var convSessionRe = regexp.MustCompile(`^conv_(\d+)$`)

// baseline 是迁移前的行数快照，用于迁移后对账。
var baseline = map[string]int64{
	"conversations": 0,
	"messages":      0,
	"readings":      0,
	"reading_cards": 0,
	"feedbacks":     0,
}

type column struct {
	table    string
	name     string
	ddl      string // ALTER TABLE ... ADD COLUMN ...
	index    string // 索引名，空表示不建
	indexDDL string
}

func main() {
	dropLegacy := flag.Bool("drop-legacy", false, "迁移校验通过后删除 conversations.user_id 老列")
	flag.Parse()

	dryRun := os.Getenv("DRY_RUN") == "1"

	cfg := config.Load()
	driver, dsn := cfg.EffectiveDB()

	fmt.Printf("数据库驱动: %s\n", driver)
	if dryRun {
		fmt.Println("*** DRY_RUN=1，只打印不执行 ***")
	}
	fmt.Println()

	d := db.Init(driver, dsn)

	if driver != "mysql" {
		// SQLite 是回退库。data/tarot.db 实测 conversations 为 0 行，无业务数据需要迁移，
		// 因此只建新表就退出（与 scripts/migrate_sqlite_to_mysql 的处理方式一致）。
		fmt.Println("非 mysql 驱动：仅创建 users / sessions 表，跳过历史数据迁移。")
		if dryRun {
			fmt.Println("[dry-run] AutoMigrate(&User{}, &Session{})")
			return
		}
		if err := d.AutoMigrate(&db.User{}, &db.Session{}); err != nil {
			log.Fatalf("建表失败: %v", err)
		}
		fmt.Println("完成。")
		return
	}

	snapshotBaseline(d)

	// —— S1 建新表 ——
	step("S1 创建 users / sessions 表")
	if dryRun {
		fmt.Println("  [dry-run] AutoMigrate(&User{}, &Session{})")
	} else if err := d.AutoMigrate(&db.User{}, &db.Session{}); err != nil {
		log.Fatalf("S1 建表失败: %v", err)
	}

	// —— S2 确保引导管理员存在 ——
	step("S2 确保引导管理员存在")
	var admin db.User
	if dryRun {
		fmt.Println("  [dry-run] bootstrap.Users(db, cfg)")
		// DRY_RUN 下仍然真跑一次查询以便后续计算 adminID；但表可能还不存在。
		if err := d.Where("role = ?", "admin").Order("id ASC").First(&admin).Error; err != nil {
			fmt.Println("  [dry-run] users 表暂无 admin，实际执行时将由 bootstrap 创建")
		}
	} else {
		created, err := bootstrap.Users(d, cfg)
		if err != nil {
			log.Fatalf("S2 引导账号失败: %v", err)
		}
		if created != nil {
			fmt.Printf("  已创建引导管理员: %s (id=%d)\n", created.Username, created.ID)
		}
		if err := d.Where("role = ?", "admin").Order("id ASC").First(&admin).Error; err != nil {
			log.Fatalf("S2 未找到任何管理员账号: %v", err)
		}
		fmt.Printf("  归属目标账号: %s (id=%d)\n", admin.Username, admin.ID)
	}

	// —— S3 加列 ——
	step("S3 新增归属列 owner_id / conversation_id")
	cols := []column{
		{
			table: "conversations", name: "owner_id",
			ddl:      "ALTER TABLE `conversations` ADD COLUMN `owner_id` bigint unsigned NOT NULL DEFAULT 0",
			index:    "idx_conversations_owner_id",
			indexDDL: "CREATE INDEX `idx_conversations_owner_id` ON `conversations`(`owner_id`)",
		},
		{
			table: "readings", name: "owner_id",
			ddl:      "ALTER TABLE `readings` ADD COLUMN `owner_id` bigint unsigned NOT NULL DEFAULT 0",
			index:    "idx_readings_owner_id",
			indexDDL: "CREATE INDEX `idx_readings_owner_id` ON `readings`(`owner_id`)",
		},
		{
			table: "readings", name: "conversation_id",
			ddl:      "ALTER TABLE `readings` ADD COLUMN `conversation_id` bigint unsigned NULL",
			index:    "idx_readings_conversation_id",
			indexDDL: "CREATE INDEX `idx_readings_conversation_id` ON `readings`(`conversation_id`)",
		},
		{
			table: "feedbacks", name: "owner_id",
			ddl:      "ALTER TABLE `feedbacks` ADD COLUMN `owner_id` bigint unsigned NOT NULL DEFAULT 0",
			index:    "idx_feedbacks_owner_id",
			indexDDL: "CREATE INDEX `idx_feedbacks_owner_id` ON `feedbacks`(`owner_id`)",
		},
	}
	for _, c := range cols {
		applyColumn(d, c, dryRun)
	}

	if dryRun {
		fmt.Println()
		fmt.Println("DRY_RUN 结束。以上为将要执行的语句。")
		return
	}

	// —— S4 回填 conversations ——
	step("S4 回填 conversations.owner_id -> admin")
	res := d.Exec("UPDATE conversations SET owner_id = ? WHERE owner_id = 0", admin.ID)
	if res.Error != nil {
		log.Fatalf("S4 失败: %v", res.Error)
	}
	fmt.Printf("  更新 %d 行\n", res.RowsAffected)

	// —— S5/S6 回填 readings ——
	// 刻意用 Go 侧解析而不是 SQL JOIN：conversations 是 utf8mb4_0900_ai_ci、
	// readings.session_id 是 utf8mb4_unicode_ci，直接 JOIN 会报
	// ERROR 1267 Illegal mix of collations。Go 侧解析同时也不挑方言。
	step("S5/S6 回填 readings.conversation_id 与 owner_id")
	backfillReadings(d, admin.ID)

	// —— S7 回填 feedbacks ——
	step("S7 回填 feedbacks.owner_id")
	backfillFeedbacks(d, admin.ID)

	// —— S8 校验 ——
	step("S8 校验")
	verify(d, admin.ID)

	// —— S9 删老列 ——
	if *dropLegacy {
		step("S9 删除 conversations.user_id 老列")
		if columnExists(d, "conversations", "user_id") {
			if err := d.Exec("ALTER TABLE `conversations` DROP COLUMN `user_id`").Error; err != nil {
				log.Fatalf("S9 失败: %v", err)
			}
			fmt.Println("  已删除 conversations.user_id")
		} else {
			fmt.Println("  conversations.user_id 不存在，跳过")
		}
	} else {
		step("S9 删除老列（未启用）")
		fmt.Println("  保留 conversations.user_id。确认无误后可用 --drop-legacy 删除。")
	}

	fmt.Println()
	fmt.Println("迁移完成。")
}

// ————————————————————————————— 辅助 —————————————————————————————

func step(title string) {
	fmt.Printf("\n== %s ==\n", title)
}

func snapshotBaseline(d *gorm.DB) {
	fmt.Println("迁移前行数快照:")
	for t := range baseline {
		var n int64
		d.Table(t).Count(&n)
		baseline[t] = n
		fmt.Printf("  %-15s %d\n", t, n)
	}
}

func columnExists(d *gorm.DB, table, col string) bool {
	var n int64
	d.Raw(`SELECT COUNT(*) FROM information_schema.columns
	       WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
		table, col).Scan(&n)
	return n > 0
}

func indexExists(d *gorm.DB, table, idx string) bool {
	var n int64
	d.Raw(`SELECT COUNT(*) FROM information_schema.statistics
	       WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`,
		table, idx).Scan(&n)
	return n > 0
}

func applyColumn(d *gorm.DB, c column, dryRun bool) {
	if columnExists(d, c.table, c.name) {
		fmt.Printf("  [跳过] %s.%s 已存在\n", c.table, c.name)
	} else if dryRun {
		fmt.Printf("  [dry-run] %s\n", c.ddl)
	} else {
		if err := d.Exec(c.ddl).Error; err != nil {
			log.Fatalf("加列失败 (%s.%s): %v", c.table, c.name, err)
		}
		fmt.Printf("  [新增] %s.%s\n", c.table, c.name)
	}

	if c.index == "" {
		return
	}
	if indexExists(d, c.table, c.index) {
		fmt.Printf("  [跳过] 索引 %s 已存在\n", c.index)
	} else if dryRun {
		fmt.Printf("  [dry-run] %s\n", c.indexDDL)
	} else {
		if err := d.Exec(c.indexDDL).Error; err != nil {
			log.Fatalf("建索引失败 (%s): %v", c.index, err)
		}
		fmt.Printf("  [新增] 索引 %s\n", c.index)
	}
}

type readingRow struct {
	ID             uint
	SessionID      string
	OwnerID        uint
	ConversationID *uint
}

// backfillReadings 回填 readings 的两列。
//
// readings 表原本完全没有归属列，只能靠 session_id 反推：
//   - 'conv_<id>' 形式 -> 关联到对应会话，归属随该会话
//   - 其余（随机 UUID）-> 无任何归属线索，归 admin
//
// 已知事实：历史 60 行中 45 行是 conv_ 形式且 45/45 全部可解析，15 行为随机 UUID。
func backfillReadings(d *gorm.DB, adminID uint) {
	var convs []struct {
		ID      uint
		OwnerID uint
	}
	if err := d.Table("conversations").Select("id, owner_id").Scan(&convs).Error; err != nil {
		log.Fatalf("读取 conversations 失败: %v", err)
	}
	ownerOfConv := make(map[uint]uint, len(convs))
	for _, c := range convs {
		ownerOfConv[c.ID] = c.OwnerID
	}

	var readings []readingRow
	if err := d.Table("readings").Select("id, session_id, owner_id, conversation_id").Scan(&readings).Error; err != nil {
		log.Fatalf("读取 readings 失败: %v", err)
	}

	var linked, orphan int
	err := d.Transaction(func(tx *gorm.DB) error {
		for i := range readings {
			r := &readings[i]
			convID, owner := resolveOwner(r, ownerOfConv, adminID)

			if r.OwnerID != owner {
				if err := tx.Exec("UPDATE readings SET owner_id = ? WHERE id = ?", owner, r.ID).Error; err != nil {
					return err
				}
			}
			if convID != nil && (r.ConversationID == nil || *r.ConversationID != *convID) {
				if err := tx.Exec("UPDATE readings SET conversation_id = ? WHERE id = ?", *convID, r.ID).Error; err != nil {
					return err
				}
			}
			if convID != nil {
				linked++
			} else {
				orphan++
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("S5/S6 回填失败: %v", err)
	}

	fmt.Printf("  可关联到会话: %d 行\n", linked)
	fmt.Printf("  无归属线索归 admin: %d 行\n", orphan)
}

// resolveOwner 解析一条 reading 的会话关联与归属。
func resolveOwner(r *readingRow, ownerOfConv map[uint]uint, adminID uint) (convID *uint, owner uint) {
	m := convSessionRe.FindStringSubmatch(r.SessionID)
	if m == nil {
		return nil, adminID
	}
	n, err := strconv.ParseUint(m[1], 10, 32)
	if err != nil {
		return nil, adminID
	}
	cid := uint(n)
	o, ok := ownerOfConv[cid]
	if !ok {
		// session_id 指向的会话不存在（脏数据），不强行关联。
		return nil, adminID
	}
	if o == 0 {
		o = adminID
	}
	return &cid, o
}

func backfillFeedbacks(d *gorm.DB, adminID uint) {
	var rows []struct {
		ID        uint
		ReadingID uint
		OwnerID   uint
	}
	if err := d.Table("feedbacks").Select("id, reading_id, owner_id").Scan(&rows).Error; err != nil {
		log.Fatalf("读取 feedbacks 失败: %v", err)
	}
	if len(rows) == 0 {
		fmt.Println("  feedbacks 无数据，跳过（该表当前无任何读写代码）")
		return
	}

	var readings []struct {
		ID      uint
		OwnerID uint
	}
	if err := d.Table("readings").Select("id, owner_id").Scan(&readings).Error; err != nil {
		log.Fatalf("读取 readings 失败: %v", err)
	}
	ownerOfReading := make(map[uint]uint, len(readings))
	for _, r := range readings {
		ownerOfReading[r.ID] = r.OwnerID
	}

	err := d.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			owner, ok := ownerOfReading[r.ReadingID]
			if !ok || owner == 0 {
				owner = adminID
			}
			if r.OwnerID != owner {
				if err := tx.Exec("UPDATE feedbacks SET owner_id = ? WHERE id = ?", owner, r.ID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("S7 回填失败: %v", err)
	}
	fmt.Printf("  处理 %d 行\n", len(rows))
}

// verify（S8）不通过则 log.Fatalf —— 绝不静默继续。
func verify(d *gorm.DB, adminID uint) {
	failed := false

	check := func(desc string, query string, args ...any) {
		var n int64
		if err := d.Raw(query, args...).Scan(&n).Error; err != nil {
			log.Fatalf("  校验查询失败 (%s): %v", desc, err)
		}
		status := "OK"
		if n > 0 {
			status = "FAIL"
			failed = true
		}
		fmt.Printf("  [%s] %s = %d\n", status, desc, n)
	}

	check("conversations 中 owner_id=0 的行数", "SELECT COUNT(*) FROM conversations WHERE owner_id = 0")
	check("readings 中 owner_id=0 的行数", "SELECT COUNT(*) FROM readings WHERE owner_id = 0")
	check("feedbacks 中 owner_id=0 的行数", "SELECT COUNT(*) FROM feedbacks WHERE owner_id = 0")

	// 行数守恒：迁移不得增删任何业务行。
	for table, want := range baseline {
		var got int64
		d.Table(table).Count(&got)
		status := "OK"
		if got != want {
			status = "FAIL"
			failed = true
		}
		fmt.Printf("  [%s] %s 行数守恒: 迁移前 %d / 现在 %d\n", status, table, want, got)
	}

	// 归属集中度：迁移后应只剩 admin 一个归属者。
	var ownerCount int64
	d.Raw("SELECT COUNT(DISTINCT owner_id) FROM readings").Scan(&ownerCount)
	status := "OK"
	if ownerCount != 1 {
		status = "FAIL"
		failed = true
	}
	fmt.Printf("  [%s] readings 的 distinct owner_id = %d（期望 1）\n", status, ownerCount)

	// 会话关联覆盖率（信息性，不作为失败条件）。
	var linked int64
	d.Raw("SELECT COUNT(*) FROM readings WHERE conversation_id IS NOT NULL").Scan(&linked)
	fmt.Printf("  [INFO] readings 已关联会话: %d 行\n", linked)

	if failed {
		fmt.Println()
		log.Fatalf("S8 校验未通过，已中止。数据库处于中间状态，请用 backup/tarot_pre_user_system.sql 恢复后排查。")
	}
	fmt.Println("  全部校验通过。")
	fmt.Printf("  归属账号 id=%d\n", adminID)
}
