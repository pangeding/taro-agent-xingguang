# 导入脚本 GORM Create 异常问题

## 问题描述

运行 `go run scripts/import_cards/main.go` 导入塔罗牌数据时，"魔术师"牌每次都会重复导入（显示 `[+] 已导入: 魔术师`），而其他牌都能正确跳过（显示 `[=] 跳过 (已存在)`）。

## 根本原因

1. **JSON 数据中自带 ID 字段**：
   - "愚人"：`"id": 0`
   - "魔术师"：`"id": 1`

2. **GORM Create 行为**：
   - 当结构体的主键字段（ID）为 **零值** 时，GORM 执行 INSERT 操作
   - 当结构体的主键字段为 **非零值** 时，GORM 会尝试执行 UPDATE 或产生异常行为

3. **脚本逻辑缺陷**：
   ```go
   // 原代码
   database.Where("name = ?", card.Name).First(&existing)
   if existing.ID == 0 {
       database.Create(&card)  // card.ID = 1，GORM 行为异常
   }
   ```
   - 第一次导入时，数据库中无"魔术师"记录
   - `First` 查询不到记录，`existing.ID` 为 0，进入 `Create`
   - 但 `card.ID = 1`，GORM 执行 Create 时行为不符合预期
   - 实际记录可能未正确插入，或插入方式异常

4. **判断方法不可靠**：
   - `First` 方法在找不到记录时，不会返回错误给调用者（原代码未检查 error）
   - 仅靠 `existing.ID == 0` 判断存在隐患

## 解决方案

在导入前清空 JSON 数据中的 ID 字段，并正确检查查询结果：

```go
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
```

### 修改点

1. **检查 `result.Error`**：使用 GORM 返回的 error 判断记录是否存在，而非检查 ID
2. **重置 `card.ID = 0`**：在 Create 前清空 ID，确保 GORM 执行 INSERT 而非 UPDATE
3. **避免 ID 冲突**：让数据库的 autoIncrement 机制自动分配主键

## 相关文件

- `scripts/import_cards/main.go` - 导入脚本
- `data/tarot_cards.json` - 塔罗牌 JSON 数据
- `internal/db/models.go` - 数据模型定义
