package solver

import (
	"minesweeper/game"
)

// TankSolver はバックトラック探索を行う構造体
type TankSolver struct {
	Board *game.Board
}

func NewTankSolver(b *game.Board) *TankSolver {
	return &TankSolver{Board: b}
}

// Solve はタンクアルゴリズムを実行し、確定した安全な手または地雷を返します
func (ts *TankSolver) Solve() *Move {
	// 1. 全ての境界マスと、それに関連する数字マスを特定してグループ化（連結成分分解）
	segments := ts.createSegments()

	var bestMove *Move
	bestProb := 1.0 // 1.0 = 地雷確率100% (最悪)

	// 各セグメントごとに独立して解く
	for _, seg := range segments {
		// セグメントが大きすぎる場合は解けないのでスキップ
		if len(seg.unknowns) > 18 { // 18程度が限界
			continue
		}

		solutions := ts.solveSegment(seg)
		if len(solutions) == 0 {
			continue // 解なし（矛盾）
		}

		// 各マスの地雷確率を計算
		counts := make([]int, len(seg.unknowns))
		for _, sol := range solutions {
			for i, isMine := range sol {
				if isMine {
					counts[i]++
				}
			}
		}

		total := float64(len(solutions))
		for i, count := range counts {
			prob := float64(count) / total
			pos := seg.unknowns[i]

			// 確定安全 (0%)
			if prob == 0.0 {
				return &Move{X: pos.x, Y: pos.y, Type: MoveOpen, Strategy: "Tank", Confidence: 1.0}
			}
			// 確定地雷 (100%)
			if prob == 1.0 && !ts.Board.Cells[pos.y][pos.x].IsFlagged {
				return &Move{X: pos.x, Y: pos.y, Type: MoveFlag, Strategy: "Tank", Confidence: 1.0}
			}

			// 最善手（確率）の更新
			// 確率が低いほうが安全
			if prob < bestProb {
				bestProb = prob
				bestMove = &Move{
					X: pos.x, Y: pos.y,
					Type:       MoveOpen,
					Strategy:   "Tank(Prob)",
					Confidence: 1.0 - prob,
				}
			}
		}
	}

	return bestMove
}

// --- セグメント（連結成分）管理 ---

type segment struct {
	unknowns []pos  // このセグメントに含まれる未開封マス
	rules    []rule // このセグメント内の数字マス制約
}

type rule struct {
	cells []int // unknownsのインデックスのリスト
	mines int   // 必要な地雷数
}

func (ts *TankSolver) createSegments() []*segment {
	// 1. 全ての「数字マス」と「それに隣接する未開封マス」の関係をリスト化
	unknownMap := make(map[int]pos) // key: y*w+x
	numberedCells := []pos{}

	for y := 0; y < ts.Board.Height; y++ {
		for x := 0; x < ts.Board.Width; x++ {
			c := ts.Board.Cells[y][x]
			if c.IsRevealed && c.NeighborCount > 0 {
				// 周囲の未開封をチェック
				hasUnknown := false
				_, flags, _ := ts.getNeighbors(x, y)
				if flags == c.NeighborCount {
					continue
				}

				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < ts.Board.Width && ny >= 0 && ny < ts.Board.Height {
							neighbor := ts.Board.Cells[ny][nx]
							if !neighbor.IsRevealed && !neighbor.IsFlagged {
								key := ny*ts.Board.Width + nx
								unknownMap[key] = pos{nx, ny}
								hasUnknown = true
							}
						}
					}
				}
				if hasUnknown {
					numberedCells = append(numberedCells, pos{x, y})
				}
			}
		}
	}

	// 2. 連結成分分解
	// unknownsをノード、数字マスをエッジとしたグラフを作る
	adj := make(map[int][]int) // unknownKey -> []unknownKey

	for _, numPos := range numberedCells {
		_, _, neighbors := ts.getNeighbors(numPos.x, numPos.y)
		for i := 0; i < len(neighbors)-1; i++ {
			u1 := neighbors[i].y*ts.Board.Width + neighbors[i].x
			for j := i + 1; j < len(neighbors); j++ {
				u2 := neighbors[j].y*ts.Board.Width + neighbors[j].x
				adj[u1] = append(adj[u1], u2)
				adj[u2] = append(adj[u2], u1)
			}
		}
	}

	visited := make(map[int]bool)
	var segments []*segment

	// mapのループ変数 p (pos) は使わないので省略 (keyのみ使用)
	for key := range unknownMap {
		if visited[key] {
			continue
		}

		// BFSでグループ探索
		groupKeys := []int{}
		queue := []int{key}
		visited[key] = true

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			groupKeys = append(groupKeys, curr)

			for _, neighbor := range adj[curr] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		// セグメント作成
		seg := &segment{
			unknowns: make([]pos, len(groupKeys)),
			rules:    []rule{},
		}

		localIndexMap := make(map[int]int)
		for i, k := range groupKeys {
			seg.unknowns[i] = unknownMap[k]
			localIndexMap[k] = i
		}

		// ルール生成
		for _, numPos := range numberedCells {
			_, flags, neighbors := ts.getNeighbors(numPos.x, numPos.y)
			if len(neighbors) == 0 {
				continue
			}

			// neighborsの最初の1つがこのセグメントに含まれていれば、
			// 全て含まれているはず（連結しているため）
			firstKey := neighbors[0].y*ts.Board.Width + neighbors[0].x
			if _, ok := localIndexMap[firstKey]; ok {
				r := rule{
					cells: make([]int, len(neighbors)),
					mines: ts.Board.Cells[numPos.y][numPos.x].NeighborCount - flags,
				}
				for i, n := range neighbors {
					nk := n.y*ts.Board.Width + n.x
					r.cells[i] = localIndexMap[nk]
				}
				seg.rules = append(seg.rules, r)
			}
		}
		segments = append(segments, seg)
	}

	return segments
}

// --- 探索ロジック ---

func (ts *TankSolver) solveSegment(seg *segment) [][]bool {
	solutions := [][]bool{}
	config := make([]bool, len(seg.unknowns))
	ts.backtrack(seg, 0, config, &solutions)
	return solutions
}

func (ts *TankSolver) backtrack(seg *segment, index int, config []bool, solutions *[][]bool) {
	if index == len(seg.unknowns) {
		if ts.isValid(seg, config, true) {
			sol := make([]bool, len(config))
			copy(sol, config)
			*solutions = append(*solutions, sol)
		}
		return
	}

	// 枝刈り
	if !ts.isValid(seg, config, false) {
		return
	}

	// 仮定1: 地雷
	config[index] = true
	ts.backtrack(seg, index+1, config, solutions)

	// 仮定2: 安全
	config[index] = false
	ts.backtrack(seg, index+1, config, solutions)
}

func (ts *TankSolver) isValid(seg *segment, config []bool, isFinal bool) bool {
	for _, r := range seg.rules {
		mines := 0
		// unknowns（未決定数）は削除（簡易チェックのため）

		for _, idx := range r.cells {
			if config[idx] {
				mines++
			}
		}

		if isFinal {
			// 最終チェック: 地雷数がぴったり一致すること
			if mines != r.mines {
				return false
			}
		} else {
			// 途中チェック: 既に地雷数がオーバーしていたらアウト
			if mines > r.mines {
				return false
			}
			// ※「残り全てを地雷にしても足りない」ケースの枝刈りは省略（実装の複雑化回避のため）
		}
	}
	return true
}

// ヘルパー
func (ts *TankSolver) getNeighbors(cx, cy int) (totalHidden int, flags int, hiddenList []pos) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := cx+dx, cy+dy
			if nx >= 0 && nx < ts.Board.Width && ny >= 0 && ny < ts.Board.Height {
				neighbor := ts.Board.Cells[ny][nx]
				if neighbor.IsFlagged {
					flags++
				} else if !neighbor.IsRevealed {
					totalHidden++
					hiddenList = append(hiddenList, pos{nx, ny})
				}
			}
		}
	}
	return
}
