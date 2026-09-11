package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "alarm-service/docs"
)

// MQTTDLQPublisher envia falhas críticas para o tópico de DLQ.
//
// MQTT não é fila, é pub/sub. Sem retenção, porém a última falha fica disponível para quem assinar o tópico depois.
// É uma limitação  só a última mensagem por tópico é preservada.
type MQTTDLQPublisher struct {
	Client mqtt.Client
	Topic  string
}

func (m *MQTTDLQPublisher) Publish(payload []byte) error {
	token := m.Client.Publish(m.Topic, 1, true, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return errors.New("timeout ao publicar na DLQ")
	}
	return token.Error()
}

// @title Alarm System API
// @version 1.0
// @description API para gerenciamento do ciclo de vida dos alarmes IoT.
// @BasePath /
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := LoadConfig()

	db := mustConnectDB(cfg)
	defer db.Close()

	repo := &PostgresRepository{DB: db}

	mqttClient := mustConnectMQTT(cfg, repo)
	defer mqttClient.Disconnect(250)

	api := &AlarmAPI{Repo: repo, DB: db, MQTT: mqttClient}

	mux := http.NewServeMux()
	api.Routes(mux)
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("Alarm Service rodando", "porta", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Erro ao iniciar servidor HTTP", "erro", err.Error())
			os.Exit(1)
		}
	}()

	waitForShutdown(srv)
}

func mustConnectDB(cfg Config) *sql.DB {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		slog.Error("Erro ao inicializar conexao com o banco", "erro", err.Error())
		os.Exit(1)
	}

	// Sem esses limites, o pool cresce sem teto e pode estourar o
	// max_connections do Postgres sob carga.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("Banco de dados nao esta respondendo", "erro", err.Error())
		os.Exit(1)
	}

	slog.Info("Conectado ao PostgreSQL", "host", cfg.DBHost, "database", cfg.DBName)
	return db
}

func mustConnectMQTT(cfg Config, repo AlarmRepository) mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBrokerURL).
		SetClientID(cfg.MQTTClientID).
		// KeepAlive de 2s com PingTimeout de 1s fazia o cliente derrubar a
		// conexão a cada oscilação de rede maior que 1 segundo, gerando
		// reconexões em cascata. 30s/10s é o intervalo usual.
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		// CleanSession false + ClientID fixo: o broker guarda a assinatura e as
		// mensagens QoS 1 publicadas enquanto o serviço estava fora do ar.
		// Com CleanSession true, tudo publicado durante um restart era perdido.
		SetCleanSession(false)

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		slog.Error("Conexao com o broker perdida", "erro", err.Error())
	})

	// A assinatura vai dentro do OnConnect.
	// Registrando aqui, toda reconexão restabelece a assinatura.
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		slog.Info("Conectado ao Broker MQTT", "broker", cfg.MQTTBrokerURL)

		dlq := &MQTTDLQPublisher{Client: client, Topic: cfg.MQTTDLQTopic}
		processor := NewProcessor(repo, dlq, cfg.MaxRetries, cfg.BaseRetryDelay)

		handler := func(_ mqtt.Client, msg mqtt.Message) {
			processor.ProcessEvent(msg.Payload())
		}

		// Handler explícito no Subscribe em vez do DefaultPublishHandler:
		// deixa claro qual função trata qual tópico.
		token := client.Subscribe(cfg.MQTTTopic, 1, handler)
		if token.WaitTimeout(10*time.Second) && token.Error() != nil {
			slog.Error("Erro ao assinar topico", "topico", cfg.MQTTTopic, "erro", token.Error())
			return
		}
		slog.Info("Aguardando eventos", "topico", cfg.MQTTTopic)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Erro ao conectar no MQTT", "erro", token.Error())
		os.Exit(1)
	}
	return client
}

func waitForShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Encerrando Alarm Service")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Erro no shutdown do servidor HTTP", "erro", err.Error())
	}
}
