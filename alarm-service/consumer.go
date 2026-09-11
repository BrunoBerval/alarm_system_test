package main

import (
	"encoding/json"
	"log"
)

type EventPayload struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

func ProcessEvent(repo AlarmRepository, payload []byte) {
	var event EventPayload
	
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Erro ao decodificar JSON do broker: %v\n", err)
		return
	}

	if event.Type == "MOTION_DETECTED" { //[cite: 1]
		// Verifica se já existe alarme aberto para garantir a idempotência[cite: 1]
		hasOpen, err := repo.HasOpenAlarm(event.DeviceID)
		if err != nil {
			log.Printf("Erro ao checar alarmes abertos: %v\n", err)
			return
		}

		if hasOpen {
			log.Printf("⚠️ Alarme ignorado: O dispositivo %s já possui um alarme OPEN.\n", event.DeviceID)
			return
		}

		if err := repo.CreateAlarm(event.DeviceID, event.Type); err != nil {
			log.Printf("Erro ao salvar alarme no banco: %v\n", err)
		} else {
			log.Printf("🚨 Alarme criado com sucesso para o dispositivo: %s\n", event.DeviceID)
		}
	}
}