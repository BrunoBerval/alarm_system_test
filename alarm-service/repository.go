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

// AlarmRepository define os métodos necessários para interagir com os alarmes.

type AlarmRepository interface {
	// CreateAlarm retorna created=false quando o insert foi suprimido
	// por já existir um alarme OPEN para o mesmo device. Isso não é erro.
	CreateAlarm(deviceID string, eventType string) (created bool, err error)

	GetAlarms() ([]Alarm, error)

	// CloseAlarm retorna closed=false quando nenhuma linha foi afetada,
	// ou seja, o alarme não existe ou já estava fechado.
	CloseAlarm(id string) (closed bool, err error)
}
