package solver

import (
	"math/rand"
	"minesweeper/ai"
	"minesweeper/game"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// MoveType は行動の種類
type MoveType int

const (
	MoveOpen MoveType = iota
	MoveFlag
)

// Move はBotの1手の行動を表します
type Move struct {
	X, Y    int
	Type    MoveType
	IsGuess bool
}

type Solver struct {
	Board *game.Board
	AiNet *ai.Network // AIネットワークを保持
}

func New(b *game.Board) *Solver {
	// AIの初期化
	net, _ := ai.NewNetwork(game.GetWeightsJSON())
	return &Solver{Board: b, AiNet: net}
}

// NextMove は次の最適な一手を返します
func (s *Solver) NextMove() *Move {
	// 論理的に絶対に安全
	if move := s.findSafeMove(); move != nil {
		move.IsGuess = false // 論理
		return move
	}

	// 論理的に絶対に地雷
	if move := s.findFlagMove(); move != nil {
		move.IsGuess = false // 論理
		return move
	}

	// ランダム
	move := s.findRandomMove()
	if move != nil {
		move.IsGuess = true // 運
	}
	return move
}

// findSafeMove: 「数字 == 周囲の旗数」なら、残りの未開封マスは安全
func (s *Solver) findSafeMove() *Move {
	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			cell := s.Board.Cells[y][x]
			// 開いている数字マスだけを見る
			if !cell.IsRevealed || cell.NeighborCount == 0 {
				continue
			}

			// 周囲の情報を集める
			_, flags, hidden := s.getNeighborsInfo(x, y)

			// 周囲の旗の数が、このマスの数字と一致していて、まだ開いていないマスがある場合
			if flags == cell.NeighborCount && len(hidden) > 0 {
				// 安全なマス（hiddenの中身）を1つ返す
				target := hidden[0]
				return &Move{X: target.x, Y: target.y, Type: MoveOpen}
			}
		}
	}
	return nil
}

// findFlagMove: 「数字 == 周囲の未開封マス数」なら、未開封マスは全て地雷
func (s *Solver) findFlagMove() *Move {
	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			cell := s.Board.Cells[y][x]
			if !cell.IsRevealed || cell.NeighborCount == 0 {
				continue
			}

			// 周囲の情報を集める
			totalHidden, flags, hidden := s.getNeighborsInfo(x, y)

			// まだ旗が立っていない未開封マスがあるか？
			unflaggedCount := totalHidden - flags

			// 「周囲の未開封数(旗含む)」と「数字」が同じなら、未開封で旗がない場所は地雷
			if totalHidden == cell.NeighborCount && unflaggedCount > 0 {
				// 旗を立てるべきマスを探す
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

// findRandomMove: 完全ランダム（未開封で旗のない場所）
func (s *Solver) findRandomMove() *Move {
	// AIが使えない（初期化失敗など）場合は、従来の完全ランダムへフォールバック
	if s.AiNet == nil {
		return s.findPureRandomMove()
	}

	bestProb := 1.0 // 最初は確率100%（最悪）にしておく
	var bestMove *Move

	// 未開封マスを全てスキャンして確率を計算
	for y := 0; y < s.Board.Height; y++ {
		for x := 0; x < s.Board.Width; x++ {
			c := s.Board.Cells[y][x]
			// 未開封かつフラグなしの場所だけが候補
			if !c.IsRevealed && !c.IsFlagged {
				// 入力データ作成 (5x5)
				input := s.createAiInput(x, y)
				// AI推論
				prob := s.AiNet.Predict(input)

				// より地雷確率が低い（安全な）マスが見つかったら更新
				if prob < bestProb {
					bestProb = prob
					bestMove = &Move{X: x, Y: y, Type: MoveOpen}
				}
			}
		}
	}

	// もしAIが候補を見つけたらそれを返す
	if bestMove != nil {
		return bestMove
	}

	// 万が一候補がなければ完全ランダム
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
	return &Move{X: choice.x, Y: choice.y, Type: MoveOpen}
}

// AI用の入力データ作成 (5x5)
func (s *Solver) createAiInput(tx, ty int) []float64 {
	input := make([]float64, 25)
	idx := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := tx+dx, ty+dy
			val := 9.0 // 壁

			if nx >= 0 && nx < s.Board.Width && ny >= 0 && ny < s.Board.Height {
				cell := s.Board.Cells[ny][nx]
				if !cell.IsRevealed {
					if cell.IsFlagged {
						val = -2.0 // 旗
					} else {
						val = -1.0 // 未開封
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

// ヘルパー: 周囲のマスの情報を取得
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
