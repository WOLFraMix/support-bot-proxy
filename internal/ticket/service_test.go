package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
	"github.com/WOLFraMix/support-bot-proxy/internal/llm"
)

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestService_Process_Integration(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Config{
		YandexAPIKey:   "test-key",
		YandexFolderID: "test-folder",
	}

	client := llm.NewYandexClient(cfg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "only POST", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"results":[{"alternatives":[{"message":{"text":"{\"category\":\"technical_issue\",\"draft_reply\":\"Мы проверим проблему.\",\"jira_summary\":\"Проблема с входом\",\"jira_description\":\"Пользователь не может войти в аккаунт.\"}"}}]}]}`))
	}))
	defer ts.Close()

	client.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = ts.URL[len("http://"):]
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	service := NewService(client, MockCreator{})
	req := SupportRequest{UserText: "Не могу войти в аккаунт"}
	resp, err := service.Process(ctx, req)

	require.NoError(t, err, "Сервис не должен возвращать ошибку при валидном ответе LLM")
	assert.NotEmpty(t, resp.TicketKey, "Ключ тикета должен быть сгенерирован")
	assert.True(t, strings.HasPrefix(resp.TicketKey, "TEST-"), "Ключ должен начинаться с TEST-")
	assert.Equal(t, "technical_issue", resp.Category, "Категория должна быть technical_issue")
	assert.Contains(t, resp.DraftReply, "проверим", "Черновик должен содержать ожидаемую фразу")
}

func TestService_Process_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		YandexAPIKey:   "test-key",
		YandexFolderID: "test-folder",
	}

	client := llm.NewYandexClient(cfg)

	// Возвращаем невалидный JSON: просто текст, не JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte("это вообще не JSON"))
	}))
	defer ts.Close()

	client.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = ts.URL[len("http://"):]
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	service := NewService(client, MockCreator{})
	req := SupportRequest{UserText: "Странный запрос"}

	_, err := service.Process(ctx, req)
	assert.Error(t, err, "При невалидном JSON сервис должен вернуть ошибку")

	// Проверяем тип ошибки, а не текст
	var syntaxErr *json.SyntaxError
	assert.ErrorAs(t, err, &syntaxErr, "Ошибка должна быть синтаксической ошибкой JSON")
}
