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

// Processor guarda as dependências do consumo, evitando estado global.
type Processor struct {
	Repo           AlarmRepository
	DLQ            DLQPublisher
	MaxRetries     int
	BaseRetryDelay time.Duration
}

func NewProcessor(repo AlarmRepository, dlq DLQPublisher, maxRetries int, baseDelay time.Duration) *Processor {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}
	return &Processor{
		Repo:           repo,
		DLQ:            dlq,
		MaxRetries:     maxRetries,
		BaseRetryDelay: baseDelay,
	}
}

func (p *Processor) ProcessEvent(payload []byte) {
	var event EventPayload

	// Mensagem ilegível nunca vai processar, por mais que se tente.
	// Reprocessar seria loop infinito, então vai direto para a DLQ.
	if err := json.Unmarshal(payload, &event); err != nil {
		slog.Error("Payload ilegivel vindo do broker", "erro", err.Error(), "payload", string(payload))
		p.toDLQ(payload, "invalid_json")
		return
	}

	if event.DeviceID == "" || event.Type == "" {
		slog.Error("Evento sem campos obrigatorios", "payload", string(payload))
		p.toDLQ(payload, "missing_fields")
		return
	}

	if event.Type != "MOTION_DETECTED" {
		slog.Info("Evento ignorado", "motivo", "tipo nao gera alarme", "type", event.Type, "device_id", event.DeviceID)
		return
	}

	for attempt := 1; attempt <= p.MaxRetries; attempt++ {
		created, err := p.Repo.CreateAlarm(event.DeviceID, event.Type)

		if err == nil {
			if created {
				slog.Info("Alarme criado com sucesso", "device_id", event.DeviceID)
			} else {
				// O índice único barrou: já existe alarme OPEN para o device.
				slog.Info("Evento duplicado ignorado",
					"motivo", "dispositivo ja possui alarme OPEN", "device_id", event.DeviceID)
			}
			return
		}

		slog.Warn("Falha ao salvar no banco",
			"tentativa", attempt, "erro", err.Error(), "device_id", event.DeviceID)

		if attempt < p.MaxRetries {
			time.Sleep(p.BaseRetryDelay * time.Duration(attempt))
		}
	}

	slog.Error("Esgotadas as tentativas de persistencia", "device_id", event.DeviceID)
	p.toDLQ(payload, "db_failure")
}

func (p *Processor) toDLQ(payload []byte, motivo string) {
	if err := p.DLQ.Publish(payload); err != nil {
		// Se nem a DLQ funciona, o log estruturado é o último registro do evento.
		slog.Error("Falha ao enviar para DLQ",
			"erro", err.Error(), "motivo_original", motivo, "payload", string(payload))
		return
	}
	slog.Warn("Evento enviado para DLQ", "motivo", motivo, "payload", string(payload))
}
