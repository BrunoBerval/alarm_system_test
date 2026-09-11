package main

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTPublisher struct {
	Client mqtt.Client
}

func (m *MQTTPublisher) Publish(topic string, payload []byte) error {
	token := m.Client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Leitura das variáveis de ambiente com valores padrão (fallbacks)
	workersStr := os.Getenv("EVENT_WORKERS")
	workers, err := strconv.Atoi(workersStr)
	if err != nil || workers <= 0 {
		workers = 3 // Padrão seguro
	}

	bufferStr := os.Getenv("EVENT_BUFFER_SIZE")
	bufferSize, err := strconv.Atoi(bufferStr)
	if err != nil || bufferSize <= 0 {
		bufferSize = 50 // Padrão seguro
	}

	opts := mqtt.NewClientOptions().AddBroker("tcp://broker:1883").SetClientID("event-service")
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao conectar no MQTT", "erro", token.Error())
		os.Exit(1)
	}
	slog.Info("Conectado ao Broker MQTT")

	publisher := &MQTTPublisher{Client: client}
	
	// Inicializa o Handler com o Worker Pool e Buffer configurados
	eventHandler := NewEventHandler(publisher, bufferSize)
	eventHandler.StartWorkers(workers)

	slog.Info("Worker Pool inicializado", "workers", workers, "buffer_size", bufferSize)

	http.HandleFunc("/events", eventHandler.HandleEvent)

	slog.Info("Event Service rodando", "porta", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Erro ao iniciar servidor HTTP", "erro", err.Error())
		os.Exit(1)
	}
}