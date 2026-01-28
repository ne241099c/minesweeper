package main

import (
	"fmt"
	"minesweeper/game"
)

func main() {
	// gameパッケージのNewBoard関数を呼び出す
	b := game.NewBoard(10, 10, 10)

	fmt.Println("--- 生成された盤面 ---")
	b.DebugPrint()
}
