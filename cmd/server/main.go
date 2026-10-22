package main

import (
	"log"
	"net/http"
	"time"

	"github.com/WOLFraMix/support-bot-proxy/internal/config"
	"github.com/WOLFraMix/support-bot-proxy/internal/handler"
	"github.com/WOLFraMix/support-bot-proxy/internal/llm"
	"github.com/WOLFraMix/support-bot-proxy/internal/metrics"
	"github.com/WOLFraMix/support-bot-proxy/internal/ticket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 1. Загрузка конфига из переменных окружения
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Регистрируем метрики ДО запуска сервера
	metrics.Register()

	// 2. Инициализация зависимостей
	llmClient := llm.NewYandexClient(cfg)
	ticketService := ticket.NewService(llmClient)

	// 3. Создаём хендлер с зависимостями
	h := handler.NewHandler(ticketService)

	// 4. Маршруты
	mux := http.NewServeMux()

	// Health check — для Docker healthcheck и балансировщиков
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Основной эндпоинт (например, от n8n)
	mux.Handle("/api/v1/support", h.SupportHandler())

	// Метрики Prometheus
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":" + cfg.Port
	log.Printf("starting server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
