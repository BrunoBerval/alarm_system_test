package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
)

// Implementação real que envia falhas críticas para um tópico DLQ no Mosquitto
type MQTTDLQPublisher struct {
	Client mqtt.Client
}

func (m *MQTTDLQPublisher) Publish(payload []byte) error {
	// Publica a mensagem de falha no tópico DLQ
	token := m.Client.Publish("alarms/events/dlq", 1, false, payload)
	token.Wait()
	return token.Error()
}

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

	repo := &PostgresRepository{DB: db}

	// 2. Conexão com o MQTT
	opts := mqtt.NewClientOptions().AddBroker("tcp://broker:1883").SetClientID("alarm-service")
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	// Define o callback que será executado quando uma mensagem chegar
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		dlq := &MQTTDLQPublisher{Client: client} // Instancia o publicador DLQ real
		ProcessEvent(repo, dlq, msg.Payload())   // Repassa para a regra de negócio do consumidor
	})

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao conectar no MQTT", "erro", token.Error())
		os.Exit(1)
	}
	slog.Info("Conectado ao Broker MQTT")

	// Assina o tópico
	if token := mqttClient.Subscribe("alarms/events", 1, nil); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao assinar tópico", "erro", token.Error())
		os.Exit(1)
	}
	slog.Info("Aguardando eventos", "topico", "alarms/events")

	// 3. Servidor HTTP (Para listagem e encerramento dos alarmes)
	http.HandleFunc("/alarms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*") // Libera CORS para o frontend React
		
		if r.Method == http.MethodGet {
			alarms, err := repo.GetAlarms()
			if err != nil {
				slog.Error("Erro ao buscar alarmes no banco", "erro", err.Error())
				http.Error(w, `{"error": "erro ao buscar alarmes"}`, http.StatusInternalServerError)
				return
			}
			if alarms == nil {
				alarms = []Alarm{} // Garante que retorne [] ao invés de null
			}
			json.NewEncoder(w).Encode(alarms)
			return
		}
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	})

	http.HandleFunc("/alarms/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "PATCH")

		// Trata rota /alarms/{id}/close
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
	})

	slog.Info("Alarm Service rodando", "porta", 8081)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		slog.Error("Erro ao iniciar servidor HTTP", "erro", err.Error())
		os.Exit(1)
	}
}