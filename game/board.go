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
// 未開封のマスは「-」、地雷は「*」、数字は数字を表示します
func (b *Board) DebugPrint() {
	fmt.Print("   ")
	for x := 0; x < b.Width; x++ {
		fmt.Printf("%d ", x) // 列番号を表示
	}
	fmt.Println()

	for y := 0; y < b.Height; y++ {
		fmt.Printf("%d: ", y) // 行番号を表示
		for x := 0; x < b.Width; x++ {
			cell := b.Cells[y][x]

			if cell.IsRevealed {
				if cell.IsMine {
					fmt.Print("* ") // 踏んでしまった地雷
				} else {
					if cell.NeighborCount == 0 {
						fmt.Print(". ") // 0は見やすく「.」にする
					} else {
						fmt.Printf("%d ", cell.NeighborCount)
					}
				}
			} else {
				// 未開封
				fmt.Print("- ") // ■ の代わり
			}
		}
		fmt.Println()
	}
}

// Open は指定された座標のマスを開けます
// 戻り値: ゲーム継続なら true, 地雷を踏んだら false
func (b *Board) Open(x, y int) bool {
	// 1. 範囲外チェック
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return true
	}

	cell := &b.Cells[y][x]

	// 2. すでに開いている、またはフラグがあるなら何もしない
	if cell.IsRevealed || cell.IsFlagged {
		return true
	}

	// 3. 開ける
	cell.IsRevealed = true

	// 4. 地雷判定
	if cell.IsMine {
		return false // ゲームオーバー
	}

	// 5. 0連鎖（Flood Fill）
	// 周囲の地雷数が0の場合、自動的に周囲8マスも開ける
	if cell.NeighborCount == 0 {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				// 自分自身はスキップしなくても、上の「IsRevealedチェック」で弾かれるので大丈夫ですが、
				// 無駄な呼び出しを避けるならスキップしても良いです
				if dx != 0 || dy != 0 {
					b.Open(x+dx, y+dy) // 再帰呼び出し！
				}
			}
		}
	}

	return true
}

// ToggleFlag は指定された座標のフラッグを切り替えます
func (b *Board) ToggleFlag(x, y int) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}
	cell := &b.Cells[y][x]

	// すでに開いているマスにはフラッグを置けない
	if cell.IsRevealed {
		return
	}

	cell.IsFlagged = !cell.IsFlagged
}
