package main

import (
	"testing"
)

// MockRepository simula o banco de dados para os testes
type MockRepository struct {
	SavedAlarms int
	MockHasOpen bool
}

func (m *MockRepository) HasOpenAlarm(deviceID string) (bool, error) {
	return m.MockHasOpen, nil
}

func (m *MockRepository) CreateAlarm(deviceID, eventType string) error {
	m.SavedAlarms++
	return nil
}

func (m *MockRepository) GetAlarms() ([]Alarm, error) {
	return nil, nil
}

func (m *MockRepository) CloseAlarm(id string) error {
	return nil
}

func TestProcessEvent_MotionDetected_NewAlarm(t *testing.T) {
	mockRepo := &MockRepository{MockHasOpen: false}
	payload := []byte(`{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`) //[cite: 1]
	
	ProcessEvent(mockRepo, payload)

	if mockRepo.SavedAlarms != 1 {
		t.Errorf("Esperava 1 alarme salvo, mas obteve %d", mockRepo.SavedAlarms)
	}
}

func TestProcessEvent_MotionDetected_Duplicate(t *testing.T) {
	// Simulamos que já existe um alarme OPEN no banco
	mockRepo := &MockRepository{MockHasOpen: true}
	payload := []byte(`{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`) //[cite: 1]
	
	ProcessEvent(mockRepo, payload)

	if mockRepo.SavedAlarms != 0 {
		t.Errorf("Não esperava criar alarme duplicado, mas obteve %d", mockRepo.SavedAlarms)
	}
}

func TestProcessEvent_IgnoredEvent(t *testing.T) {
	mockRepo := &MockRepository{MockHasOpen: false}
	payload := []byte(`{"device_id": "sensor-001", "type": "TEMPERATURE_HIGH"}`)
	
	ProcessEvent(mockRepo, payload)

	if mockRepo.SavedAlarms != 0 {
		t.Errorf("Não esperava nenhum alarme salvo, mas obteve %d", mockRepo.SavedAlarms)
	}
}