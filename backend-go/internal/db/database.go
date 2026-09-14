package db

import (
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init 按驱动名连接数据库。driver 为 "mysql" 或 "sqlite"。
func Init(driver, dsn string) *gorm.DB {
	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		// ReadingCard 的关联 tag 为非标准写法，MySQL 下避免 AutoMigrate 建外键失败
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("failed to connect %s database (dsn=%s): %v", driver, maskDSN(dsn), err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		if driver == "mysql" {
			sqlDB.SetMaxOpenConns(20)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(time.Hour)
		} else {
			sqlDB.Exec("PRAGMA journal_mode=WAL;")
		}
	}
	return db
}

// maskDSN 隐藏 DSN 中的账号密码，避免日志泄露。
func maskDSN(dsn string) string {
	at := strings.Index(dsn, "@")
	colon := strings.Index(dsn, ":")
	if at > 0 && colon > 0 && colon < at {
		return "***" + dsn[at:]
	}
	return dsn
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&TarotCard{}, &Reading{}, &ReadingCard{}, &Feedback{}, &Conversation{}, &Message{})
}
