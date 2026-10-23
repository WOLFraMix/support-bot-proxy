package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
	"github.com/WOLFraMix/support-bot-proxy/internal/llm"
	"github.com/WOLFraMix/support-bot-proxy/internal/ticket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCreator — заглушка для создания тикетов в тестах
type mockCreator struct {
	key string
	err error
}

func (m mockCreator) CreateTicket(_ context.Context, summary, description string) (string, error) { 
	if m.err != nil {
		return "", m.err
	}
	if m.key != "" {
		return m.key, nil
	}
	return "TEST-MOCK-1", nil
}

func newTestHandlerWithMockLLM(t *testing.T, llmResponse string, llmStatus int) *Handler {
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

	svc := ticket.NewService(client, mockCreator{key: "PROJ-42"})
	return NewHandler(svc)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSupportHandler_Success(t *testing.T) {
	h := newTestHandlerWithMockLLM(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"technical_issue\",\"draft_reply\":\"Мы проверим проблему.\",\"jira_summary\":\"Проблема с входом\",\"jira_description\":\"Пользователь не может войти в аккаунт.\"}"}}]}}`,
		http.StatusOK,
	)

	body := []byte(`{"user_text":"Не могу войти"}`)
	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode, "Ожидается 200 OK")
	assert.Equal(t, "application/json; charset=utf-8", res.Header.Get("Content-Type"))

	var jsonResp struct {
		TicketKey   string `json:"ticket_key"`
		Category    string `json:"category"`
		DraftReply  string `json:"draft_reply"`
		JiraSummary string `json:"jira_summary"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&jsonResp), "Ответ должен быть валидным JSON")
	assert.Equal(t, "PROJ-42", jsonResp.TicketKey)
	assert.Equal(t, "technical_issue", jsonResp.Category)
	assert.Contains(t, jsonResp.DraftReply, "проверим")
}

func TestSupportHandler_InvalidJSON(t *testing.T) {
	h := newTestHandlerWithMockLLM(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"other\",\"draft_reply\":\"OK\",\"jira_summary\":\"Test\",\"jira_description\":\"Test\"}"}}]}}`,
		http.StatusOK,
	)

	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader([]byte("{bad json")))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Ожидается 400 Bad Request для невалидного JSON")
}

func TestSupportHandler_WrongMethod(t *testing.T) {
	h := newTestHandlerWithMockLLM(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"other\",\"draft_reply\":\"OK\",\"jira_summary\":\"Test\",\"jira_description\":\"Test\"}"}}]}}`,
		http.StatusOK,
	)

	req := httptest.NewRequest(http.MethodGet, "/support", nil)
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode, "Ожидается 405 Method Not Allowed для GET")
}

func TestSupportHandler_EmptyBody(t *testing.T) {
	h := newTestHandlerWithMockLLM(t,
		`{"result":{"alternatives":[{"message":{"role":"assistant","text":"{\"category\":\"other\",\"draft_reply\":\"OK\",\"jira_summary\":\"Test\",\"jira_description\":\"Test\"}"}}]}}`,
		http.StatusOK,
	)

	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader(nil))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Ожидается 400 для пустого тела")
}

func TestSupportHandler_LLMServerError(t *testing.T) {
	h := newTestHandlerWithMockLLM(t,
		`{"error":"internal server error"}`,
		http.StatusInternalServerError,
	)

	body := []byte(`{"user_text":"Тест"}`)
	req := httptest.NewRequest(http.MethodPost, "/support", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.SupportHandler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode, "Ожидается 500 при ошибке LLM")
}
