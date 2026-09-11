package main

import (
	"time"
)

// Alarm representa a estrutura da tabela no banco de dados
type Alarm struct {
	ID         string     `json:"id"`
	DeviceID   string     `json:"device_id"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// AlarmRepository define os métodos necessários para interagir com os alarmes
type AlarmRepository interface {
	HasOpenAlarm(deviceID string) (bool, error)
	CreateAlarm(deviceID string, eventType string) error
	GetAlarms() ([]Alarm, error)
	CloseAlarm(id string) error
}