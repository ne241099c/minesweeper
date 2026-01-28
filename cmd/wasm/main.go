//go:build js && wasm

package main

import (
	"syscall/js"

	"minesweeper/game"
	"minesweeper/solver"
	"minesweeper/viewmodel"
)

// GameSession はゲームの状態を保持・管理します
type GameSession struct {
	board *game.Board
}

var session = &GameSession{}

// NewGame は新しいゲームを開始します
func (s *GameSession) NewGame(width, height, mineCount int) string {
	s.board = game.NewBoard(width, height, mineCount)
	return viewmodel.NewGameView(s.board)
}

// Open は指定されたセルを開きます
func (s *GameSession) Open(x, y int) string {
	if s.board == nil {
		return ""
	}
	s.board.Open(x, y)
	return viewmodel.NewGameView(s.board)
}

// ToggleFlag はフラグを切り替えます
func (s *GameSession) ToggleFlag(x, y int) string {
	if s.board == nil {
		return ""
	}
	s.board.ToggleFlag(x, y)
	return viewmodel.NewGameView(s.board)
}

// BotStep はBotに1手進めさせます
func (s *GameSession) BotStep() string {
	if s.board == nil || s.board.CheckClear() {
		return ""
	}

	// Botを初期化して次の手を計算
	bot := solver.New(s.board)
	move := bot.NextMove()

	if move == nil {
		return viewmodel.NewGameView(s.board) // 打つ手なし
	}

	// 行動を実行
	switch move.Type {
	case solver.MoveOpen:
		s.board.Open(move.X, move.Y)
	case solver.MoveFlag:
		s.board.ToggleFlag(move.X, move.Y)
	}

	return viewmodel.NewGameView(s.board)
}

func newGameWrapper(this js.Value, args []js.Value) interface{} {
	// デフォルト値
	w, h, m := 10, 10, 10

	// 引数があれば上書き (JS側から goNewGame(w, h, m) と呼ばれる想定)
	if len(args) >= 3 {
		w = args[0].Int()
		h = args[1].Int()
		m = args[2].Int()
	}

	return session.NewGame(w, h, m)
}

// openCellWrapper, toggleFlagWrapper, botStepWrapper は変更なし
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

	println("Go WebAssembly Initialized (Configurable)")
	<-c
}
