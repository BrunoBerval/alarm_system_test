package main

import (
	"log"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTPublisher é a implementação real que conversa com o Mosquitto
type MQTTPublisher struct {
	Client mqtt.Client
}

func (m *MQTTPublisher) Publish(topic string, payload []byte) error {
	// QoS 1: Garante entrega da mensagem pelo menos uma vez
	token := m.Client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

func main() {
	// O endereço "broker" resolve para o container no docker-compose
	opts := mqtt.NewClientOptions().AddBroker("tcp://broker:1883").SetClientID("event-service")
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Erro ao conectar no MQTT: %v", token.Error())
	}
	log.Println("✅ Conectado ao Broker MQTT")

	// Injeta o cliente real no handler
	publisher := &MQTTPublisher{Client: client}
	eventHandler := &EventHandler{Publisher: publisher}

	// Registra a rota HTTP
	http.HandleFunc("/events", eventHandler.HandleEvent)

	// Inicia o servidor HTTP na porta 8080
	log.Println("🚀 Event Service rodando na porta 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erro ao iniciar servidor HTTP: %v", err)
	}
}