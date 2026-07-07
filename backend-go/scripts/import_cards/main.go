package main

import (
	"encoding/json"
	"fmt"
	"os"

	"backend-go/internal/db"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	d, err := gorm.Open(sqlite.Open("data/taro.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, _ := d.DB()
	sqlDB.Exec("PRAGMA journal_mode=WAL;")

	d.AutoMigrate(&db.TarotCard{})

	data, err := os.ReadFile("data/tarot_cards.json")
	if err != nil {
		panic(err)
	}

	var cards []db.TarotCard
	if err := json.Unmarshal(data, &cards); err != nil {
		panic(err)
	}

	for _, card := range cards {
		var existing db.TarotCard
		d.Where("name = ?", card.Name).First(&existing)
		if existing.ID == 0 {
			d.Create(&card)
		}
	}

	fmt.Println("导入完成")
}
