// internal/handler/handler.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/WOLFraMix/support-bot-proxy/internal/ticket"
)

// Используем тип из пакета ticket — никаких дубликатов
type Handler struct {
	service *ticket.Service
}

func NewHandler(service *ticket.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SupportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Используем тот же тип, что и в ticket.Process
		var req ticket.SupportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		resp, err := h.service.Process(r.Context(), req)
		if err != nil {
			// В реальном проекте тут лучше логировать ошибку, но для MVP хватит 500
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			TicketKey   string `json:"ticket_key"`
			Category    string `json:"category"`
			DraftReply  string `json:"draft_reply"`
			JiraSummary string `json:"jira_summary"`
		}{
			TicketKey:   resp.TicketKey,
			Category:    resp.Category,
			DraftReply:  resp.DraftReply,
			JiraSummary: resp.JiraSummary,
		})
	}
}
