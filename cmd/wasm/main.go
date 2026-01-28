//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"
	"time"

	"minesweeper/game"
	"minesweeper/solver"
	"minesweeper/viewmodel"
)

// GameSession はゲームの状態と統計情報を管理します
type GameSession struct {
	board *game.Board
	stats struct {
		Logic  int
		AI     int
		Random int
	}
}

var session = &GameSession{}

// NewGame: ゲームと統計をリセットします
func (s *GameSession) NewGame(width, height, mineCount int) string {
	s.board = game.NewBoard(width, height, mineCount)

	// 統計リセット
	s.stats.Logic = 0
	s.stats.AI = 0
	s.stats.Random = 0

	return viewmodel.NewGameView(s.board, "")
}

func (s *GameSession) Open(x, y int) string {
	if s.board == nil {
		return "{}"
	}
	s.board.Open(x, y)
	return viewmodel.NewGameView(s.board, "")
}

func (s *GameSession) ToggleFlag(x, y int) string {
	if s.board == nil {
		return "{}"
	}
	s.board.ToggleFlag(x, y)
	return viewmodel.NewGameView(s.board, "")
}

// BotStep: Botに1手進めさせ、統計を取ります
func (s *GameSession) BotStep() string {
	if s.board == nil || s.board.CheckClear() {
		return "{}"
	}
	bot := solver.New(s.board)

	var move *solver.Move
	// AI学習対応のNextMoveを呼び出す
	if move = bot.NextMove(); move != nil {
		// 戦略ごとの統計カウント
		switch move.Strategy {
		case "Logic":
			s.stats.Logic++
		case "AI":
			s.stats.AI++
		case "Random":
			s.stats.Random++
		}

		// 行動実行
		if move.Type == solver.MoveOpen {
			s.board.Open(move.X, move.Y)
		} else {
			s.board.ToggleFlag(move.X, move.Y)
		}
	}

	// レポート作成
	report := ""

	// ゲームオーバー判定
	// (直前のOpenで地雷を踏んだかチェック)
	isGameOver := false
	if move != nil && move.Type == solver.MoveOpen {
		// 範囲内チェック
		if move.Y >= 0 && move.Y < s.board.Height && move.X >= 0 && move.X < s.board.Width {
			if s.board.Cells[move.Y][move.X].IsMine && s.board.Cells[move.Y][move.X].IsRevealed {
				isGameOver = true
			}
		}
	}

	if isGameOver {
		report = fmt.Sprintf("💥 GAME OVER\n----------------\nLogic : %d\nAI    : %d\nRandom: %d\n\nLast Move: %s (Confidence: %.1f%%)",
			s.stats.Logic, s.stats.AI, s.stats.Random, move.Strategy, move.Confidence*100)
	} else if s.board.CheckClear() {
		report = fmt.Sprintf("🎉 GAME CLEAR\n----------------\nLogic : %d\nAI    : %d\nRandom: %d",
			s.stats.Logic, s.stats.AI, s.stats.Random)
	}

	return viewmodel.NewGameView(s.board, report)
}

// --- ベンチマーク機能 ---

func runBenchmarkWrapper(_ js.Value, args []js.Value) interface{} {
	width := args[0].Int()
	height := args[1].Int()
	mines := args[2].Int()
	runs := args[3].Int()

	wins := 0
	start := time.Now()

	for i := 0; i < runs; i++ {
		b := game.NewBoard(width, height, mines)
		bot := solver.New(b)

		for {
			if b.CheckClear() {
				wins++
				break
			}
			move := bot.NextMove()
			if move == nil {
				break
			}

			if move.Type == solver.MoveOpen {
				if !b.Open(move.X, move.Y) {
					break
				}
			} else {
				b.ToggleFlag(move.X, move.Y)
			}
		}
	}

	duration := time.Since(start)
	return fmt.Sprintf("Benchmark Result:\nRuns: %d\nWins: %d (%.1f%%)\nTime: %v\nSpeed: %.0f games/sec",
		runs, wins, float64(wins)/float64(runs)*100, duration, float64(runs)/duration.Seconds())
}

// --- Wrapper Functions ---

func newGameWrapper(_ js.Value, args []js.Value) interface{} {
	w, h, m := 10, 10, 10
	if len(args) >= 3 {
		w = args[0].Int()
		h = args[1].Int()
		m = args[2].Int()
	}
	return session.NewGame(w, h, m)
}

func openCellWrapper(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	return session.Open(args[0].Int(), args[1].Int())
}

func toggleFlagWrapper(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	return session.ToggleFlag(args[0].Int(), args[1].Int())
}

func botStepWrapper(_ js.Value, args []js.Value) interface{} {
	return session.BotStep()
}

func main() {
	c := make(chan struct{})

	js.Global().Set("goNewGame", js.FuncOf(newGameWrapper))
	js.Global().Set("goOpenCell", js.FuncOf(openCellWrapper))
	js.Global().Set("goToggleFlag", js.FuncOf(toggleFlagWrapper))
	js.Global().Set("goBotStep", js.FuncOf(botStepWrapper))
	js.Global().Set("goRunBenchmark", js.FuncOf(runBenchmarkWrapper))

	println("Go WebAssembly Initialized")
	<-c
}
