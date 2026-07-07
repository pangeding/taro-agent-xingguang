package agent

import (
	"fmt"
	"strings"

	"backend-go/internal/db"
)

const SYSTEM_PROMPT = "你是一位专业、富有洞察力的塔罗牌解读师，擅长结合牌面含义和用户具体问题提供深入、个性化的解读。你的解读充满智慧、共情和启发性。"

func BuildCardPrompt(card *db.TarotCard, isReversed bool, question string, position int, spreadType string) string {
	positionDesc := ""
	if spreadType == "three" {
		positions := []string{"过去", "现在", "未来"}
		if position < len(positions) {
			positionDesc = fmt.Sprintf("这张牌在牌阵中代表%s。", positions[position])
		}
	}

	reversalDesc := "逆位"
	if !isReversed {
		reversalDesc = "正位"
	}
	meaning := card.MeaningReversed
	if !isReversed {
		meaning = card.MeaningUpright
	}

	arcanaLabel := "大阿尔卡纳"
	if card.ArcanaType != "major" {
		arcanaLabel = "小阿尔卡纳"
	}

	suitLine := ""
	if card.Suit != nil {
		suitLine = fmt.Sprintf("- 花色：%s", *card.Suit)
	}

	numberLine := ""
	if card.Number != nil {
		numberLine = fmt.Sprintf("- 数字：%d", *card.Number)
	}

	prompt := fmt.Sprintf(`你是一位专业的塔罗牌解读师，请为以下抽牌结果提供深入、富有洞察力的解读：

## 用户问题
%s

## 抽牌结果
- 牌名：%s
- 牌型：%s（%s）
- 方位：%s
%s
%s
- 关键词：%s
%s

## 牌面基本含义
%s

## 解读要求
1. 结合用户问题分析这张牌的含义
2. 解释这张牌在当前问题中的象征意义
3. 提供具体的建议或启示
4. 语言亲切、富有共情力
5. 避免过于笼统的描述，要具体化
6. 字数在200-300字左右

请开始你的解读：`,
		question, card.Name, card.ArcanaType, arcanaLabel, reversalDesc,
		suitLine, numberLine, card.Keywords, positionDesc, meaning,
	)

	return strings.TrimSpace(prompt)
}

func BuildSynthesisPrompt(interps []CardInterpretation, question string) string {
	var cardsText strings.Builder
	for i, interp := range interps {
		posLabel := "未来"
		if i == 0 {
			posLabel = "过去"
		} else if i == 1 {
			posLabel = "现在"
		}
		reversalDesc := "逆位"
		if !interp.IsReversed {
			reversalDesc = "正位"
		}
		cardsText.WriteString(fmt.Sprintf("- 位置%d（%s）: %s（%s）\n  解读: %s\n",
			i+1, posLabel, interp.CardName, reversalDesc, interp.Interpretation))
	}

	prompt := fmt.Sprintf(`你是一位专业的塔罗牌解读师。以下是三张牌的独立解读：

## 用户问题
%s

## 各牌解读
%s

## 综合解读要求
1. 将三张牌串联为一个完整的时间线叙事
2. 分析牌与牌之间的能量流转和关联
3. 给出整体性的建议
4. 字数在300-500字左右`,
		question, cardsText.String())

	return strings.TrimSpace(prompt)
}
