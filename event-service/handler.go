package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type EventPayload struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

type EventPublisher interface {
	Publish(topic string, payload []byte) error
}

type EventHandler struct {
	Publisher    EventPublisher
	Topic        string
	EventChannel chan []byte

	// Retry de publicação, exposto para os testes controlarem o tempo
	MaxPublishRetries int
	PublishRetryDelay time.Duration

	wg sync.WaitGroup
}

func NewEventHandler(publisher EventPublisher, topic string, bufferSize int) *EventHandler {
	return &EventHandler{
		Publisher:         publisher,
		Topic:             topic,
		EventChannel:      make(chan []byte, bufferSize),
		MaxPublishRetries: 3,
		PublishRetryDelay: 100 * time.Millisecond,
	}
}

// StartWorkers inicia o pool de goroutines em background
func (h *EventHandler) StartWorkers(numWorkers int) {
	for i := 1; i <= numWorkers; i++ {
		h.wg.Add(1)
		go func(workerID int) {
			defer h.wg.Done()
			for msgBytes := range h.EventChannel {
				h.publishWithRetry(workerID, msgBytes)
			}
		}(i)
	}
}

// Shutdown fecha o canal e espera os workers drenarem o que sobrou.

func (h *EventHandler) Shutdown() {
	close(h.EventChannel)
	h.wg.Wait()
}

func (h *EventHandler) publishWithRetry(workerID int, msgBytes []byte) {
	var payload EventPayload
	if err := json.Unmarshal(msgBytes, &payload); err != nil {
		slog.Error("Worker recebeu payload ilegivel", "worker", workerID, "erro", err.Error())
		return
	}

	var lastErr error
	for attempt := 1; attempt <= h.MaxPublishRetries; attempt++ {
		lastErr = h.Publisher.Publish(h.Topic, msgBytes)
		if lastErr == nil {
			slog.Info("Evento publicado no broker",
				"worker", workerID, "device_id", payload.DeviceID, "tentativa", attempt)
			return
		}

		slog.Warn("Falha ao publicar no broker",
			"worker", workerID, "tentativa", attempt,
			"erro", lastErr.Error(), "device_id", payload.DeviceID)

		if attempt < h.MaxPublishRetries {
			time.Sleep(h.PublishRetryDelay * time.Duration(attempt))
		}
	}

	// O cliente já recebeu 202. O payload completo vai para o log para
	// que o evento possa ser reprocessado manualmente.
	slog.Error("Evento descartado apos esgotar as tentativas de publicacao",
		"worker", workerID, "device_id", payload.DeviceID,
		"erro", lastErr.Error(), "payload", string(msgBytes))
}

func (h *EventHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload EventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("Payload JSON invalido recebido", "erro", err.Error())
		writeJSONError(w, http.StatusBadRequest, "json invalido")
		return
	}

	if payload.DeviceID == "" || payload.Type == "" {
		slog.Warn("Requisicao rejeitada", "motivo", "campos obrigatorios ausentes", "device_id", payload.DeviceID)
		writeJSONError(w, http.StatusBadRequest, "device_id e type sao obrigatorios")
		return
	}

	msgBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Erro ao serializar evento", "erro", err.Error())
		writeJSONError(w, http.StatusInternalServerError, "erro interno")
		return
	}

	select {
	case h.EventChannel <- msgBytes:
		slog.Info("Evento enfileirado com sucesso", "device_id", payload.DeviceID)
		// 202 e nao 200: a publicacao no broker acontece de forma assincrona,
		// entao neste ponto o evento foi aceito, nao processado.
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	default:
		slog.Warn("Buffer de eventos cheio, requisicao rejeitada por sobrecarga", "device_id", payload.DeviceID)
		writeJSONError(w, http.StatusServiceUnavailable, "servico sobrecarregado, tente novamente")
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
