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

type GameSession struct {
	board *game.Board
}

var session = &GameSession{}

// NewGame: autoOpen引数を追加
func (s *GameSession) NewGame(width, height, mineCount int, autoOpen bool) string {
	s.board = game.NewBoard(width, height, mineCount)

	// 最初にランダムで1マス開けるオプション
	if autoOpen {
		// Solverのランダムロジックを借用して安全な場所（未開封）を一つ選ぶ
		bot := solver.New(s.board)
		if move := bot.NextMove(); move != nil {
			s.board.Open(move.X, move.Y)
		}
	}

	return viewmodel.NewGameView(s.board)
}

func (s *GameSession) Open(x, y int) string {
	if s.board == nil {
		return "{}"
	}
	s.board.Open(x, y)
	return viewmodel.NewGameView(s.board)
}

func (s *GameSession) ToggleFlag(x, y int) string {
	if s.board == nil {
		return "{}"
	}
	s.board.ToggleFlag(x, y)
	return viewmodel.NewGameView(s.board)
}

func (s *GameSession) BotStep() string {
	if s.board == nil || s.board.CheckClear() {
		return "{}"
	}
	bot := solver.New(s.board)
	if move := bot.NextMove(); move != nil {
		if move.Type == solver.MoveOpen {
			s.board.Open(move.X, move.Y)
		} else {
			s.board.ToggleFlag(move.X, move.Y)
		}
	}
	return viewmodel.NewGameView(s.board)
}

// --- ベンチマーク機能（高速周回） ---
func runBenchmarkWrapper(this js.Value, args []js.Value) interface{} {
	width := args[0].Int()
	height := args[1].Int()
	mines := args[2].Int()
	runs := args[3].Int()

	wins := 0
	start := time.Now()

	for i := 0; i < runs; i++ {
		// 画面更新なしでゲームを作成
		b := game.NewBoard(width, height, mines)
		bot := solver.New(b)

		// ゲーム終了までループ
		for {
			// クリアかゲームオーバーで終了
			if b.CheckClear() {
				wins++
				break
			}
			// 地雷を踏んでいたら終了（CheckClearでは判定できない敗北状態）
			// ※Solverは地雷を踏むとNextMoveを返さないわけではないが、
			//   Board.Openがfalseを返した時点でループを抜ける必要がある

			move := bot.NextMove()
			if move == nil {
				break // 手詰まり（通常ありえない）
			}

			if move.Type == solver.MoveOpen {
				safe := b.Open(move.X, move.Y)
				if !safe {
					break // 爆発
				}
			} else {
				b.ToggleFlag(move.X, move.Y)
			}
		}
	}

	duration := time.Since(start)

	// 結果を文字列で返す
	return fmt.Sprintf("Benchmark Result:\nRuns: %d\nWins: %d (%.1f%%)\nTime: %v\nSpeed: %.0f games/sec",
		runs, wins, float64(wins)/float64(runs)*100, duration, float64(runs)/duration.Seconds())
}

// --- Wrapper ---

func newGameWrapper(this js.Value, args []js.Value) interface{} {
	w, h, m := 10, 10, 10
	autoOpen := false

	if len(args) >= 3 {
		w = args[0].Int()
		h = args[1].Int()
		m = args[2].Int()
	}
	if len(args) >= 4 {
		autoOpen = args[3].Bool()
	}

	return session.NewGame(w, h, m, autoOpen)
}

func openCellWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	return session.Open(args[0].Int(), args[1].Int())
}

func toggleFlagWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	return session.ToggleFlag(args[0].Int(), args[1].Int())
}

func botStepWrapper(this js.Value, args []js.Value) interface{} {
	return session.BotStep()
}

func main() {
	c := make(chan struct{})

	js.Global().Set("goNewGame", js.FuncOf(newGameWrapper))
	js.Global().Set("goOpenCell", js.FuncOf(openCellWrapper))
	js.Global().Set("goToggleFlag", js.FuncOf(toggleFlagWrapper))
	js.Global().Set("goBotStep", js.FuncOf(botStepWrapper))

	// 新機能
	js.Global().Set("goRunBenchmark", js.FuncOf(runBenchmarkWrapper))

	println("Go WebAssembly Initialized (Benchmark Ready)")
	<-c
}
