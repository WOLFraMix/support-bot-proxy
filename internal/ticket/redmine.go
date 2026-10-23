package ticket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RedmineConfig — параметры подключения к Redmine.
type RedmineConfig struct {
	URL       string // например, http://redmine:3000 — внутри docker-сети
	APIKey    string // персональный ключ из «Моя учётная запись → Доступ к API»
	ProjectID string // identifier проекта, например support
	TrackerID int    // например 1 = Bug
}

// RedmineClient — создание тикетов через REST API Redmine.
type RedmineClient struct {
	cfg    RedmineConfig
	client *http.Client
}

// NewRedmineClient создаёт клиент с таймаутом 15 секунд.
func NewRedmineClient(cfg RedmineConfig) *RedmineClient {
	return &RedmineClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateTicket создаёт задачу в Redmine и возвращает её ключ (#123).
func (r *RedmineClient) CreateTicket(ctx context.Context, summary, description string) (string, error) {
	payload := map[string]any{
		"issue": map[string]any{
			"subject":     summary,     // jira_summary из LLM
			"description": description, // jira_description из LLM
			"project_id":  r.cfg.ProjectID,
			"tracker_id":  r.cfg.TrackerID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.cfg.URL+"/issues.json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Redmine-API-Key", r.cfg.APIKey)

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("redmine request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return "", fmt.Errorf("redmine returned %d: %s", resp.StatusCode, buf.String())
	}

	var created struct {
		Issue struct {
			ID int `json:"id"`
		} `json:"issue"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return fmt.Sprintf("#%d", created.Issue.ID), nil
}
