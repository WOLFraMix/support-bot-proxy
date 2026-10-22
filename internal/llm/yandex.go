package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
)

const yandexURL = "https://llm.api.cloud.yandex.net/foundation-models/v1/completion"

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

// Вложенные структуры с правильными JSON-тегами
type yandexRequest struct {
	ModelUri       string         `json:"modelUri"`
	CompletionSpec completionSpec `json:"completionSpec"`
}

type completionSpec struct {
	TextCompletionSpec textCompletionSpec `json:"textCompletionSpec"`
}

type textCompletionSpec struct {
	Prompt        string   `json:"prompt"`
	Temperature   float64  `json:"temperature"`
	MaxTokens     int      `json:"maxTokens"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type yandexResponse struct {
	Results []struct {
		Alternatives []struct {
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
		} `json:"alternatives"`
	} `json:"results"`
}

func (c *YandexClient) GenerateResponse(ctx context.Context, userText string) (string, error) {
	prompt := fmt.Sprintf(`Ты — помощник поддержки. Пользователь прислал сообщение: "%s". Верни JSON строго в этом формате (ничего кроме JSON):
{
  "category": "technical_issue|billing|feature_request|other",
  "draft_reply": "Короткий черновик ответа пользователю (не более 150 символов)",
  "jira_summary": "Заголовок задачи для Jira (до 100 символов)",
  "jira_description": "Описание задачи для Jira"
}
Если не можешь классифицировать — поставь category: other.`, userText)

	reqBody := yandexRequest{
		ModelUri: fmt.Sprintf("gpt://%s/yandexgpt/latest", c.cfg.YandexFolderID),
		CompletionSpec: completionSpec{
			TextCompletionSpec: textCompletionSpec{
				Prompt:      prompt,
				Temperature: 0.3,
				MaxTokens:   512,
			},
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

	if len(yResp.Results) == 0 || len(yResp.Results[0].Alternatives) == 0 {
		return "", errors.New("empty response from YandexGPT")
	}

	return yResp.Results[0].Alternatives[0].Message.Text, nil
}
