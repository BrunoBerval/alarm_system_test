package main

import (
	"encoding/json"
	"log/slog"
	"time"
)

type EventPayload struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

// Interface para isolarmos o envio para DLQ
type DLQPublisher interface {
	Publish(payload []byte) error
}

var baseRetryDelay = time.Second // Variável para podermos alterar no teste

func ProcessEvent(repo AlarmRepository, dlq DLQPublisher, payload []byte) {
	var event EventPayload

	if err := json.Unmarshal(payload, &event); err != nil {
		slog.Error("Erro ao decodificar JSON do broker", "erro", err.Error())
		return
	}

	if event.Type == "MOTION_DETECTED" {
		hasOpen, err := repo.HasOpenAlarm(event.DeviceID)
		if err != nil {
			slog.Error("Erro ao checar alarmes abertos", "erro", err.Error(), "device_id", event.DeviceID)
			return
		}

		if hasOpen {
			slog.Warn("Alarme ignorado", "motivo", "dispositivo já possui alarme OPEN", "device_id", event.DeviceID)
			return
		}

		// Estratégia de Retry Local
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := repo.CreateAlarm(event.DeviceID, event.Type)
			if err == nil {
				slog.Info("Alarme criado com sucesso", "device_id", event.DeviceID)
				return // Sucesso, sai da função
			}

			slog.Error("Falha ao salvar no banco", "tentativa", attempt, "erro", err.Error(), "device_id", event.DeviceID)
			
			if attempt < maxRetries {
				time.Sleep(baseRetryDelay * time.Duration(attempt)) // Exponential backoff simples
			}
		}

		// Se chegou aqui, esgotou as tentativas. Envia para DLQ.
		slog.Error("Esgotadas as tentativas", "acao", "enviando para DLQ", "device_id", event.DeviceID)
		if err := dlq.Publish(payload); err != nil {
			slog.Error("Erro fatal ao enviar para DLQ", "erro", err.Error(), "device_id", event.DeviceID)
		}
	}
}