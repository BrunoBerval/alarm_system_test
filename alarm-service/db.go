package main

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	DB *sql.DB
}

func (r *PostgresRepository) HasOpenAlarm(deviceID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM alarms WHERE device_id=$1 AND status='OPEN')`
	err := r.DB.QueryRow(query, deviceID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) CreateAlarm(deviceID, eventType string) error {
	query := `INSERT INTO alarms (device_id, type) VALUES ($1, $2)`
	_, err := r.DB.Exec(query, deviceID, eventType)
	return err
}

func (r *PostgresRepository) GetAlarms() ([]Alarm, error) {
	query := `SELECT id, device_id, type, status, created_at, finished_at FROM alarms ORDER BY created_at DESC`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alarms []Alarm
	for rows.Next() {
		var a Alarm
		var finishedAt sql.NullTime

		err := rows.Scan(&a.ID, &a.DeviceID, &a.Type, &a.Status, &a.CreatedAt, &finishedAt)
		if err != nil {
			return nil, err
		}

		if finishedAt.Valid {
			a.FinishedAt = &finishedAt.Time
		}

		alarms = append(alarms, a)
	}
	return alarms, nil
}

func (r *PostgresRepository) CloseAlarm(id string) error {
	query := `UPDATE alarms SET status='CLOSED', finished_at=$1 WHERE id=$2`
	_, err := r.DB.Exec(query, time.Now(), id)
	return err
}