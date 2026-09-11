package main

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

const queryTimeout = 3 * time.Second

type PostgresRepository struct {
	DB *sql.DB
}

// CreateAlarm delega a deduplicação ao índice único parcial
// uniq_open_alarm_per_device. Se já existir um alarme OPEN para o device,
// o ON CONFLICT suprime o insert e RowsAffected volta 0.

func (r *PostgresRepository) CreateAlarm(deviceID, eventType string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	query := `INSERT INTO alarms (device_id, type)
	          VALUES ($1, $2)
	          ON CONFLICT DO NOTHING`

	res, err := r.DB.ExecContext(ctx, query, deviceID, eventType)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *PostgresRepository) GetAlarms() ([]Alarm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	query := `SELECT id, device_id, type, status, created_at, finished_at
	          FROM alarms
	          ORDER BY created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alarms := []Alarm{}
	for rows.Next() {
		var a Alarm
		var finishedAt sql.NullTime

		if err := rows.Scan(&a.ID, &a.DeviceID, &a.Type, &a.Status, &a.CreatedAt, &finishedAt); err != nil {
			return nil, err
		}

		if finishedAt.Valid {
			a.FinishedAt = &finishedAt.Time
		}

		alarms = append(alarms, a)
	}

	// rows.Err() captura falhas ocorridas no meio da iteração,
	// que o laço acima não detecta sozinho.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alarms, nil
}

// CloseAlarm só afeta alarmes que estão OPEN, o que torna a operação
// idempotente: fechar duas vezes não sobrescreve o finished_at original.
// O retorno false indica que nada foi alterado (id inexistente ou já fechado).
func (r *PostgresRepository) CloseAlarm(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	// NOW() é do banco, não do relógio da aplicação: evita divergência de
	// horário entre o container do serviço e o do Postgres.
	query := `UPDATE alarms
	          SET status = 'CLOSED', finished_at = NOW()
	          WHERE id = $1 AND status = 'OPEN'`

	res, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}
