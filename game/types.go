package game

type Cell struct {
	IsMine        bool
	IsRevealed    bool
	IsFlagged     bool
	NeighborCount int
}

type Board struct {
	Width         int
	Height        int
	MineCount     int
	Cells         [][]Cell
	IsInitialized bool // 初回クリックが終わったかどうか
	IsGameOver    bool // ゲームオーバーフラグ
}
