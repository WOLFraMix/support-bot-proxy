package ticket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/WOLFraMix/support-bot-proxy/internal/llm"
)

type Service struct {
	llmClient *llm.YandexClient
	tickets   TicketCreator // вместо прямого вызова GenerateTicketKey
}

// TicketCreator — то, что умеет создавать тикеты. Реализации: мок и Redmine.
type TicketCreator interface {
	CreateTicket(ctx context.Context, summary, description string) (string, error)
}

func NewService(llmClient *llm.YandexClient, tickets TicketCreator) *Service {
	return &Service{llmClient: llmClient, tickets: tickets}
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

	// Модель часто заворачивает ответ в markdown-блок ```json ... ```
	// Вырезаем чистый JSON перед парсингом
	clean := llm.ExtractJSON(raw)

	var parsed struct {
		Category        string `json:"category"`
		DraftReply      string `json:"draft_reply"`
		JiraSummary     string `json:"jira_summary"`
		JiraDescription string `json:"jira_description"`
	}

	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w (raw: %s)", err, raw)
	}

	ticketKey, err := s.tickets.CreateTicket(ctx, parsed.JiraSummary, parsed.JiraDescription)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}

	return &SupportResponse{
		TicketKey:       ticketKey,
		Category:        parsed.Category,
		DraftReply:      parsed.DraftReply,
		JiraSummary:     parsed.JiraSummary,
		JiraDescription: parsed.JiraDescription,
	}, nil
}
