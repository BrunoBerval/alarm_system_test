package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"

	// O pacote docs gerado pelo Swaggo
	_ "alarm-service/docs"
)

// Implementação real que envia falhas críticas para um tópico DLQ no Mosquitto
type MQTTDLQPublisher struct {
	Client mqtt.Client
}

func (m *MQTTDLQPublisher) Publish(payload []byte) error {
	token := m.Client.Publish("alarms/events/dlq", 1, false, payload)
	token.Wait()
	return token.Error()
}

// Variável global temporária para os handlers acessarem o repositório
var repo AlarmRepository

// healthHandler checa as duas dependências externas do serviço.
// Se o banco ou o broker estiverem fora, responde 503 e o Docker
// marca o container como unhealthy.
func healthHandler(db *sql.DB, client mqtt.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbUp := db.PingContext(ctx) == nil
		mqttUp := client.IsConnected()

		body := map[string]string{
			"status":   "UP",
			"database": state(dbUp),
			"mqtt":     state(mqttUp),
		}

		if !dbUp || !mqttUp {
			body["status"] = "DOWN"
			slog.Warn("Health check falhou", "database", state(dbUp), "mqtt", state(mqttUp))
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(body)
	}
}

func state(up bool) string {
	if up {
		return "up"
	}
	return "down"
}

// @title Alarm System API
// @version 1.0
// @description API para gerenciamento do ciclo de vida dos alarmes IoT.
// @host localhost:8081
// @BasePath /
func main() {
	// Configura o Log Estruturado (JSON) como padrão
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Conexão com o PostgreSQL
	dbConnStr := "postgres://admin:password123@database:5432/alarm_db?sslmode=disable"
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		slog.Error("Erro ao inicializar conexão com o banco", "erro", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("Banco de dados não está respondendo", "erro", err.Error())
		os.Exit(1)
	}
	slog.Info("Conectado ao PostgreSQL")

	repo = &PostgresRepository{DB: db}

	// 2. Conexão com o MQTT
	opts := mqtt.NewClientOptions().AddBroker("tcp://broker:1883").SetClientID("alarm-service")
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		dlq := &MQTTDLQPublisher{Client: client}
		ProcessEvent(repo, dlq, msg.Payload())
	})

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao conectar no MQTT", "erro", token.Error())
		os.Exit(1)
	}
	slog.Info("Conectado ao Broker MQTT")

	if token := mqttClient.Subscribe("alarms/events", 1, nil); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao assinar tópico", "erro", token.Error())
		os.Exit(1)
	}
	slog.Info("Aguardando eventos", "topico", "alarms/events")

	// 3. Rotas da API
	http.HandleFunc("/alarms", getAlarmsHandler)
	http.HandleFunc("/alarms/", closeAlarmHandler)

	// Rota de Health Check
	http.HandleFunc("/health", healthHandler(db, mqttClient))

	// Rota do Swagger UI (com apontamento explícito para corrigir o 404)
	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	slog.Info("Alarm Service rodando", "porta", 8081)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		slog.Error("Erro ao iniciar servidor HTTP", "erro", err.Error())
		os.Exit(1)
	}
}

// getAlarmsHandler godoc
// @Summary Lista todos os alarmes
// @Description Retorna a lista completa de alarmes (abertos e fechados)
// @Tags alarms
// @Produce json
// @Success 200 {array} Alarm
// @Failure 500 {string} string "Erro ao buscar alarmes"
// @Router /alarms [get]
func getAlarmsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		alarms, err := repo.GetAlarms()
		if err != nil {
			slog.Error("Erro ao buscar alarmes no banco", "erro", err.Error())
			http.Error(w, `{"error": "erro ao buscar alarmes"}`, http.StatusInternalServerError)
			return
		}
		if alarms == nil {
			alarms = []Alarm{}
		}
		json.NewEncoder(w).Encode(alarms)
		return
	}
	http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
}

// closeAlarmHandler godoc
// @Summary Desliga um alarme
// @Description Altera o status do alarme para CLOSED e preenche a data de encerramento
// @Tags alarms
// @Param id path string true "UUID do Alarme"
// @Success 200 {string} string "OK"
// @Failure 404 {string} string "Rota não encontrada"
// @Failure 500 {string} string "Erro interno"
// @Router /alarms/{id}/close [patch]
func closeAlarmHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PATCH")

	if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/close") {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 3 {
			id := parts[2]
			if err := repo.CloseAlarm(id); err != nil {
				slog.Error("Erro ao fechar alarme no banco", "erro", err.Error(), "alarm_id", id)
				http.Error(w, `{"error": "erro ao fechar alarme"}`, http.StatusInternalServerError)
				return
			}
			slog.Info("Alarme fechado via API", "alarm_id", id)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	http.Error(w, "Rota não encontrada", http.StatusNotFound)
}