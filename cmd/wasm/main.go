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

// NewGame: autoOpen引数を削除
func (s *GameSession) NewGame(width, height, mineCount int) string {
	s.board = game.NewBoard(width, height, mineCount)
	// Botの最初の手はBot自身（solver）がランダムに決めるため、ここでは何もしない
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

// --- ベンチマーク機能 ---
func runBenchmarkWrapper(this js.Value, args []js.Value) interface{} {
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

// --- Wrapper ---

func newGameWrapper(this js.Value, args []js.Value) interface{} {
	w, h, m := 10, 10, 10
	if len(args) >= 3 {
		w = args[0].Int()
		h = args[1].Int()
		m = args[2].Int()
	}
	return session.NewGame(w, h, m)
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
	js.Global().Set("goRunBenchmark", js.FuncOf(runBenchmarkWrapper))

	println("Go WebAssembly Initialized (Random Start Removed)")
	<-c
}
