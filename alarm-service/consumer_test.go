package main

import (
	"errors"
	"testing"
	"time"
)

type MockRepository struct {
	SavedAlarms   int
	AttemptsCount int
	FailCount     int  // quantas chamadas devem retornar erro antes de passar
	Duplicate     bool // simula o índice único suprimindo o insert
}

func (m *MockRepository) CreateAlarm(deviceID, eventType string) (bool, error) {
	m.AttemptsCount++
	if m.FailCount > 0 {
		m.FailCount--
		return false, errors.New("erro simulado no banco")
	}
	if m.Duplicate {
		return false, nil
	}
	m.SavedAlarms++
	return true, nil
}

func (m *MockRepository) GetAlarms() ([]Alarm, error)        { return nil, nil }
func (m *MockRepository) CloseAlarm(id string) (bool, error) { return true, nil }

type MockDLQ struct {
	PublishedMessages int
	ShouldFail        bool
}

func (m *MockDLQ) Publish(payload []byte) error {
	if m.ShouldFail {
		return errors.New("dlq indisponivel")
	}
	m.PublishedMessages++
	return nil
}

func novoProcessor(repo AlarmRepository, dlq DLQPublisher) *Processor {
	return NewProcessor(repo, dlq, 3, time.Millisecond)
}

const payloadValido = `{"device_id": "sensor-001", "type": "MOTION_DETECTED"}`

func TestProcessEvent_Success(t *testing.T) {
	repo := &MockRepository{}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(payloadValido))

	if repo.SavedAlarms != 1 {
		t.Errorf("esperava 1 alarme salvo, obteve %d", repo.SavedAlarms)
	}
	if dlq.PublishedMessages != 0 {
		t.Errorf("nao deveria usar a DLQ")
	}
}

func TestProcessEvent_SuccessAfterRetries(t *testing.T) {
	repo := &MockRepository{FailCount: 2}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(payloadValido))

	if repo.AttemptsCount != 3 {
		t.Errorf("esperava 3 tentativas, obteve %d", repo.AttemptsCount)
	}
	if repo.SavedAlarms != 1 {
		t.Errorf("esperava salvar na 3a tentativa, salvou %d", repo.SavedAlarms)
	}
	if dlq.PublishedMessages != 0 {
		t.Errorf("nao deveria ir para DLQ")
	}
}

func TestProcessEvent_FailsAndGoesToDLQ(t *testing.T) {
	repo := &MockRepository{FailCount: 5}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(payloadValido))

	if repo.AttemptsCount != 3 {
		t.Errorf("esperava desistir apos 3 tentativas, tentou %d", repo.AttemptsCount)
	}
	if repo.SavedAlarms != 0 {
		t.Errorf("nao deveria salvar nada")
	}
	if dlq.PublishedMessages != 1 {
		t.Errorf("esperava 1 mensagem na DLQ, obteve %d", dlq.PublishedMessages)
	}
}

// Duplicata suprimida pelo índice único não é erro e não vai para a DLQ.
func TestProcessEvent_DuplicadoNaoViraErro(t *testing.T) {
	repo := &MockRepository{Duplicate: true}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(payloadValido))

	if repo.AttemptsCount != 1 {
		t.Errorf("duplicata nao deveria gerar retry, tentou %d vezes", repo.AttemptsCount)
	}
	if repo.SavedAlarms != 0 {
		t.Errorf("nao deveria contar como salvo")
	}
	if dlq.PublishedMessages != 0 {
		t.Errorf("duplicata nao e falha, nao deveria ir para DLQ")
	}
}


func TestProcessEvent_JSONInvalidoVaiParaDLQ(t *testing.T) {
	repo := &MockRepository{}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(`{quebrado`))

	if repo.AttemptsCount != 0 {
		t.Errorf("nao deveria tocar no banco")
	}
	if dlq.PublishedMessages != 1 {
		t.Errorf("esperava 1 mensagem na DLQ, obteve %d", dlq.PublishedMessages)
	}
}

func TestProcessEvent_CamposObrigatoriosVaoParaDLQ(t *testing.T) {
	repo := &MockRepository{}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(`{"device_id": "", "type": "MOTION_DETECTED"}`))

	if repo.AttemptsCount != 0 {
		t.Errorf("nao deveria tocar no banco")
	}
	if dlq.PublishedMessages != 1 {
		t.Errorf("esperava 1 mensagem na DLQ, obteve %d", dlq.PublishedMessages)
	}
}

func TestProcessEvent_TipoDesconhecidoNaoGeraAlarme(t *testing.T) {
	repo := &MockRepository{}
	dlq := &MockDLQ{}

	novoProcessor(repo, dlq).ProcessEvent([]byte(`{"device_id": "sensor-001", "type": "HEARTBEAT"}`))

	if repo.AttemptsCount != 0 {
		t.Errorf("nao deveria criar alarme para tipo desconhecido")
	}
	if dlq.PublishedMessages != 0 {
		t.Errorf("tipo desconhecido e valido, nao e caso de DLQ")
	}
}

// A falha da própria DLQ não pode derrubar o processamento.
func TestProcessEvent_DLQIndisponivelNaoQuebra(t *testing.T) {
	repo := &MockRepository{FailCount: 5}
	dlq := &MockDLQ{ShouldFail: true}

	novoProcessor(repo, dlq).ProcessEvent([]byte(payloadValido))

	if dlq.PublishedMessages != 0 {
		t.Errorf("DLQ falhou, nao deveria contabilizar publicacao")
	}
}
