// 「つくよみちゃん会話AI育成計画」の Excel（data/raw）を、
// Vertex AI の Gemini チューニング用 JSONL（data/processed）に整形するスクリプト。
//
// 使い方: go run ./cmd/prepare-dataset -in ../data/raw/kaiwa-ai.xlsx -out ../data/processed
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// Excel の列（ヘッダーの先頭の文字で探す）
const (
	colA     = "【A】"  // 話しかけ
	colB     = "【B】"  // お返事
	colC     = "【C】"  // Bに対するA話者の返事（２往復のみ）
	colD     = "【D】"  // Cに対するつくよみちゃんのお返事（２往復のみ）
	colTwo   = "２往復"  // マーク
	colOwn   = "固有ネタ" // マーク
	colNamed = "名前入り" // マーク
)

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

// example は Vertex AI の教師ありチューニングの1行。
type example struct {
	SystemInstruction *content  `json:"systemInstruction,omitempty"`
	Contents          []content `json:"contents"`
}

type stats struct {
	rows, twoTurn, own, named, skipped int
}

func main() {
	in := flag.String("in", "../data/raw/kaiwa-ai.xlsx", "入力の Excel ファイル")
	out := flag.String("out", "../data/processed", "出力先ディレクトリ")
	valRatio := flag.Float64("val", 0.1, "validation に回す割合")
	seed := flag.Uint64("seed", 42, "シャッフルの乱数シード")
	maxRunes := flag.Int("max-runes", 500, "1発言の最大文字数（超える会話は除外）")
	system := flag.String("system", "", "全例に付けるシステムプロンプト（空なら付けない）")
	flag.Parse()

	examples, st, err := load(*in, *maxRunes, *system)
	if err != nil {
		log.Fatal(err)
	}
	if len(examples) == 0 {
		log.Fatal("会話が1件も読み込めませんでした")
	}

	r := rand.New(rand.NewPCG(*seed, *seed))
	r.Shuffle(len(examples), func(i, j int) { examples[i], examples[j] = examples[j], examples[i] })
	nVal := int(math.Round(float64(len(examples)) * *valRatio))
	val, train := examples[:nVal], examples[nVal:]

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	trainPath := filepath.Join(*out, "train.jsonl")
	valPath := filepath.Join(*out, "validation.jsonl")
	if err := writeJSONL(trainPath, train); err != nil {
		log.Fatal(err)
	}
	if err := writeJSONL(valPath, val); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("読み込んだ行: %d（２往復 %d / 固有ネタ %d / 名前入り %d / 除外 %d）\n",
		st.rows, st.twoTurn, st.own, st.named, st.skipped)
	fmt.Printf("train: %d 件 → %s\n", len(train), trainPath)
	fmt.Printf("validation: %d 件 → %s\n", len(val), valPath)
}

func load(path string, maxRunes int, system string) ([]example, stats, error) {
	var st stats
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, st, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, st, errors.New("シートがありません")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, st, err
	}

	// 1行目はタイトルなので、【A】で始まるセルがある行をヘッダーとみなす
	headerRow, cols := -1, map[string]int{}
	for i, row := range rows {
		if idx := findCol(row, colA); idx >= 0 {
			headerRow = i
			for _, name := range []string{colA, colB, colC, colD, colTwo, colOwn, colNamed} {
				cols[name] = findCol(row, name)
			}
			break
		}
	}
	if headerRow < 0 || cols[colB] < 0 {
		return nil, st, errors.New("ヘッダー行（【A】話しかけ / 【B】お返事）が見つかりません")
	}

	var sys *content
	if system != "" {
		sys = &content{Parts: []part{{Text: system}}}
	}

	var out []example
	for _, row := range rows[headerRow+1:] {
		a, b := cell(row, cols[colA]), cell(row, cols[colB])
		if a == "" && b == "" {
			continue
		}
		st.rows++
		if marked(row, cols[colOwn]) {
			st.own++
		}
		if marked(row, cols[colNamed]) {
			st.named++
		}

		turns := []string{a, b}
		if c, d := cell(row, cols[colC]), cell(row, cols[colD]); c != "" && d != "" {
			turns = append(turns, c, d)
			st.twoTurn++
		}
		if !valid(turns, maxRunes) {
			st.skipped++
			continue
		}

		ex := example{SystemInstruction: sys}
		for i, t := range turns {
			role := "user"
			if i%2 == 1 {
				role = "model"
			}
			ex.Contents = append(ex.Contents, content{Role: role, Parts: []part{{Text: t}}})
		}
		out = append(out, ex)
	}
	return out, st, nil
}

func findCol(row []string, prefix string) int {
	for i, v := range row {
		if strings.HasPrefix(strings.TrimSpace(v), prefix) {
			return i
		}
	}
	return -1
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// marked はマークの列に何か入っているか（データでは "1"）。
func marked(row []string, idx int) bool {
	return cell(row, idx) != ""
}

// valid は空の発言や長すぎる発言がないか確認する。
func valid(turns []string, maxRunes int) bool {
	for _, t := range turns {
		if t == "" || utf8.RuneCountInString(t) > maxRunes {
			return false
		}
	}
	return true
}

func writeJSONL(path string, examples []example) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	for _, ex := range examples {
		if err := enc.Encode(ex); err != nil {
			f.Close()
			return err
		}
	}
	return f.Close()
}
