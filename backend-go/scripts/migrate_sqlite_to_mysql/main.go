package main

import (
	"fmt"
	"log"
	"os"

	"backend-go/internal/config"
	"backend-go/internal/db"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 将 SQLite 数据全量迁移到 MySQL（目标由 .env 的 MYSQL_* / DB_DRIVER 决定）。
// 保留原主键 ID，幂等可重复执行。
func main() {
	cfg := config.Load()
	driver, dsn := cfg.EffectiveDB()
	if driver != "mysql" {
		log.Fatal("目标库不是 mysql，请检查 .env 的 MYSQL_* 或 DB_DRIVER=mysql")
	}

	dst := db.Init(driver, dsn)
	db.AutoMigrate(dst)

	srcPath := os.Getenv("SQLITE_PATH")
	if srcPath == "" {
		srcPath = "data/tarot.db"
	}
	src, err := gorm.Open(sqlite.Open(srcPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("打开源 SQLite 失败 (%s): %v", srcPath, err)
	}

	fmt.Printf("迁移源: SQLite %s -> 目标: MySQL\n", srcPath)

	// 按依赖顺序迁移，先父表后子表
	migrateTable[db.TarotCard](src, dst, "tarot_cards")
	migrateTable[db.Reading](src, dst, "readings")
	migrateTable[db.ReadingCard](src, dst, "reading_cards")
	migrateTable[db.Conversation](src, dst, "conversations")
	migrateTable[db.Message](src, dst, "messages")
	migrateTable[db.Feedback](src, dst, "feedbacks")

	fmt.Println("迁移完成！")
}

func migrateTable[T any](src, dst *gorm.DB, name string) {
	var rows []T
	if err := src.Table(name).Find(&rows).Error; err != nil {
		log.Fatalf("读取 %s 失败: %v", name, err)
	}
	if len(rows) > 0 {
		if err := dst.Clauses(clause.OnConflict{DoNothing: true}).
			Omit(clause.Associations).
			CreateInBatches(rows, 100).Error; err != nil {
			log.Fatalf("写入 %s 失败: %v", name, err)
		}
	}

	var srcCount, dstCount int64
	src.Table(name).Count(&srcCount)
	dst.Table(name).Count(&dstCount)
	fmt.Printf("  [%-15s] 源=%d 目标=%d\n", name, srcCount, dstCount)
}
