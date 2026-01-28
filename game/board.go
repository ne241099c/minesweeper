package game

import (
	"fmt"
	"math/rand"
	"time"
)

// NewBoard は指定されたサイズと地雷数で盤面を初期化して返します
func NewBoard(width, height, mineCount int) *Board {
	cells := make([][]Cell, height)
	for y := 0; y < height; y++ {
		cells[y] = make([]Cell, width)
	}

	board := &Board{
		Width:  width,
		Height: height,
		Cells:  cells,
	}

	board.placeMines(mineCount)
	board.calculateNeighbors()

	return board
}

// placeMines は地雷をランダムに配置します
func (b *Board) placeMines(count int) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	minesPlaced := 0

	for minesPlaced < count {
		x := rand.Intn(b.Width)
		y := rand.Intn(b.Height)

		if !b.Cells[y][x].IsMine {
			b.Cells[y][x].IsMine = true
			minesPlaced++
		}
	}
}

// calculateNeighbors は全マスの NeighborCount を計算します
func (b *Board) calculateNeighbors() {
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.Cells[y][x].IsMine {
				continue
			}
			count := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					ny := y + dy
					nx := x + dx
					if nx >= 0 && nx < b.Width && ny >= 0 && ny < b.Height {
						if b.Cells[ny][nx].IsMine {
							count++
						}
					}
				}
			}
			b.Cells[y][x].NeighborCount = count
		}
	}
}

// DebugPrint は現在の盤面をコンソールに表示します
func (b *Board) DebugPrint() {
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			cell := b.Cells[y][x]
			if cell.IsMine {
				fmt.Print("* ")
			} else {
				fmt.Printf("%d ", cell.NeighborCount)
			}
		}
		fmt.Println()
	}
}
