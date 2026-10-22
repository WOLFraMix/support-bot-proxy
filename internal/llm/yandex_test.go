package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RoundTripFunc — тип-обёртка, чтобы легко подменять транспорт в тестах
type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestYandexClient_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"results":[{"alternatives":[{"message":{"text":"{\"category\":\"technical_issue\",\"draft_reply\":\"Мы проверим проблему.\",\"jira_summary\":\"Проблема с входом\",\"jira_description\":\"Пользователь не может войти в аккаунт.\"}"}}]}]}`))
	}))
	defer ts.Close()

	cfg := &config.Config{
		YandexAPIKey:   "test-key",
		YandexFolderID: "test-folder",
	}

	client := NewYandexClient(cfg)

	// Подменяем HTTPClient: никаких лишних переменных, сразу передаём RoundTripFunc
	client.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = ts.URL[len("http://"):]
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	ctx := context.Background()
	text, err := client.GenerateResponse(ctx, "Не могу войти в аккаунт")
	require.NoError(t, err, "GenerateResponse не должен возвращать ошибку при валидном ответе")
	assert.True(t, strings.Contains(text, "technical_issue"), "Ответ должен содержать category: technical_issue")
}

func TestYandexClient_InvalidResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("something went wrong"))
	}))
	defer ts.Close()

	cfg := &config.Config{
		YandexAPIKey:   "test-key",
		YandexFolderID: "test-folder",
	}

	client := NewYandexClient(cfg)
	client.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = ts.URL[len("http://"):]
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	ctx := context.Background()
	_, err := client.GenerateResponse(ctx, "Запрос")
	assert.Error(t, err, "При ошибке сервера должен вернуться error")
}
