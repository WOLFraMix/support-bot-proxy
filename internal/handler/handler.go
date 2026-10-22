package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/metrics"
	"github.com/WOLFraMix/support-bot-proxy/internal/ticket"
)

type Handler struct {
	service *ticket.Service
}

func NewHandler(service *ticket.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SupportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handlerName := "support"

		if r.Method != http.MethodPost {
			metrics.SupportRequestsTotal.WithLabelValues(handlerName, "405").Inc()
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ticket.SupportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			metrics.SupportRequestsTotal.WithLabelValues(handlerName, "400").Inc()
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		resp, err := h.service.Process(r.Context(), req)
		duration := time.Since(start).Seconds()

		if err != nil {
			// Считаем как ошибку
			metrics.SupportRequestsTotal.WithLabelValues(handlerName, "500").Inc()
			metrics.SupportRequestDuration.WithLabelValues(handlerName).Observe(duration)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Успешный запрос
		metrics.SupportRequestsTotal.WithLabelValues(handlerName, "200").Inc()
		metrics.SupportRequestDuration.WithLabelValues(handlerName).Observe(duration)

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
