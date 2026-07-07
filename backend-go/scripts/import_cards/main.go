package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"backend-go/internal/db"
)

func loadEnv(path string) map[string]string {
	env := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return env
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")
			env[key] = val
		}
	}
	return env
}

func main() {
	fmt.Println("正在加载 .env 配置...")
	env := loadEnv(".env")

	dbPath := env["DATABASE_URL"]
	if dbPath == "" {
		dbPath = "data/tarot.db"
		fmt.Println("未配置 DATABASE_URL，使用默认路径:", dbPath)
	}

	// 清洗 sqlite:/// 前缀 (兼容 Python 写法)
	dbPath = strings.TrimPrefix(dbPath, "sqlite:///")
	dbPath = strings.TrimPrefix(dbPath, "sqlite://")
	fmt.Println("使用数据库文件:", dbPath)

	// 初始化并迁移数据库
	database := db.Init(dbPath)
	db.AutoMigrate(database)

	// 读取 JSON 数据
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
			// 清空 JSON 中的 ID，让数据库自动分配
			card.ID = 0
			database.Create(&card)
			fmt.Printf("  [+] 已导入: %s\n", card.Name)
		} else {
			fmt.Printf("  [=] 跳过 (已存在): %s\n", card.Name)
		}
	}

	fmt.Println("导入完成！")
}
