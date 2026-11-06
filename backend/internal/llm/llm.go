// Package llm は Gemini の呼び出しをまとめる。
// ベースモデルとチューニング済みモデルの違いは「モデルID」だけにする。
package llm

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/genai"

	"github.com/enternewgit/fufuitchi-chat/backend/internal/model"
)

// Generator は service から使うモデル呼び出しの interface。テストで差し替えられるようにする。
type Generator interface {
	// Stream は返答を少しずつ onDelta に渡し、最後に返答全体を返す。
	Stream(ctx context.Context, req Request, onDelta func(string) error) (string, error)
}

// Request は1回のモデル呼び出しに必要なもの。
type Request struct {
	Model        string
	SystemPrompt string
	History      []model.Message // 古い順
	Message      string
}

// Config は Gemini クライアントの接続先。
// APIKey があれば Gemini API、なければ Vertex AI（Project / Location）を使う。
// チューニング済みモデルは Vertex AI でしか呼べない。
type Config struct {
	APIKey   string
	Project  string
	Location string
}

type Client struct {
	c *genai.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	cc := &genai.ClientConfig{}
	if cfg.APIKey != "" {
		cc.Backend = genai.BackendGeminiAPI
		cc.APIKey = cfg.APIKey
	} else {
		if cfg.Project == "" {
			return nil, errors.New("llm: GEMINI_API_KEY か GOOGLE_CLOUD_PROJECT のどちらかが必要です")
		}
		cc.Backend = genai.BackendVertexAI
		cc.Project = cfg.Project
		cc.Location = cfg.Location
	}
	c, err := genai.NewClient(ctx, cc)
	if err != nil {
		return nil, err
	}
	return &Client{c: c}, nil
}

func (c *Client) Stream(ctx context.Context, req Request, onDelta func(string) error) (string, error) {
	contents := make([]*genai.Content, 0, len(req.History)+1)
	for _, m := range req.History {
		contents = append(contents, genai.NewContentFromText(m.Content, genai.Role(m.Role)))
	}
	contents = append(contents, genai.NewContentFromText(req.Message, genai.RoleUser))

	var config *genai.GenerateContentConfig
	if req.SystemPrompt != "" {
		config = &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(req.SystemPrompt, genai.RoleUser),
		}
	}

	var full strings.Builder
	for resp, err := range c.c.Models.GenerateContentStream(ctx, req.Model, contents, config) {
		if err != nil {
			return full.String(), err
		}
		text := resp.Text()
		if text == "" {
			continue
		}
		full.WriteString(text)
		if err := onDelta(text); err != nil {
			return full.String(), err
		}
	}
	return full.String(), nil
}
