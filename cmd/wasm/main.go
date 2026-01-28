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
	mode solver.SolverMode // 現在のBotモード
}

// デフォルトはHybridモード
var session = &GameSession{mode: solver.ModeHybrid}

// モード切替関数 (JSから呼ばれる)
func setSolverModeWrapper(_ js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		modeStr := args[0].String()
		if modeStr == "pure" {
			session.mode = solver.ModePureAI
			return "Switched to Pure AI Mode"
		}
	}
	session.mode = solver.ModeHybrid
	return "Switched to Hybrid Mode"
}

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
	// モードを指定してSolverを作成
	bot := solver.New(s.board, s.mode)

	var move *solver.Move
	if move = bot.NextMove(); move != nil {
		// 戦略ごとの統計カウント
		switch move.Strategy {
		case "Logic", "Advanced", "Tank":
			s.stats.Logic++
		case "AI", "PureAI", "Tank(Prob)": // AI関連
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

	var callback js.Value
	if len(args) >= 5 && args[4].Type() == js.TypeFunction {
		callback = args[4]
	}

	wins := 0
	start := time.Now()

	// 現在のセッションモードを使用
	benchMode := session.mode

	for i := 0; i < runs; i++ {
		b := game.NewBoard(width, height, mines)
		bot := solver.New(b, benchMode)

		logicCnt, aiCnt, randomCnt := 0, 0, 0
		var lastMove *solver.Move
		isWin := false

		for {
			if b.CheckClear() {
				wins++
				isWin = true
				break
			}
			move := bot.NextMove()
			if move == nil {
				break
			}
			lastMove = move

			switch move.Strategy {
			case "Logic", "Advanced", "Tank":
				logicCnt++
			case "AI", "PureAI", "Tank(Prob)":
				aiCnt++
			case "Random":
				randomCnt++
			}

			if move.Type == solver.MoveOpen {
				if !b.Open(move.X, move.Y) {
					break
				}
			} else {
				b.ToggleFlag(move.X, move.Y)
			}
		}

		if callback.Type() == js.TypeFunction {
			resStr := "💥 OVER "
			if isWin {
				resStr = "🎉 CLEAR"
			}

			lastStrat := "-"
			lastConf := 0.0
			if lastMove != nil {
				lastStrat = lastMove.Strategy
				lastConf = lastMove.Confidence * 100
			}

			logMsg := fmt.Sprintf("[%03d/%d] %s (L:%d, A:%d, R:%d) Last: %s(%.1f%%)",
				i+1, runs, resStr, logicCnt, aiCnt, randomCnt, lastStrat, lastConf)

			callback.Invoke(logMsg)
		}
	}

	duration := time.Since(start)
	modeName := "Hybrid"
	if benchMode == solver.ModePureAI {
		modeName = "Pure AI"
	}

	return fmt.Sprintf("Benchmark Finished (%s):\nRuns: %d, Wins: %d (%.1f%%)\nTime: %v\nSpeed: %.0f games/sec",
		modeName, runs, wins, float64(wins)/float64(runs)*100, duration, float64(runs)/duration.Seconds())
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
	// 新規追加
	js.Global().Set("goSetSolverMode", js.FuncOf(setSolverModeWrapper))

	println("Go WebAssembly Initialized")
	<-c
}
