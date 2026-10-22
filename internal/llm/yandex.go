package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
)

const yandexURL = "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"

type YandexClient struct {
	// Делаем поле экспортируемым, чтобы тесты могли его подменить
	HTTPClient *http.Client
	cfg        *config.Config
}

func NewYandexClient(cfg *config.Config) *YandexClient {
	return &YandexClient{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		cfg:        cfg,
	}
}

// ---- Тело запроса (формат completion API) ----

type yandexRequest struct {
	ModelURI          string            `json:"modelUri"`
	CompletionOptions completionOptions `json:"completionOptions"`
	Messages          []message         `json:"messages"`
}

type completionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// ---- Разбор ответа API: { "result": { "alternatives": [...] } } ----

type yandexResponse struct {
	Result struct {
		Alternatives []struct {
			Message struct {
				Role string `json:"role"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"alternatives"`
	} `json:"result"`
}

// ExtractJSON вырезает чистый JSON из ответа модели,
// даже если тот завёрнут в markdown-блок ```...```
func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	// Страховка: берём текст от первой { до последней }
	if start := strings.Index(s, "{"); start != -1 {
		if end := strings.LastIndex(s, "}"); end != -1 && end > start {
			return s[start : end+1]
		}
	}
	return s
}

func (c *YandexClient) GenerateResponse(ctx context.Context, userText string) (string, error) {
	systemPrompt := `Ты — помощник поддержки. Верни JSON строго в этом формате (ничего кроме JSON):
{
  "category": "technical_issue|billing|feature_request|other",
  "draft_reply": "Короткий черновик ответа пользователю (не более 150 символов)",
  "jira_summary": "Заголовок задачи для Jira (до 100 символов)",
  "jira_description": "Описание задачи для Jira"
}
Если не можешь классифицировать — поставь category: other.`

	reqBody := yandexRequest{
		ModelURI: fmt.Sprintf("gpt://%s/yandexgpt/latest", c.cfg.YandexFolderID),
		CompletionOptions: completionOptions{
			Stream:      false,
			Temperature: 0.3,
			MaxTokens:   512,
		},
		Messages: []message{
			{Role: "system", Text: systemPrompt},
			{Role: "user", Text: userText},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, yandexURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", c.cfg.YandexAPIKey))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("YandexGPT error: status %d, body %s", resp.StatusCode, string(body))
	}

	var yResp yandexResponse
	if err := json.NewDecoder(resp.Body).Decode(&yResp); err != nil {
		return "", err
	}

	if len(yResp.Result.Alternatives) == 0 {
		return "", errors.New("empty response from YandexGPT")
	}

	return yResp.Result.Alternatives[0].Message.Text, nil
}
