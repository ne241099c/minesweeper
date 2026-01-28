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
	IsGuess    bool
	Strategy   string
	Confidence float64
}

// SolverMode : ソルバーの動作モード定義
type SolverMode int

const (
	ModeHybrid SolverMode = iota // Logic -> Advanced -> Tank -> AI (最強モード)
	ModePureAI                   // AI Only (実験モード)
)

type Solver struct {
	Board *game.Board
	AiNet *ai.Network
	Mode  SolverMode
}

// New : モードを受け取るように変更
func New(b *game.Board, mode SolverMode) *Solver {
	net, err := ai.NewNetwork(game.GetWeightsJSON())
	if err != nil {
		fmt.Println("AI Load Error:", err)
	}
	return &Solver{Board: b, AiNet: net, Mode: mode}
}

// NextMove : モードに応じて戦略を切り替え
func (s *Solver) NextMove() *Move {
	if s.Mode == ModePureAI {
		return s.nextMovePureAI()
	}
	return s.nextMoveHybrid()
}

// 従来のハイブリッド戦略（最強）
func (s *Solver) nextMoveHybrid() *Move {
	// 1. 基本ロジック: 安全
	if move := s.findSafeMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Logic"
		move.Confidence = 1.0
		return move
	}

	// 2. 基本ロジック: 地雷
	if move := s.findFlagMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Logic"
		move.Confidence = 1.0
		return move
	}

	// 3. 発展ロジック
	if move := s.findAdvancedMove(); move != nil {
		move.IsGuess = false
		move.Strategy = "Advanced"
		move.Confidence = 1.0
		return move
	}

	// 4. タンクソルバー (全探索 & 厳密確率)
	if move := s.findTankMove(); move != nil {
		if move.Confidence == 1.0 {
			move.IsGuess = false
		} else {
			move.IsGuess = true
		}
		return move
	}

	// 5. AI または ランダム
	move := s.findRandomMove()
	if move != nil {
		move.IsGuess = true
	}
	return move
}

// Pure AI戦略（ロジックなし・AIのみ）
func (s *Solver) nextMovePureAI() *Move {
	// AIがロードできていない場合はランダム
	if s.AiNet == nil {
		return s.findPureRandomMove()
	}

	bestProb := 1.0
	var bestMove *Move

	// 全マスをスキャンしてAIに判断させる
	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			c := s.Board.Cells[y][x]
			// 未開封かつフラグなしの場所を評価
			if !c.IsRevealed && !c.IsFlagged {
				input := s.createAiInput(x, y)
				prob := s.AiNet.Predict(input)

				// 最も安全（地雷確率が低い）手を選ぶ
				if prob < bestProb {
					bestProb = prob
					bestMove = &Move{
						X: x, Y: y,
						Type:       MoveOpen,
						Strategy:   "PureAI",
						Confidence: 1.0 - prob,
					}
				}
			}
		}
	}

	if bestMove != nil {
		return bestMove
	}
	return s.findPureRandomMove()
}

// --- 以下、ロジック実装 ---

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

func (s *Solver) findAdvancedMove() *Move {
	for y1 := 0; y1 < s.Board.Height; y1++ {
		for x1 := 0; x1 < s.Board.Width; x1++ {
			c1 := s.Board.Cells[y1][x1]
			if !c1.IsRevealed || c1.NeighborCount == 0 {
				continue
			}
			_, f1, h1 := s.getNeighborsInfo(x1, y1)
			needed1 := c1.NeighborCount - f1
			if len(h1) == 0 {
				continue
			}

			checkedNeighbors := make(map[int]bool)
			for _, emptyPos := range h1 {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := emptyPos.x+dx, emptyPos.y+dy
						if nx < 0 || nx >= s.Board.Width || ny < 0 || ny >= s.Board.Height {
							continue
						}
						if nx == x1 && ny == y1 {
							continue
						}
						key := ny*s.Board.Width + nx
						if checkedNeighbors[key] {
							continue
						}
						checkedNeighbors[key] = true

						c2 := s.Board.Cells[ny][nx]
						if !c2.IsRevealed || c2.NeighborCount == 0 {
							continue
						}
						_, f2, h2 := s.getNeighborsInfo(nx, ny)
						needed2 := c2.NeighborCount - f2

						if isSubset(h1, h2) {
							diff := getDifference(h2, h1)
							if len(diff) == 0 {
								continue
							}
							minesInDiff := needed2 - needed1

							if minesInDiff == 0 {
								target := diff[0]
								return &Move{X: target.x, Y: target.y, Type: MoveOpen}
							} else if minesInDiff == len(diff) {
								target := diff[0]
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

func (s *Solver) findTankMove() *Move {
	tank := NewTankSolver(s.Board)
	return tank.Solve()
}

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
