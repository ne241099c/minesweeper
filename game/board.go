package game

import (
	"fmt"
	"math/rand"
	"time"
)

// パッケージ初期化時に一度だけ乱数シードを設定
func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewBoard はメモリ確保だけ行います
func NewBoard(width, height, mineCount int) *Board {
	cells := make([][]Cell, height)
	for y := 0; y < height; y++ {
		cells[y] = make([]Cell, width)
	}

	return &Board{
		Width:         width,
		Height:        height,
		MineCount:     mineCount,
		Cells:         cells,
		IsInitialized: false,
		IsGameOver:    false,
	}
}

// InitializeMines は最初のクリック位置(safeX, safeY)を避けて地雷を配置します
func (b *Board) InitializeMines(safeX, safeY int) {
	// rand.Seedはinitで行っているため削除

	minesPlaced := 0
	for minesPlaced < b.MineCount {
		x := rand.Intn(b.Width)
		y := rand.Intn(b.Height)

		// 既に地雷があるならスキップ
		if b.Cells[y][x].IsMine {
			continue
		}

		// 初回クリック位置の「周囲9マス」には地雷を置かない
		if x >= safeX-1 && x <= safeX+1 && y >= safeY-1 && y <= safeY+1 {
			continue
		}

		b.Cells[y][x].IsMine = true
		minesPlaced++
	}

	b.calculateNeighbors()
	b.IsInitialized = true
}

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

// Open
func (b *Board) Open(x, y int) bool {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return true
	}

	if !b.IsInitialized {
		b.InitializeMines(x, y)
	}

	cell := &b.Cells[y][x]
	if cell.IsRevealed || cell.IsFlagged {
		return true
	}

	cell.IsRevealed = true

	if cell.IsMine {
		b.IsGameOver = true
		return false // ゲームオーバー
	}

	if cell.NeighborCount == 0 {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx != 0 || dy != 0 {
					b.Open(x+dx, y+dy)
				}
			}
		}
	}
	return true
}

func (b *Board) ToggleFlag(x, y int) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}
	cell := &b.Cells[y][x]
	if !cell.IsRevealed {
		cell.IsFlagged = !cell.IsFlagged
	}
}

func (b *Board) DebugPrint() {
	fmt.Println("DebugPrint called")
}

func (b *Board) GetFlagCount() int {
	count := 0
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.Cells[y][x].IsFlagged {
				count++
			}
		}
	}
	return count
}

func (b *Board) CheckClear() bool {
	revealedCount := 0
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.Cells[y][x].IsRevealed {
				revealedCount++
			}
		}
	}
	return (b.Width*b.Height - revealedCount) == b.MineCount
}
