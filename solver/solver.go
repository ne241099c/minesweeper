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
	Strategy   string  // "Logic", "AI", "Random"
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
	// 1. 論理的に「絶対に安全」
	if move := s.findSafeMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Logic"
		move.Confidence = 1.0
		return move
	}

	// 2. 論理的に「絶対に地雷」
	if move := s.findFlagMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Logic"
		move.Confidence = 1.0
		return move
	}

	// 3. AI または ランダム
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

func (s *Solver) findRandomMove() *Move {
	// AIが使える場合
	if s.AiNet != nil {
		bestProb := 1.0 // 地雷確率（低いほうが良い）
		var bestMove *Move

		for y := 0; y < s.Board.Height; y++ {
			for x := 0; x < s.Board.Width; x++ {
				c := s.Board.Cells[y][x]
				if !c.IsRevealed && !c.IsFlagged {
					input := s.createAiInput(x, y)
					prob := s.AiNet.Predict(input)

					// より安全なマスが見つかったら更新
					if prob < bestProb {
						bestProb = prob
						bestMove = &Move{
							X: x, Y: y,
							Type:       MoveOpen,
							Strategy:   "AI",
							Confidence: 1.0 - prob, // 安全確率
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
