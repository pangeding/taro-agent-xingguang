package main

import (
	"encoding/json"
	"fmt"
	"os"

	"backend-go/internal/config"
	"backend-go/internal/db"
)

func main() {
	cfg := config.Load()
	driver, dsn := cfg.EffectiveDB()

	fmt.Printf("使用数据库驱动: %s\n", driver)
	database := db.Init(driver, dsn)
	db.AutoMigrate(database)

	jsonData, err := os.ReadFile("data/tarot_cards.json")
	if err != nil {
		fmt.Println("错误: 未找到 data/tarot_cards.json")
		panic(err)
	}

	var cards []db.TarotCard
	if err := json.Unmarshal(jsonData, &cards); err != nil {
		panic(err)
	}

	fmt.Printf("准备导入 %d 张塔罗牌...\n", len(cards))

	for _, card := range cards {
		var existing db.TarotCard
		result := database.Where("name = ?", card.Name).First(&existing)
		if result.Error != nil {
			card.ID = 0
			database.Create(&card)
			fmt.Printf("  [+] 已导入: %s\n", card.Name)
		} else {
			fmt.Printf("  [=] 跳过 (已存在): %s\n", card.Name)
		}
	}

	fmt.Println("导入完成！")
}
