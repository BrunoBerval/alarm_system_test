package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleEvent_MissingFields(t *testing.T) {
	// Payload inválido, sem o campo "type" e com "device_id" vazio
	payload := []byte(`{"device_id": ""}`)
	
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	
	// Chamamos a função do handler que ainda não existe
	HandleEvent(rr, req)

	// Esperamos que a API recuse a requisição com 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("código de status incorreto: obtido %v, esperado %v", status, http.StatusBadRequest)
	}
}