package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"minesweeper/game"
	"minesweeper/solver"
)

func main() {
	// 1万試合のデータを集める
	gamesToPlay := 10000
	filename := "dataset.csv"

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// CSVヘッダー: 周囲5x5マスの情報(25個) + 正解ラベル
	header := []string{}
	for i := 0; i < 25; i++ {
		header = append(header, fmt.Sprintf("cell_%d", i))
	}
	header = append(header, "is_mine")
	writer.Write(header)

	fmt.Printf("Generating data from %d games...\n", gamesToPlay)
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < gamesToPlay; i++ {
		playGameAndRecord(writer)
		if i%1000 == 0 {
			fmt.Print(".")
		}
	}
	fmt.Println("\nDone! Saved to", filename)
}

func playGameAndRecord(writer *csv.Writer) {
	// AI学習用には「中級」程度の密度が良い
	w, h, mines := 9, 9, 10
	b := game.NewBoard(w, h, mines)

	// 最初の一手（ランダムオープン）
	bot := solver.New(b)
	if first := bot.NextMove(); first != nil {
		b.Open(first.X, first.Y)
	}

	for {
		if b.CheckClear() {
			break
		}

		move := bot.NextMove()
		if move == nil {
			break
		}

		// ★重要: 「運任せ（Guess）」の場面だけを記録する
		// ロジックで解ける場面を学習させても意味がないため
		if move.IsGuess {
			recordState(writer, b, move.X, move.Y)
		}

		if move.Type == solver.MoveOpen {
			if !b.Open(move.X, move.Y) {
				break
			} // Game Over
		} else {
			b.ToggleFlag(move.X, move.Y)
		}
	}
}

func recordState(writer *csv.Writer, b *game.Board, tx, ty int) {
	row := []string{}

	// 対象マスを中心に 5x5 の情報を取得
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := tx+dx, ty+dy
			val := 9 // 範囲外(壁)

			if nx >= 0 && nx < b.Width && ny >= 0 && ny < b.Height {
				cell := b.Cells[ny][nx]
				if !cell.IsRevealed {
					if cell.IsFlagged {
						val = -2 // 旗
					} else {
						val = -1 // 未開封
					}
				} else {
					val = cell.NeighborCount // 0-8
				}
			}
			row = append(row, strconv.Itoa(val))
		}
	}

	// 正解ラベル（0:安全, 1:地雷）
	label := "0"
	if b.Cells[ty][tx].IsMine {
		label = "1"
	}
	row = append(row, label)

	writer.Write(row)
}
