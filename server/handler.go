package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"minesweeper/game"
)

// Server はゲームの状態とHTTPハンドラを管理します
type Server struct {
	Board *game.Board
	Mutex sync.Mutex
}

// NewServer はサーバーインスタンスを初期化します
func NewServer() *Server {
	s := &Server{}
	s.StartNewGame() // 初期ゲーム作成
	return s
}

// StartNewGame はゲームをリセットします
func (s *Server) StartNewGame() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	// 設定はとりあえず固定値
	s.Board = game.NewBoard(10, 10, 10)
}

// クライアントへのレスポンス用構造体
type CellView struct {
	State         string `json:"state"`
	NeighborCount int    `json:"count"`
	IsMine        bool   `json:"is_mine"`
}

type Response struct {
	Cells     [][]CellView `json:"cells"`
	GameOver  bool         `json:"game_over"`
	GameClear bool         `json:"game_clear"`
}

// HandleNew はゲームリセットAPI
func (s *Server) HandleNew(w http.ResponseWriter, r *http.Request) {
	s.StartNewGame()
	s.sendBoardState(w, false)
}

// HandleOpen はマスを開けるAPI
func (s *Server) HandleOpen(w http.ResponseWriter, r *http.Request) {
	xStr := r.URL.Query().Get("x")
	yStr := r.URL.Query().Get("y")
	x, _ := strconv.Atoi(xStr)
	y, _ := strconv.Atoi(yStr)

	s.Mutex.Lock()
	// まだ盤面がない場合は作るなどの安全策を入れても良い
	if s.Board == nil {
		s.Board = game.NewBoard(10, 10, 10)
	}
	isSafe := s.Board.Open(x, y)
	s.Mutex.Unlock()

	s.sendBoardState(w, !isSafe)
}

// sendBoardState は現在の盤面状態をJSONで返します
func (s *Server) sendBoardState(w http.ResponseWriter, isGameOver bool) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	board := s.Board
	h := board.Height
	w_len := board.Width

	resp := Response{
		Cells:    make([][]CellView, h),
		GameOver: isGameOver,
	}

	for y := 0; y < h; y++ {
		resp.Cells[y] = make([]CellView, w_len)
		for x := 0; x < w_len; x++ {
			c := board.Cells[y][x]
			view := CellView{}

			if c.IsRevealed {
				view.State = "opened"
				if c.IsMine {
					view.IsMine = true
				} else {
					view.NeighborCount = c.NeighborCount
				}
			} else {
				view.State = "hidden"
			}

			if isGameOver && c.IsMine {
				view.IsMine = true
				view.State = "opened"
			}

			resp.Cells[y][x] = view
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
