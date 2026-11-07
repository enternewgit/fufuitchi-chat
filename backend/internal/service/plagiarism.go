package service

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"unicode"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
)

// Memorizer は、モデルの返答が学習データの「お返事」をどれくらい丸暗記しているかを測る。
// 返答を弾くのではなく、比較モードの指標として記録・表示する。
//
// 割合（一致した部分 ÷ 返答の長さ）だと短い返答ほど高く出てしまうので、
// 「最長で何文字連続して一致したか」という長さに左右されにくい値を使う。
type Memorizer struct {
	refs [][]rune // 学習データの各お返事（記号・空白を除いたもの）
}

// LoadMemorizer は dir 内の *.jsonl（prepare-dataset の出力）から model の発言を読み込む。
// ファイルがなければ参照なしの Memorizer を返す（Overlap は常に ok=false）。
func LoadMemorizer(dir string) (*Memorizer, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	m := &Memorizer{}
	for _, p := range paths {
		if err := m.loadFile(p); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *Memorizer) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var ex struct {
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		ex.Contents = nil
		if err := json.Unmarshal(sc.Bytes(), &ex); err != nil {
			return err
		}
		for _, c := range ex.Contents {
			if c.Role != "model" {
				continue
			}
			for _, p := range c.Parts {
				m.Add(p.Text)
			}
		}
	}
	return sc.Err()
}

// Add は参照するお返事を1つ追加する。
func (m *Memorizer) Add(ref string) {
	if r := normalize(ref); len(r) > 0 {
		m.refs = append(m.refs, r)
	}
}

// Len は参照しているお返事の数。
func (m *Memorizer) Len() int {
	return len(m.refs)
}

// Overlap は返答と学習データのお返事の重なりを返す。参照がないときや返答が空のときは ok=false。
func (m *Memorizer) Overlap(output string) (model.Overlap, bool) {
	out := normalize(output)
	if len(out) == 0 || len(m.refs) == 0 {
		return model.Overlap{}, false
	}
	var o model.Overlap
	for _, ref := range m.refs {
		if string(ref) == string(out) {
			o.Exact = true
		}
		if n := longestCommonRun(out, ref); n > o.Longest {
			o.Longest = n
		}
	}
	return o, true
}

// longestCommonRun は a と b で連続して一致する部分の最長の長さを返す（最長共通部分文字列）。
func longestCommonRun(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	best := 0
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
				if cur[j] > best {
					best = cur[j]
				}
			} else {
				cur[j] = 0
			}
		}
		prev, cur = cur, prev
	}
	return best
}

// normalize は記号と空白を除き、英字を小文字にする。
func normalize(s string) []rune {
	var rs []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			rs = append(rs, unicode.ToLower(r))
		}
	}
	return rs
}
