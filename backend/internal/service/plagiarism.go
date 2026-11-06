package service

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Memorizer は、モデルの返答が学習データの「お返事」をどれくらい丸暗記しているかを測る。
// 返答を弾くのではなく、比較モードの指標として記録・表示する。
type Memorizer struct {
	refs []map[string]struct{} // 学習データの各お返事の文字 n-gram
}

const ngramSize = 3

// LoadMemorizer は dir 内の *.jsonl（prepare-dataset の出力）から model の発言を読み込む。
// ファイルがなければ参照なしの Memorizer を返す（Score は常に ok=false）。
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
	if g := ngrams(ref); len(g) > 0 {
		m.refs = append(m.refs, g)
	}
}

// Len は参照しているお返事の数。
func (m *Memorizer) Len() int {
	return len(m.refs)
}

// Score は、返答の n-gram のうち、最も似ているお返事に含まれる割合（0〜1）を返す。
// 1 に近いほど学習データをそのまま返している。参照がないときや返答が短すぎるときは ok=false。
func (m *Memorizer) Score(output string) (score float64, ok bool) {
	out := ngrams(output)
	if len(out) == 0 || len(m.refs) == 0 {
		return 0, false
	}
	best := 0
	for _, ref := range m.refs {
		hit := 0
		for g := range out {
			if _, ok := ref[g]; ok {
				hit++
			}
		}
		if hit > best {
			best = hit
		}
	}
	return float64(best) / float64(len(out)), true
}

// ngrams は記号と空白を除いた文字列の文字 n-gram の集合を返す。
func ngrams(s string) map[string]struct{} {
	var rs []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			rs = append(rs, unicode.ToLower(r))
		}
	}
	if len(rs) < ngramSize {
		return nil
	}
	set := make(map[string]struct{}, len(rs))
	for i := 0; i+ngramSize <= len(rs); i++ {
		set[strings.Clone(string(rs[i:i+ngramSize]))] = struct{}{}
	}
	return set
}
