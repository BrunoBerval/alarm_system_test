package main

import (
	"encoding/json"
	"net/http"
)

// EventPayload representa a estrutura que o cliente vai enviar no JSON
type EventPayload struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

func HandleEvent(w http.ResponseWriter, r *http.Request) {
	var payload EventPayload

	// Decodifica o corpo da requisição (JSON) para a nossa struct
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Valida se os campos obrigatórios vieram preenchidos
	if payload.DeviceID == "" || payload.Type == "" {
		http.Error(w, "device_id e type sao obrigatorios", http.StatusBadRequest)
		return
	}

	// Se passou pelas validações, retorna 200 OK (por enquanto)
	w.WriteHeader(http.StatusOK)
}