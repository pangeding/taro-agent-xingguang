package agent

import (
	"context"
	"strings"

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

func interpretCards(ctx context.Context, llm *ChatClient, state *ReadingState) (*ReadingState, error) {
	for _, ci := range state.CardsInfo {
		prompt := BuildCardPrompt(ci.Card, ci.IsReversed, state.Question, ci.Position, state.SpreadType)
		text, err := llm.Chat(ctx, SYSTEM_PROMPT, prompt)
		if err != nil {
			state.Interpretations = append(state.Interpretations, CardInterpretation{
				CardID:         ci.ID,
				CardName:       ci.Name,
				IsReversed:     ci.IsReversed,
				Position:       ci.Position,
				Interpretation: GetBasicInterpretation(ci.Card, ci.IsReversed),
				Status:         "fallback",
			})
		} else {
			state.Interpretations = append(state.Interpretations, CardInterpretation{
				CardID:         ci.ID,
				CardName:       ci.Name,
				IsReversed:     ci.IsReversed,
				Position:       ci.Position,
				Interpretation: strings.TrimSpace(text),
				Status:         "success",
			})
		}
	}
	return state, nil
}

func synthesize(ctx context.Context, llm *ChatClient, state *ReadingState) (*ReadingState, error) {
	prompt := BuildSynthesisPrompt(state.Interpretations, state.Question)
	text, err := llm.Chat(ctx, SYSTEM_PROMPT, prompt)
	if err != nil {
		state.Synthesis = "（综合分析暂时不可用）"
		state.Status = "partial"
		state.Error = err.Error()
	} else {
		state.Synthesis = strings.TrimSpace(text)
	}
	return state, nil
}
