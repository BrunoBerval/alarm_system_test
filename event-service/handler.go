package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type EventPayload struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

type EventPublisher interface {
	Publish(topic string, payload []byte) error
}

type EventHandler struct {
	Publisher   EventPublisher
	EventChannel chan []byte
}

func NewEventHandler(publisher EventPublisher, bufferSize int) *EventHandler {
	return &EventHandler{
		Publisher:    publisher,
		EventChannel: make(chan []byte, bufferSize),
	}
}

// StartWorkers inicia o pool de goroutines em background
func (h *EventHandler) StartWorkers(numWorkers int) {
	for i := 1; i <= numWorkers; i++ {
		go func(workerID int) {
			for msgBytes := range h.EventChannel {
				var payload EventPayload
				json.Unmarshal(msgBytes, &payload)

				if err := h.Publisher.Publish("alarms/events", msgBytes); err != nil {
					slog.Error("Worker falhou ao publicar evento", "worker", workerID, "erro", err.Error(), "device_id", payload.DeviceID)
				} else {
					slog.Info("Worker processou e publicou evento", "worker", workerID, "device_id", payload.DeviceID)
				}
			}
		}(i)
	}
}

func (h *EventHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	var payload EventPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("Payload JSON invalido recebido", "erro", err.Error())
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if payload.DeviceID == "" || payload.Type == "" {
		slog.Warn("Requisicao rejeitada", "motivo", "campos obrigatorios ausentes", "device_id", payload.DeviceID)
		http.Error(w, "device_id e type sao obrigatorios", http.StatusBadRequest)
		return
	}

	msgBytes, _ := json.Marshal(payload)

	// Tenta enfileirar a mensagem no canal não-bloqueante (se houver espaço no buffer)
	select {
	case h.EventChannel <- msgBytes:
		slog.Info("Evento enfileirado com sucesso", "device_id", payload.DeviceID)
		w.WriteHeader(http.StatusOK)
	default:
		slog.Warn("Buffer de eventos cheio, requisicao rejeitada por sobrecarga", "device_id", payload.DeviceID)
		http.Error(w, "server overloaded", http.StatusServiceUnavailable)
	}
}