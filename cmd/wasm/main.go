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

	// ゲームオーバー情報を追加でねじ込む（簡易実装）
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
	// JSに返すための簡易構造体
	type CellData struct {
		State  string `json:"state"` // "hidden", "opened", "flagged"
		Count  int    `json:"count"`
		IsMine bool   `json:"is_mine"`
	}

	h := currentBoard.Height
	w := currentBoard.Width
	grid := make([][]CellData, h)

	for y := 0; y < h; y++ {
		grid[y] = make([]CellData, w)
		for x := 0; x < w; x++ {
			c := currentBoard.Cells[y][x]
			d := CellData{}

			if c.IsRevealed {
				d.State = "opened"
				d.IsMine = c.IsMine
				d.Count = c.NeighborCount
			} else if c.IsFlagged {
				d.State = "flagged"
			} else {
				d.State = "hidden"
			}
			grid[y][x] = d
		}
	}

	bytes, _ := json.Marshal(grid)
	return string(bytes)
}

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
