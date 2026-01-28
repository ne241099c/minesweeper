package viewmodel

import (
	"encoding/json"
	"minesweeper/game"
)

// CellView, GameView 構造体は変更なし（そのままでOK）
type CellView struct {
	State  string `json:"state"`
	Count  int    `json:"count"`
	IsMine bool   `json:"is_mine"`
}

type GameView struct {
	Cells          [][]CellView `json:"cells"`
	MinesRemaining int          `json:"mines_remaining"`
	IsGameOver     bool         `json:"is_game_over"`
	IsGameClear    bool         `json:"is_game_clear"`
}

// NewGameView は安全にJSONを返します
func NewGameView(b *game.Board) string {
	// 【修正点】nilの場合は空のJSONオブジェクトを返す
	if b == nil {
		return "{}"
	}

	h := b.Height
	w := b.Width

	isClear := b.CheckClear()
	flagCount := b.GetFlagCount()
	isGameOver := false

	grid := make([][]CellView, h)
	for y := 0; y < h; y++ {
		grid[y] = make([]CellView, w)
		for x := 0; x < w; x++ {
			c := b.Cells[y][x]
			v := CellView{}

			if c.IsRevealed {
				v.State = "opened"
				v.IsMine = c.IsMine
				v.Count = c.NeighborCount
				if c.IsMine {
					isGameOver = true
				}
			} else if c.IsFlagged {
				v.State = "flagged"
			} else {
				v.State = "hidden"
			}

			if isClear && c.IsMine {
				v.State = "flagged"
			}
			grid[y][x] = v
		}
	}

	if isGameOver {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if b.Cells[y][x].IsMine {
					grid[y][x].State = "opened"
					grid[y][x].IsMine = true
				}
			}
		}
	}

	view := GameView{
		Cells:          grid,
		MinesRemaining: b.MineCount - flagCount,
		IsGameOver:     isGameOver,
		IsGameClear:    isClear,
	}

	bytes, _ := json.Marshal(view)
	return string(bytes)
}
