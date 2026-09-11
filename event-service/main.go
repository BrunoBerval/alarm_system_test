package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTPublisher struct {
	Client mqtt.Client
}

func (m *MQTTPublisher) Publish(topic string, payload []byte) error {
	token := m.Client.Publish(topic, 1, false, payload)
	// WaitTimeout no lugar de Wait: sem prazo, um broker travado prendia
	// o worker indefinidamente e o retry nunca chegava a acontecer.
	if !token.WaitTimeout(5 * time.Second) {
		return errors.New("timeout ao publicar no broker")
	}
	return token.Error()
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := LoadConfig()

	client := mustConnectMQTT(cfg)
	defer client.Disconnect(250)

	publisher := &MQTTPublisher{Client: client}

	eventHandler := NewEventHandler(publisher, cfg.MQTTTopic, cfg.EventBufferSize)
	eventHandler.StartWorkers(cfg.EventWorkers)
	slog.Info("Worker Pool inicializado",
		"workers", cfg.EventWorkers, "buffer_size", cfg.EventBufferSize)

	mux := http.NewServeMux()
	// Padrão com método: um GET /events agora devolve 405 automaticamente,
	// em vez de cair no handler e falhar tentando ler um corpo vazio.
	mux.HandleFunc("POST /events", eventHandler.HandleEvent)
	mux.HandleFunc("GET /health", healthHandler(client))

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           withCORS(cfg.CORSAllowedOrigins, mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("Event Service rodando", "porta", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Erro ao iniciar servidor HTTP", "erro", err.Error())
			os.Exit(1)
		}
	}()

	waitForShutdown(srv, eventHandler)
}

func mustConnectMQTT(cfg Config) mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBrokerURL).
		SetClientID(cfg.MQTTClientID).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second)

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		slog.Error("Conexao com o broker perdida", "erro", err.Error())
	})
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		slog.Info("Conectado ao Broker MQTT", "broker", cfg.MQTTBrokerURL)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao conectar no MQTT", "erro", token.Error())
		os.Exit(1)
	}
	return client
}

// healthHandler reporta o estado real do serviço.
// Um endpoint que devolve "UP" fixo deixaria o container marcado como
// healthy mesmo com o broker fora do ar.
func healthHandler(client mqtt.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !client.IsConnected() {
			slog.Warn("Health check falhou", "motivo", "MQTT desconectado")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "DOWN", "mqtt": "down"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "UP", "mqtt": "up"})
	}
}

func waitForShutdown(srv *http.Server, h *EventHandler) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Encerrando Event Service")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Para de aceitar requisições primeiro, depois drena o buffer.
	// Na ordem inversa, eventos aceitos durante o shutdown seriam perdidos.
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Erro no shutdown do servidor HTTP", "erro", err.Error())
	}

	h.Shutdown()
	slog.Info("Buffer de eventos drenado")
}
