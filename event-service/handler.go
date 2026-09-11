package main

import (
	"encoding/json"
	"net/http"
)

type EventPayload struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

// EventPublisher é a interface que permite injetar a dependência (Broker Real ou Mock)
type EventPublisher interface {
	Publish(topic string, payload []byte) error
}

// EventHandler carrega as dependências necessárias para a rota
type EventHandler struct {
	Publisher EventPublisher
}

// HandleEvent agora pertence à struct EventHandler
func (h *EventHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	var payload EventPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if payload.DeviceID == "" || payload.Type == "" {
		http.Error(w, "device_id e type sao obrigatorios", http.StatusBadRequest)
		return
	}

	// Converte a struct de volta para JSON para envio ao Broker
	msgBytes, _ := json.Marshal(payload)

	// Publica no tópico "alarms/events"[cite: 1]
	if err := h.Publisher.Publish("alarms/events", msgBytes); err != nil {
		http.Error(w, "erro ao publicar evento", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}