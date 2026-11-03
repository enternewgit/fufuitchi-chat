// 会話データ（data/raw）を、チューニング用のJSONL（data/processed）に整形するスクリプト。
//
// 使い方: go run ./cmd/prepare-dataset
package main

import "log"

func main() {
	// TODO:
	//   1. data/raw の会話データを読み込む
	//   2. 空のメッセージや長すぎる例を除外する
	//   3. Gemini のチューニング形式（contents: role / parts）に変換する
	//   4. train / validation に分けて data/processed に書き出す
	log.Println("prepare-dataset: not implemented yet")
}
