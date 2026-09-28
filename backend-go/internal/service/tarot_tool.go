package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type ctxKey int

const conversationIDKey ctxKey = iota

// WithConversationID 把会话 ID 注入 context，供抽牌工具确定归属。
func WithConversationID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, conversationIDKey, id)
}

func conversationIDFromContext(ctx context.Context) uint {
	if v, ok := ctx.Value(conversationIDKey).(uint); ok {
		return v
	}
	return 0
}

type DrawTarotCard struct {
	Name           string `json:"name"`
	IsReversed     bool   `json:"is_reversed"`
	Position       int    `json:"position"`
	PositionName   string `json:"position_name,omitempty"`
	Interpretation string `json:"interpretation"`
	ImageURL       string `json:"image_url,omitempty"`
}

type DrawTarotOutput struct {
	SpreadType string          `json:"spread_type"`
	ReadingID  uint            `json:"reading_id"`
	Cards      []DrawTarotCard `json:"cards"`
}

// NewTarotDrawTool 构造 draw_tarot 工具：真实抽取塔罗牌并生成解读。
func NewTarotDrawTool(readingSvc *ReadingService) (tool.InvokableTool, error) {
	return toolutils.InferTool(
		"draw_tarot",
		"抽取真实的塔罗牌并进行解读。当用户想占卜、询问运势、要求抽牌时调用此工具。"+
			"single 表示单张牌（快速指引），three 表示三张牌阵（过去/现在/未来）。"+
			"调用后会返回真实抽到的牌面与解读，请据此作答，不要自行编造牌名。",
		func(ctx context.Context, in struct {
			Question   string `json:"question" jsonschema:"description=用户想占卜的问题，尽量复述用户的原话"`
			SpreadType string `json:"spread_type" jsonschema:"description=牌阵类型，single(单张) 或 three(三张)，默认 single,enum=single,enum=three"`
		}) (DrawTarotOutput, error) {
			spreadType := in.SpreadType
			if spreadType != "three" {
				spreadType = "single"
			}
			if in.Question == "" {
				return DrawTarotOutput{}, fmt.Errorf("question is required")
			}

			sessionID := fmt.Sprintf("conv_%d", conversationIDFromContext(ctx))
			result, err := readingSvc.CreateReadingLangGraph(in.Question, spreadType, &sessionID, nil)
			if err != nil {
				return DrawTarotOutput{}, err
			}

			out := DrawTarotOutput{SpreadType: spreadType, ReadingID: result.ReadingID}
			for _, c := range result.Cards {
				card := DrawTarotCard{
					Name:           c.Name,
					IsReversed:     c.IsReversed,
					Position:       c.Position,
					PositionName:   positionName(c.Position, spreadType),
					Interpretation: c.Interpretation,
				}
				if c.ImageURL != nil {
					card.ImageURL = *c.ImageURL
				}
				out.Cards = append(out.Cards, card)
			}
			return out, nil
		},
	)
}
