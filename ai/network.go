package ai

import (
	"encoding/json"
	"math"
)

// 重みデータの構造体（JSONと同じ構造）
type Weights struct {
	Fc1Weight [][]float64 `json:"fc1_weight"`
	Fc1Bias   []float64   `json:"fc1_bias"`
	Fc2Weight [][]float64 `json:"fc2_weight"`
	Fc2Bias   []float64   `json:"fc2_bias"`
	Fc3Weight [][]float64 `json:"fc3_weight"`
	Fc3Bias   []float64   `json:"fc3_bias"`
}

// Network は推論を行うための構造体
type Network struct {
	w Weights
}

// NewNetwork はJSONデータからネットワークを初期化します
func NewNetwork(jsonData []byte) (*Network, error) {
	var w Weights
	if err := json.Unmarshal(jsonData, &w); err != nil {
		return nil, err
	}
	return &Network{w: w}, nil
}

// Predict は入力(25個の数値)を受け取り、地雷確率(0.0~1.0)を返します
func (n *Network) Predict(input []float64) float64 {
	// Layer 1
	out1 := matVecMul(n.w.Fc1Weight, input)
	out1 = addBias(out1, n.w.Fc1Bias)
	out1 = relu(out1)

	// Layer 2
	out2 := matVecMul(n.w.Fc2Weight, out1)
	out2 = addBias(out2, n.w.Fc2Bias)
	out2 = relu(out2)

	// Layer 3 (Output)
	out3 := matVecMul(n.w.Fc3Weight, out2)
	out3 = addBias(out3, n.w.Fc3Bias)

	// Sigmoidで0~1の確率に変換
	return sigmoid(out3[0])
}

// --- 以下、行列演算などのヘルパー関数 ---

// 行列とベクトルの掛け算
func matVecMul(mat [][]float64, vec []float64) []float64 {
	rows := len(mat)
	cols := len(mat[0])
	result := make([]float64, rows)

	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			sum += mat[i][j] * vec[j]
		}
		result[i] = sum
	}
	return result
}

// バイアスの加算
func addBias(vec []float64, bias []float64) []float64 {
	result := make([]float64, len(vec))
	for i := 0; i < len(vec); i++ {
		result[i] = vec[i] + bias[i]
	}
	return result
}

// ReLU関数
func relu(vec []float64) []float64 {
	result := make([]float64, len(vec))
	for i := 0; i < len(vec); i++ {
		if vec[i] > 0 {
			result[i] = vec[i]
		} else {
			result[i] = 0
		}
	}
	return result
}

// Sigmoid関数
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}
