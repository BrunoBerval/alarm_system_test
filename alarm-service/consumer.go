package main

import (
	"encoding/json"
	"log"
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
		log.Printf("Erro ao decodificar JSON do broker: %v\n", err)
		return
	}

	if event.Type == "MOTION_DETECTED" {
		hasOpen, err := repo.HasOpenAlarm(event.DeviceID)
		if err != nil {
			log.Printf("Erro ao checar alarmes abertos: %v\n", err)
			return
		}

		if hasOpen {
			log.Printf("⚠️ Alarme ignorado: O dispositivo %s já possui um alarme OPEN.\n", event.DeviceID)
			return
		}

		// Estratégia de Retry Local
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := repo.CreateAlarm(event.DeviceID, event.Type)
			if err == nil {
				log.Printf("🚨 Alarme criado com sucesso para o dispositivo: %s\n", event.DeviceID)
				return // Sucesso, sai da função
			}

			log.Printf("❌ Falha ao salvar no banco (Tentativa %d/%d): %v\n", attempt, maxRetries, err)
			
			if attempt < maxRetries {
				time.Sleep(baseRetryDelay * time.Duration(attempt)) // Exponential backoff simples
			}
		}

		// Se chegou aqui, esgotou as tentativas. Envia para DLQ.
		log.Printf("💀 Esgotadas as tentativas para o dispositivo %s. Enviando para DLQ...\n", event.DeviceID)
		if err := dlq.Publish(payload); err != nil {
			log.Printf("Erro fatal ao enviar para DLQ: %v\n", err)
		}
	}
}