package agent

import (
	"context"
	"log"
	"strings"
	"sync"

	"backend-go/internal/db"

	"github.com/cloudwego/eino/compose"
)

type ReadingState struct {
	Question        string
	SpreadType      string
	SessionID       string
	CardsInfo       []CardInfo
	Interpretations []CardInterpretation
	Synthesis       string
	Status          string
	Error           string
}

type CardInfo struct {
	ID         uint
	Name       string
	IsReversed bool
	Position   int
	Card       *db.TarotCard
}

type CardInterpretation struct {
	CardID         uint
	CardName       string
	IsReversed     bool
	Position       int
	Interpretation string
	Status         string
}

func CreateReadingGraph(llm *ChatClient) (compose.Runnable[*ReadingState, *ReadingState], error) {
	const (
		nodeInterpret  = "interpret"
		nodeSynthesize = "synthesize"
	)

	graph := compose.NewGraph[*ReadingState, *ReadingState]()

	_ = graph.AddLambdaNode(nodeInterpret, compose.InvokableLambda(
		func(ctx context.Context, state *ReadingState) (*ReadingState, error) {
			return interpretCards(ctx, llm, state)
		},
	))

	_ = graph.AddLambdaNode(nodeSynthesize, compose.InvokableLambda(
		func(ctx context.Context, state *ReadingState) (*ReadingState, error) {
			return synthesize(ctx, llm, state)
		},
	))

	_ = graph.AddEdge(compose.START, nodeInterpret)

	_ = graph.AddBranch(nodeInterpret, compose.NewGraphBranch(
		func(ctx context.Context, state *ReadingState) (string, error) {
			if state.SpreadType == "three" {
				return nodeSynthesize, nil
			}
			return compose.END, nil
		},
		map[string]bool{compose.END: true, nodeSynthesize: true},
	))

	_ = graph.AddEdge(nodeSynthesize, compose.END)

	runner, err := graph.Compile(context.Background())
	return runner, err
}

// interpretCards 并发解读各张牌。
//
// 三张牌阵如果顺序解读，耗时是单张的三倍；并发后总耗时约等于最慢的一张。
// 结果按下标写入预分配的切片，保证 Interpretations 的顺序与 CardsInfo 一致
// （synthesize 与上层的落库循环都依赖这个顺序）。
func interpretCards(ctx context.Context, llm *ChatClient, state *ReadingState) (*ReadingState, error) {
	interps := make([]CardInterpretation, len(state.CardsInfo))
	var wg sync.WaitGroup

	for i, ci := range state.CardsInfo {
		wg.Add(1)
		go func(i int, ci CardInfo) {
			defer wg.Done()

			build := func(text, status string) CardInterpretation {
				return CardInterpretation{
					CardID:         ci.ID,
					CardName:       ci.Name,
					IsReversed:     ci.IsReversed,
					Position:       ci.Position,
					Interpretation: text,
					Status:         status,
				}
			}

			prompt := BuildCardPrompt(ci.Card, ci.IsReversed, state.Question, ci.Position, state.SpreadType)
			text, err := llm.Chat(ctx, SYSTEM_PROMPT, prompt)
			if err != nil {
				log.Printf("LLM interpret failed for %s: %v", ci.Name, err)
				interps[i] = build(GetBasicInterpretation(ci.Card, ci.IsReversed), "fallback")
				return
			}
			interps[i] = build(strings.TrimSpace(text), "success")
		}(i, ci)
	}

	wg.Wait()
	state.Interpretations = interps
	return state, nil
}

func synthesize(ctx context.Context, llm *ChatClient, state *ReadingState) (*ReadingState, error) {
	prompt := BuildSynthesisPrompt(state.Interpretations, state.Question)
	text, err := llm.Chat(ctx, SYSTEM_PROMPT, prompt)
	if err != nil {
		log.Printf("LLM synthesize failed: %v", err)
		state.Synthesis = "（综合分析暂时不可用）"
		state.Status = "partial"
		state.Error = err.Error()
	} else {
		state.Synthesis = strings.TrimSpace(text)
	}
	return state, nil
}
