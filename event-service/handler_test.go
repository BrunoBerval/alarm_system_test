package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockPublisher simula o comportamento do MQTT para os testes
type MockPublisher struct {
	PublishedMessages int
}

func (m *MockPublisher) Publish(topic string, payload []byte) error {
	m.PublishedMessages++
	return nil
}

func TestHandleEvent_MissingFields(t *testing.T) {
	payload := []byte(`{"device_id": ""}`)
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	mockPub := &MockPublisher{}
	handler := &EventHandler{Publisher: mockPub}

	handler.HandleEvent(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("código de status incorreto: obtido %v, esperado %v", status, http.StatusBadRequest)
	}

	// Garante que não publicou nada se o payload estava inválido
	if mockPub.PublishedMessages != 0 {
		t.Errorf("esperava 0 publicações, obteve %d", mockPub.PublishedMessages)
	}
}

func TestHandleEvent_Success(t *testing.T) {
	payload := []byte(`{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`) //[cite: 1]
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	mockPub := &MockPublisher{}
	handler := &EventHandler{Publisher: mockPub}

	handler.HandleEvent(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("código de status incorreto: obtido %v, esperado %v", status, http.StatusOK)
	}

	// Garante que a mensagem foi enviada para o Broker
	if mockPub.PublishedMessages != 1 {
		t.Errorf("esperava 1 publicação, obteve %d", mockPub.PublishedMessages)
	}
}