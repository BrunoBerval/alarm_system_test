package main

import (
	"errors"
	"testing"
	"time"
)

// Diminui o tempo de espera no teste para não ficar lento
func init() {
	baseRetryDelay = time.Millisecond
}

// Mock do Banco de Dados
type MockRepository struct {
	SavedAlarms     int
	MockHasOpen     bool
	FailCreateCount int // Quantas vezes deve simular erro antes de acertar
	AttemptsCount   int // Conta quantas vezes o CreateAlarm foi chamado
}

func (m *MockRepository) HasOpenAlarm(deviceID string) (bool, error) {
	return m.MockHasOpen, nil
}

func (m *MockRepository) CreateAlarm(deviceID, eventType string) error {
	m.AttemptsCount++
	if m.FailCreateCount > 0 {
		m.FailCreateCount--
		return errors.New("erro simulado no banco")
	}
	m.SavedAlarms++
	return nil
}

func (m *MockRepository) GetAlarms() ([]Alarm, error) { return nil, nil }
func (m *MockRepository) CloseAlarm(id string) error  { return nil }

// Mock do Publicador DLQ
type MockDLQ struct {
	PublishedMessages int
}

func (m *MockDLQ) Publish(payload []byte) error {
	m.PublishedMessages++
	return nil
}

// 1. Testa Sucesso de primeira
func TestProcessEvent_Success(t *testing.T) {
	mockRepo := &MockRepository{MockHasOpen: false}
	mockDLQ := &MockDLQ{}
	payload := []byte(`{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`)

	ProcessEvent(mockRepo, mockDLQ, payload)

	if mockRepo.SavedAlarms != 1 {
		t.Errorf("Esperava 1 alarme salvo, mas obteve %d", mockRepo.SavedAlarms)
	}
}

// 2. Testa Sucesso após 2 falhas (Retry)
func TestProcessEvent_SuccessAfterRetries(t *testing.T) {
	mockRepo := &MockRepository{MockHasOpen: false, FailCreateCount: 2}
	mockDLQ := &MockDLQ{}
	payload := []byte(`{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`)

	ProcessEvent(mockRepo, mockDLQ, payload)

	if mockRepo.AttemptsCount != 3 {
		t.Errorf("Esperava 3 tentativas no banco, mas obteve %d", mockRepo.AttemptsCount)
	}
	if mockRepo.SavedAlarms != 1 {
		t.Errorf("Esperava que salvasse na 3ª tentativa, mas salvou %d", mockRepo.SavedAlarms)
	}
	if mockDLQ.PublishedMessages != 0 {
		t.Errorf("Não deveria ir para DLQ")
	}
}

// 3. Testa Falha total indo para DLQ
func TestProcessEvent_FailsAndGoesToDLQ(t *testing.T) {
	// Vai falhar sempre (configuramos para falhar 5 vezes, ou seja, excede as 3 tentativas)
	mockRepo := &MockRepository{MockHasOpen: false, FailCreateCount: 5}
	mockDLQ := &MockDLQ{}
	payload := []byte(`{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`)

	ProcessEvent(mockRepo, mockDLQ, payload)

	if mockRepo.AttemptsCount != 3 {
		t.Errorf("Esperava desistir após 3 tentativas, mas tentou %d", mockRepo.AttemptsCount)
	}
	if mockRepo.SavedAlarms != 0 {
		t.Errorf("Não deveria salvar nada")
	}
	if mockDLQ.PublishedMessages != 1 {
		t.Errorf("Esperava 1 mensagem na DLQ, obteve %d", mockDLQ.PublishedMessages)
	}
}