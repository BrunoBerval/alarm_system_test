package main

import (
	"log/slog"
	"net/http"
	"os"
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
	// Configura o Log Estruturado (JSON) como padrão
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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
	eventHandler := &EventHandler{Publisher: publisher}

	http.HandleFunc("/events", eventHandler.HandleEvent)

	slog.Info("Event Service rodando", "porta", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Erro ao iniciar servidor HTTP", "erro", err.Error())
		os.Exit(1)
	}
}