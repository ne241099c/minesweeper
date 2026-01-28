package solver

import (
	"fmt"
	"math/rand"
	"minesweeper/ai"
	"minesweeper/game"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type MoveType int

const (
	MoveOpen MoveType = iota
	MoveFlag
)

type Move struct {
	X, Y       int
	Type       MoveType
	IsGuess    bool    // 運任せかどうか
	Strategy   string  // "Logic", "Advanced", "AI", "Random"
	Confidence float64 // 0.0 ~ 1.0 (安全確率)
}

type Solver struct {
	Board *game.Board
	AiNet *ai.Network
}

func New(b *game.Board) *Solver {
	net, err := ai.NewNetwork(game.GetWeightsJSON())
	if err != nil {
		fmt.Println("AI Load Error:", err)
	}
	return &Solver{Board: b, AiNet: net}
}

func (s *Solver) NextMove() *Move {
	if move := s.findSafeMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Logic"
		move.Confidence = 1.0
		return move
	}

	if move := s.findFlagMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Logic"
		move.Confidence = 1.0
		return move
	}

	if move := s.findAdvancedMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Advanced" // レポートで見分けられるように
		move.Confidence = 1.0
		return move
	}

	if move := s.findTankMove(); move != nil {
		// 確率100%ならGuessではない、それ以外はGuess扱いだが「高精度なGuess」
		if move.Confidence == 1.0 {
			move.IsGuess = false
		} else {
			move.IsGuess = true
		}
		return move
	}

	move := s.findRandomMove()
	if move != nil {
		move.IsGuess = true
	}
	return move
}

// ... findSafeMove, findFlagMove は変更なし ...
func (s *Solver) findSafeMove() *Move {
	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			cell := s.Board.Cells[y][x]
			if !cell.IsRevealed || cell.NeighborCount == 0 {
				continue
			}
			_, flags, hidden := s.getNeighborsInfo(x, y)
			if flags == cell.NeighborCount && len(hidden) > 0 {
				target := hidden[0]
				return &Move{X: target.x, Y: target.y, Type: MoveOpen}
			}
		}
	}
	return nil
}

func (s *Solver) findFlagMove() *Move {
	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			cell := s.Board.Cells[y][x]
			if !cell.IsRevealed || cell.NeighborCount == 0 {
				continue
			}
			totalHidden, flags, hidden := s.getNeighborsInfo(x, y)
			if totalHidden == cell.NeighborCount && (totalHidden-flags) > 0 {
				for _, p := range hidden {
					if !s.Board.Cells[p.y][p.x].IsFlagged {
						return &Move{X: p.x, Y: p.y, Type: MoveFlag}
					}
				}
			}
		}
	}
	return nil
}

// ★追加: 集合論を使った発展ロジック
func (s *Solver) findAdvancedMove() *Move {
	// 全ての開いている数字マスを走査
	for y1 := 0; y1 < s.Board.Height; y1++ {
		for x1 := 0; x1 < s.Board.Width; x1++ {
			c1 := s.Board.Cells[y1][x1]
			if !c1.IsRevealed || c1.NeighborCount == 0 {
				continue
			}

			// c1の未確定情報
			_, f1, h1 := s.getNeighborsInfo(x1, y1)
			needed1 := c1.NeighborCount - f1
			if len(h1) == 0 {
				continue
			}

			// c1の周囲にある、別の数字マスc2を探す
			// (c1の隣接マスだけでなく、c1の隣接未開封マスを共有している可能性のある範囲を見るべきだが、
			//  簡易的に「ボード全体」または「c1の近傍」を探索する。
			//  ここでは計算量を抑えるため「c1から距離2以内」などを探索するのが一般的だが、
			//  実装を簡単にするため「c1の隣接未開封マスを共有している数字マス」を探すアプローチをとる)

			// アプローチ: 全探索は重いので、h1（c1の空きマス）のどれかに隣接している数字マスを探す
			checkedNeighbors := make(map[int]bool) // 重複チェック用 (y*width + x)

			for _, emptyPos := range h1 {
				// emptyPosの周囲を調べる
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := emptyPos.x+dx, emptyPos.y+dy
						if nx < 0 || nx >= s.Board.Width || ny < 0 || ny >= s.Board.Height {
							continue
						}
						if nx == x1 && ny == y1 {
							continue
						} // 自分自身はスキップ

						key := ny*s.Board.Width + nx
						if checkedNeighbors[key] {
							continue
						}
						checkedNeighbors[key] = true

						c2 := s.Board.Cells[ny][nx]
						if !c2.IsRevealed || c2.NeighborCount == 0 {
							continue
						}

						// c2が見つかった。c1とc2の関係を調べる。
						_, f2, h2 := s.getNeighborsInfo(nx, ny)
						needed2 := c2.NeighborCount - f2

						// 判定: h1 が h2 の部分集合か？ (h1 ⊆ h2)
						if isSubset(h1, h2) {
							// 差分を計算 (diff = h2 - h1)
							diff := getDifference(h2, h1)
							if len(diff) == 0 {
								continue
							}

							minesInDiff := needed2 - needed1

							if minesInDiff == 0 {
								// 差分はすべて「安全」
								target := diff[0]
								return &Move{X: target.x, Y: target.y, Type: MoveOpen}
							} else if minesInDiff == len(diff) {
								// 差分はすべて「地雷」
								target := diff[0]
								// まだ旗が立っていないものを返す
								if !s.Board.Cells[target.y][target.x].IsFlagged {
									return &Move{X: target.x, Y: target.y, Type: MoveFlag}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// ヘルパー: subsetがsupersetに含まれているか
func isSubset(subset, superset []pos) bool {
	if len(subset) > len(superset) {
		return false
	}
	for _, p1 := range subset {
		found := false
		for _, p2 := range superset {
			if p1.x == p2.x && p1.y == p2.y {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// ヘルパー: superset - subset の差分を返す
func getDifference(superset, subset []pos) []pos {
	diff := []pos{}
	for _, p2 := range superset {
		isShared := false
		for _, p1 := range subset {
			if p1.x == p2.x && p1.y == p2.y {
				isShared = true
				break
			}
		}
		if !isShared {
			diff = append(diff, p2)
		}
	}
	return diff
}

// ... findRandomMove, findPureRandomMove は変更なし ...
func (s *Solver) findRandomMove() *Move {
	if s.AiNet != nil {
		bestProb := 1.0
		var bestMove *Move

		for y := 0; y < s.Board.Height; y++ {
			for x := 0; x < s.Board.Width; x++ {
				c := s.Board.Cells[y][x]
				if !c.IsRevealed && !c.IsFlagged {
					input := s.createAiInput(x, y)
					prob := s.AiNet.Predict(input)

					if prob < bestProb {
						bestProb = prob
						bestMove = &Move{
							X: x, Y: y,
							Type:       MoveOpen,
							Strategy:   "AI",
							Confidence: 1.0 - prob,
						}
					}
				}
			}
		}
		if bestMove != nil {
			return bestMove
		}
	}
	return s.findPureRandomMove()
}

func (s *Solver) findPureRandomMove() *Move {
	type point struct{ x, y int }
	candidates := []point{}

	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			c := s.Board.Cells[y][x]
			if !c.IsRevealed && !c.IsFlagged {
				candidates = append(candidates, point{x, y})
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}
	choice := candidates[rand.Intn(len(candidates))]
	return &Move{
		X: choice.x, Y: choice.y,
		Type:       MoveOpen,
		Strategy:   "Random",
		Confidence: 0.0,
	}
}

func (s *Solver) createAiInput(tx, ty int) []float64 {
	input := make([]float64, 25)
	idx := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := tx+dx, ty+dy
			val := 9.0
			if nx >= 0 && nx < s.Board.Width && ny >= 0 && ny < s.Board.Height {
				cell := s.Board.Cells[ny][nx]
				if !cell.IsRevealed {
					if cell.IsFlagged {
						val = -2.0
					} else {
						val = -1.0
					}
				} else {
					val = float64(cell.NeighborCount)
				}
			}
			input[idx] = val
			idx++
		}
	}
	return input
}

type pos struct{ x, y int }

func (s *Solver) getNeighborsInfo(cx, cy int) (totalHidden int, flags int, hiddenList []pos) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := cx+dx, cy+dy
			if nx >= 0 && nx < s.Board.Width && ny >= 0 && ny < s.Board.Height {
				neighbor := s.Board.Cells[ny][nx]
				if !neighbor.IsRevealed {
					totalHidden++
					if neighbor.IsFlagged {
						flags++
					} else {
						hiddenList = append(hiddenList, pos{nx, ny})
					}
				}
			}
		}
	}
	return
}

func (s *Solver) findTankMove() *Move {
	tank := NewTankSolver(s.Board)
	return tank.Solve()
}
