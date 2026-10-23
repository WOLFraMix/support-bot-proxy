package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
	"github.com/WOLFraMix/support-bot-proxy/internal/handler"
	"github.com/WOLFraMix/support-bot-proxy/internal/llm"
	"github.com/WOLFraMix/support-bot-proxy/internal/ticket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc — вспомогательный тип для подмены HTTP-транспорта
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// setupTestHandler создаёт Handler с подменённым LLM (httptest) и моком для создания тикетов
func setupTestHandler(t *testing.T, llmResponse string, llmStatus int) *handler.Handler {
	t.Helper()

	cfg := &config.Config{
		YandexAPIKey:   "test-key",
		YandexFolderID: "test-folder",
	}

	client := llm.NewYandexClient(cfg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(llmStatus)
		_, _ = w.Write([]byte(llmResponse))
	}))
	t.Cleanup(func() { ts.Close() })

	client.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = ts.URL[len("http://"):]
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	svc := ticket.NewService(client, ticket.MockCreator{})
	return handler.NewHandler(svc)
}

func TestFullFlow_SuccessfulRequest(t *testing.T) {
	h := setupTestHandler(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"billing\",\"draft_reply\":\"Проверяем ваш платёж.\",\"jira_summary\":\"Ошибка оплаты\",\"jira_description\":\"Не проходит оплата картой.\"}"}}]}}`,
		http.StatusOK,
	)

	body, _ := json.Marshal(map[string]string{"user_text": "Не проходит оплата картой"})
	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode, "Ожидается 200 OK для полного успешного потока")

	var jsonResp struct {
		TicketKey   string `json:"ticket_key"`
		Category    string `json:"category"`
		DraftReply  string `json:"draft_reply"`
		JiraSummary string `json:"jira_summary"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&jsonResp), "Ответ должен быть валидным JSON")
	assert.Equal(t, "billing", jsonResp.Category, "Категория должна быть billing")
	assert.Contains(t, strings.ToLower(jsonResp.DraftReply), "проверяем", "Черновик должен содержать ожидаемую фразу")
	assert.NotEmpty(t, jsonResp.JiraSummary, "Jira summary не должен быть пустым")
}

func TestFullFlow_InvalidJSON(t *testing.T) {
	h := setupTestHandler(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"other\",\"draft_reply\":\"OK\",\"jira_summary\":\"Test\",\"jira_description\":\"Test\"}"}}]}}`,
		http.StatusOK,
	)

	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader([]byte(`{invalid`)))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Ожидается 400 для невалидного JSON")
}

func TestFullFlow_WrongMethod(t *testing.T) {
	h := setupTestHandler(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"other\",\"draft_reply\":\"OK\",\"jira_summary\":\"Test\",\"jira_description\":\"Test\"}"}}]}}`,
		http.StatusOK,
	)

	req := httptest.NewRequest(http.MethodGet, "/support", nil)
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode, "Ожидается 405 для GET запроса")
}

func TestFullFlow_LLMUnavailable(t *testing.T) {
	h := setupTestHandler(t,
		`{"error":"service unavailable"}`,
		http.StatusBadGateway,
	)

	body, _ := json.Marshal(map[string]string{"user_text": "Тест"})
	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode, "Ожидается 500 при ошибке LLM")
}
