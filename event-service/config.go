package main

import (
	"os"
	"strconv"
)

// Config concentra tudo que vem do ambiente para evitar espalhar getenv por todo o código.
type Config struct {
	HTTPPort        string
	MQTTBrokerURL   string
	MQTTClientID    string
	MQTTTopic       string
	EventWorkers    int
	EventBufferSize int
}

func LoadConfig() Config {
	return Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		MQTTBrokerURL:   getEnv("MQTT_BROKER_URL", "tcp://broker:1883"),
		MQTTClientID:    getEnv("MQTT_CLIENT_ID", "event-service"),
		MQTTTopic:       getEnv("MQTT_TOPIC", "alarms/events"),
		EventWorkers:    getEnvInt("EVENT_WORKERS", 3),
		EventBufferSize: getEnvInt("EVENT_BUFFER_SIZE", 50),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
