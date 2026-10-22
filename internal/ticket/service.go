package ticket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/WOLFraMix/support-bot-proxy/internal/llm"
)

type Service struct {
	llmClient *llm.YandexClient
}

func NewService(llmClient *llm.YandexClient) *Service {
	return &Service{llmClient: llmClient}
}

type SupportRequest struct {
	UserText string `json:"user_text"`
}

type SupportResponse struct {
	TicketKey       string `json:"ticket_key"`
	Category        string `json:"category"`
	DraftReply      string `json:"draft_reply"`
	JiraSummary     string `json:"jira_summary"`
	JiraDescription string `json:"jira_description"`
}

// Process обрабатывает запрос: LLM → парсинг JSON → Mock Jira → ответ
func (s *Service) Process(ctx context.Context, req SupportRequest) (*SupportResponse, error) {
	raw, err := s.llmClient.GenerateResponse(ctx, req.UserText)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Category        string `json:"category"`
		DraftReply      string `json:"draft_reply"`
		JiraSummary     string `json:"jira_summary"`
		JiraDescription string `json:"jira_description"`
	}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}

	ticketKey := GenerateTicketKey()

	return &SupportResponse{
		TicketKey:       ticketKey,
		Category:        parsed.Category,
		DraftReply:      parsed.DraftReply,
		JiraSummary:     parsed.JiraSummary,
		JiraDescription: parsed.JiraDescription,
	}, nil
}
