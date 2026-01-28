//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"minesweeper/game"
)

var currentBoard *game.Board

// JavaScriptから呼ばれる関数: New Game
func newGame(this js.Value, args []js.Value) interface{} {
	currentBoard = game.NewBoard(10, 10, 10)
	return getBoardState()
}

// JavaScriptから呼ばれる関数: Open
func openCell(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	x := args[0].Int()
	y := args[1].Int()

	isSafe := currentBoard.Open(x, y)

	jsonStr := getBoardState()

	// ゲームオーバー情報を追加でねじ込む
	// 本来は構造体を分けてJSON化すべきですが、今回は文字列操作で簡易対応
	if !isSafe {
		// JS側で判定しやすいようにフラグを仕込む等の処理が必要ですが、
		// 今回は盤面データに含まれる IsMine で判定させます
	}

	return jsonStr
}

// JavaScriptから呼ばれる関数: Flag
func toggleFlag(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	x := args[0].Int()
	y := args[1].Int()

	currentBoard.ToggleFlag(x, y)
	return getBoardState()
}

// 盤面状態をJSON文字列として返す
func getBoardState() string {
	type CellData struct {
		State  string `json:"state"`
		Count  int    `json:"count"`
		IsMine bool   `json:"is_mine"`
	}

	// フロントエンドに送る全体のデータ構造
	type GameState struct {
		Cells          [][]CellData `json:"cells"`
		MinesRemaining int          `json:"mines_remaining"` // 残り地雷数（表示用）
		IsGameOver     bool         `json:"is_game_over"`
		IsGameClear    bool         `json:"is_game_clear"`
	}

	h := currentBoard.Height
	w := currentBoard.Width

	// クリア判定
	isClear := currentBoard.CheckClear()
	// フラグの計算
	flagCount := currentBoard.GetFlagCount()

	grid := make([][]CellData, h)

	// ゲームオーバー判定用フラグ（ループ内で検出）
	isGameOver := false

	for y := 0; y < h; y++ {
		grid[y] = make([]CellData, w)
		for x := 0; x < w; x++ {
			c := currentBoard.Cells[y][x]
			d := CellData{}

			if c.IsRevealed {
				d.State = "opened"
				d.IsMine = c.IsMine
				d.Count = c.NeighborCount
				// もし開いたマスが地雷だったらゲームオーバー
				if c.IsMine {
					isGameOver = true
				}
			} else if c.IsFlagged {
				d.State = "flagged"
			} else {
				d.State = "hidden"
			}

			// クリア時は全ての地雷にフラグを立てたような見た目にする（任意）
			if isClear && c.IsMine {
				d.State = "flagged"
			}

			grid[y][x] = d
		}
	}

	// ゲームオーバー時は全地雷をオープンにする処理
	if isGameOver {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if currentBoard.Cells[y][x].IsMine {
					grid[y][x].State = "opened"
					grid[y][x].IsMine = true
				}
			}
		}
	}

	state := GameState{
		Cells:          grid,
		MinesRemaining: currentBoard.MineCount - flagCount, // 全地雷 - フラグ数
		IsGameOver:     isGameOver,
		IsGameClear:    isClear,
	}

	bytes, _ := json.Marshal(state)
	return string(bytes)
}

// (後略: main関数などはそのままでOK)

func main() {
	c := make(chan struct{})

	// JavaScriptのグローバル関数としてGoの関数を登録
	js.Global().Set("goNewGame", js.FuncOf(newGame))
	js.Global().Set("goOpenCell", js.FuncOf(openCell))
	js.Global().Set("goToggleFlag", js.FuncOf(toggleFlag))

	println("Go WebAssembly Initialized")

	// プログラムが終了しないように待機
	<-c
}
