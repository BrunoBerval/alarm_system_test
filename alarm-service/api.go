package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)


type AlarmAPI struct {
	Repo AlarmRepository
	DB   *sql.DB
	MQTT mqtt.Client
}

var uuidRegex = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)


func (a *AlarmAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /alarms", a.GetAlarms)
	mux.HandleFunc("PATCH /alarms/{id}/close", a.CloseAlarm)
	mux.HandleFunc("GET /health", a.Health)
}

// GetAlarms godoc
// @Summary Lista todos os alarmes
// @Description Retorna a lista completa de alarmes (abertos e fechados)
// @Tags alarms
// @Produce json
// @Success 200 {array} Alarm
// @Failure 500 {object} map[string]string
// @Router /alarms [get]
func (a *AlarmAPI) GetAlarms(w http.ResponseWriter, r *http.Request) {
	alarms, err := a.Repo.GetAlarms()
	if err != nil {
		slog.Error("Erro ao buscar alarmes no banco", "erro", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao buscar alarmes"})
		return
	}
	writeJSON(w, http.StatusOK, alarms)
}

// CloseAlarm godoc
// @Summary Desliga um alarme
// @Description Altera o status do alarme para CLOSED e preenche a data de encerramento
// @Tags alarms
// @Produce json
// @Param id path string true "UUID do Alarme"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/{id}/close [patch]
func (a *AlarmAPI) CloseAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !uuidRegex.MatchString(id) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id deve ser um UUID valido"})
		return
	}

	closed, err := a.Repo.CloseAlarm(id)
	if err != nil {
		slog.Error("Erro ao fechar alarme no banco", "erro", err.Error(), "alarm_id", id)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao fechar alarme"})
		return
	}


	if !closed {
		slog.Warn("Tentativa de fechar alarme inexistente ou ja fechado", "alarm_id", id)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alarme nao encontrado ou ja fechado"})
		return
	}

	slog.Info("Alarme fechado via API", "alarm_id", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed", "id": id})
}

// Health checa as dependências externas 
func (a *AlarmAPI) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbUp := a.DB.PingContext(ctx) == nil
	mqttUp := a.MQTT.IsConnected()

	body := map[string]string{
		"status":   "UP",
		"database": state(dbUp),
		"mqtt":     state(mqttUp),
	}

	if !dbUp || !mqttUp {
		body["status"] = "DOWN"
		slog.Warn("Health check falhou", "database", state(dbUp), "mqtt", state(mqttUp))
		writeJSON(w, http.StatusServiceUnavailable, body)
		return
	}

	writeJSON(w, http.StatusOK, body)
}

func state(up bool) string {
	if up {
		return "up"
	}
	return "down"
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("Erro ao serializar resposta", "erro", err.Error())
	}
}
