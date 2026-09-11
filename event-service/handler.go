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
	Publisher EventPublisher
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

	if err := h.Publisher.Publish("alarms/events", msgBytes); err != nil {
		slog.Error("Erro ao publicar evento no broker", "erro", err.Error(), "device_id", payload.DeviceID)
		http.Error(w, "erro ao publicar evento", http.StatusInternalServerError)
		return
	}

	slog.Info("Evento publicado com sucesso", "device_id", payload.DeviceID, "type", payload.Type)
	w.WriteHeader(http.StatusOK)
}