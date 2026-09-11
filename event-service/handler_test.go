package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// MockPublisher precisa ser thread-safe: os workers rodam em goroutines.
type MockPublisher struct {
	mu       sync.Mutex
	count    int
	failNext int // quantas chamadas devem falhar antes de passar
}

func (m *MockPublisher) Publish(topic string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	if m.failNext > 0 {
		m.failNext--
		return errors.New("broker indisponivel (simulado)")
	}
	return nil
}

func (m *MockPublisher) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.count
}

// eventually evita sleep fixo: faz polling ate a condicao valer ou estourar o prazo.
func eventually(t *testing.T, prazo time.Duration, cond func() bool) bool {
	t.Helper()
	limite := time.Now().Add(prazo)
	for time.Now().Before(limite) {
		if cond() {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return cond()
}

func novoHandler(pub EventPublisher, buffer int) *EventHandler {
	h := NewEventHandler(pub, "alarms/events", buffer)
	h.PublishRetryDelay = time.Millisecond
	return h
}

func post(t *testing.T, h *EventHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("POST", "/events", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h.HandleEvent(rr, req)
	return rr
}

func TestHandleEvent_MissingFields(t *testing.T) {
	pub := &MockPublisher{}
	h := novoHandler(pub, 10)
	h.StartWorkers(1)
	defer h.Shutdown()

	rr := post(t, h, `{"device_id": ""}`)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status incorreto: obtido %d, esperado %d", rr.Code, http.StatusBadRequest)
	}
	if pub.Count() != 0 {
		t.Errorf("esperava 0 publicacoes, obteve %d", pub.Count())
	}
}

func TestHandleEvent_InvalidJSON(t *testing.T) {
	pub := &MockPublisher{}
	h := novoHandler(pub, 10)
	h.StartWorkers(1)
	defer h.Shutdown()

	rr := post(t, h, `{isso nao e json`)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status incorreto: obtido %d, esperado %d", rr.Code, http.StatusBadRequest)
	}
}

// Antes falhava: o handler era montado sem EventChannel, o canal ficava nil
// e o select caia no default, devolvendo 503.
func TestHandleEvent_Success(t *testing.T) {
	pub := &MockPublisher{}
	h := novoHandler(pub, 10)
	h.StartWorkers(1)
	defer h.Shutdown()

	rr := post(t, h, `{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`)

	if rr.Code != http.StatusAccepted {
		t.Errorf("status incorreto: obtido %d, esperado %d", rr.Code, http.StatusAccepted)
	}
	if !eventually(t, time.Second, func() bool { return pub.Count() == 1 }) {
		t.Errorf("esperava 1 publicacao, obteve %d", pub.Count())
	}
}

// Buffer cheio e sem worker consumindo: a segunda requisicao deve ser recusada.
func TestHandleEvent_BufferCheio(t *testing.T) {
	pub := &MockPublisher{}
	h := novoHandler(pub, 1) // nenhum worker iniciado de proposito

	if rr := post(t, h, `{"device_id": "s1", "type": "MOTION_DETECTED"}`); rr.Code != http.StatusAccepted {
		t.Fatalf("primeira requisicao deveria caber no buffer, obteve %d", rr.Code)
	}

	rr := post(t, h, `{"device_id": "s2", "type": "MOTION_DETECTED"}`)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status incorreto: obtido %d, esperado %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// O worker retem o evento e tentar de novo quando o broker falha.
func TestWorker_RetentaPublicacao(t *testing.T) {
	pub := &MockPublisher{failNext: 2}
	h := novoHandler(pub, 10)
	h.StartWorkers(1)
	defer h.Shutdown()

	post(t, h, `{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`)

	if !eventually(t, time.Second, func() bool { return pub.Count() == 3 }) {
		t.Errorf("esperava 3 tentativas de publicacao, obteve %d", pub.Count())
	}
}

// Shutdown drena o que sobrou no buffer antes de encerrar.
func TestShutdown_DrenaBuffer(t *testing.T) {
	pub := &MockPublisher{}
	h := novoHandler(pub, 10)

	for i := 0; i < 5; i++ {
		post(t, h, `{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`)
	}

	h.StartWorkers(2)
	h.Shutdown()

	if pub.Count() != 5 {
		t.Errorf("esperava 5 eventos publicados no shutdown, obteve %d", pub.Count())
	}
}
