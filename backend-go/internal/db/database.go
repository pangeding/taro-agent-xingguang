package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(dbURL string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect database")
	}
	sqlDB, _ := db.DB()
	sqlDB.Exec("PRAGMA journal_mode=WAL;")
	return db
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&TarotCard{}, &Reading{}, &ReadingCard{}, &Feedback{}, &Conversation{}, &Message{})
}
