package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort      string
	MQTTBrokerURL string
	MQTTClientID  string
	MQTTTopic     string
	MQTTDLQTopic  string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	MaxRetries     int
	BaseRetryDelay time.Duration
}

func LoadConfig() Config {
	return Config{
		HTTPPort:      getEnv("HTTP_PORT", "8081"),
		MQTTBrokerURL: getEnv("MQTT_BROKER_URL", "tcp://broker:1883"),
		MQTTClientID:  getEnv("MQTT_CLIENT_ID", "alarm-service"),
		MQTTTopic:     getEnv("MQTT_TOPIC", "alarms/events"),
		MQTTDLQTopic:  getEnv("MQTT_DLQ_TOPIC", "alarms/events/dlq"),

		DBHost:     getEnv("DB_HOST", "database"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "admin"),
		DBPassword: getEnv("DB_PASSWORD", "password123"),
		DBName:     getEnv("DB_NAME", "alarm_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		MaxRetries:     getEnvInt("DB_MAX_RETRIES", 3),
		BaseRetryDelay: time.Duration(getEnvInt("DB_RETRY_DELAY_MS", 100)) * time.Millisecond,
	}
}


func (c Config) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     fmt.Sprintf("%s:%s", c.DBHost, c.DBPort),
		Path:     c.DBName,
		RawQuery: url.Values{"sslmode": {c.DBSSLMode}}.Encode(),
	}
	return u.String()
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
