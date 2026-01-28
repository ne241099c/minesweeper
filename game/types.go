package game

// Cell は1つのマスの情報を持ちます
type Cell struct {
	IsMine        bool // 地雷かどうか
	IsRevealed    bool // すでに開けられたか
	IsFlagged     bool // フラグが立てられているか
	NeighborCount int  // 周囲8マスにある地雷の数
}

// Board はゲーム盤面全体を持ちます
type Board struct {
	Width  int      // 横のマス数
	Height int      // 縦のマス数
	Cells  [][]Cell // 2次元配列でマスを管理
}
