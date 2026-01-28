package game

import (
	_ "embed"
)

//go:embed weights.json
var WeightsJSON []byte

// GetWeightsJSON は埋め込まれた重みデータを返します
func GetWeightsJSON() []byte {
	return WeightsJSON
}
