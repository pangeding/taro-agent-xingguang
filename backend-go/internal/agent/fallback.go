package agent

import (
	"fmt"

	"backend-go/internal/db"
)

func GetBasicInterpretation(card *db.TarotCard, isReversed bool) string {
	meaning := card.MeaningReversed
	if !isReversed {
		meaning = card.MeaningUpright
	}
	reversalDesc := "逆位"
	if !isReversed {
		reversalDesc = "正位"
	}

	return fmt.Sprintf(`【%s - %s】

基础含义：%s

关键词：%s

（注：AI深度解读暂时不可用，这是牌面的基础含义。请结合你的具体问题思考这张牌对你的启示。）`,
		card.Name, reversalDesc, meaning, card.Keywords)
}
