package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// muxDeTeste reproduz o roteamento real: padrões com método do Go 1.22.
// É o que torna o preflight um problema, então o teste precisa usar o mesmo.
func muxDeTeste() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}

func requisicao(t *testing.T, h http.Handler, metodo, rota string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, rota, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// Sem o middleware, o mux devolve 405 para OPTIONS. Este teste registra
// esse comportamento para deixar claro o que o middleware resolve.
func TestSemMiddleware_PreflightCaiEm405(t *testing.T) {
	rr := requisicao(t, muxDeTeste(), http.MethodOptions, "/events", nil)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("esperava 405 do mux cru, obteve %d", rr.Code)
	}
}

func TestPreflight_Responde204ComHeaders(t *testing.T) {
	h := withCORS("*", muxDeTeste())

	rr := requisicao(t, h, http.MethodOptions, "/events", map[string]string{
		"Origin":                         "http://localhost:5173",
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "content-type",
	})

	if rr.Code != http.StatusNoContent {
		t.Fatalf("esperava 204 no preflight, obteve %d", rr.Code)
	}

	esperados := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": allowedMethods,
		"Access-Control-Allow-Headers": "Content-Type",
		"Access-Control-Max-Age":       "600",
	}
	for header, valor := range esperados {
		if obtido := rr.Header().Get(header); obtido != valor {
			t.Errorf("%s: esperava %q, obteve %q", header, valor, obtido)
		}
	}
}

func TestRequisicaoReal_PassaPeloMiddleware(t *testing.T) {
	h := withCORS("*", muxDeTeste())

	rr := requisicao(t, h, http.MethodPost, "/events", map[string]string{
		"Origin": "http://localhost:5173",
	})

	if rr.Code != http.StatusAccepted {
		t.Errorf("esperava 202 do handler, obteve %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("resposta real deveria carregar o header de origem")
	}
}

// O header nao pode sair duplicado: o navegador bloqueia quando encontra
// mais de um valor em Access-Control-Allow-Origin.
func TestOrigem_NaoDuplicaHeader(t *testing.T) {
	h := withCORS("*", muxDeTeste())

	rr := requisicao(t, h, http.MethodPost, "/events", map[string]string{
		"Origin": "http://localhost:5173",
	})

	if n := len(rr.Header().Values("Access-Control-Allow-Origin")); n != 1 {
		t.Errorf("esperava 1 valor no header de origem, obteve %d", n)
	}
}

func TestOrigemRestrita_PermiteAListada(t *testing.T) {
	h := withCORS("http://localhost:5173, http://meu-front.local", muxDeTeste())

	rr := requisicao(t, h, http.MethodPost, "/events", map[string]string{
		"Origin": "http://meu-front.local",
	})

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://meu-front.local" {
		t.Errorf("esperava a origem devolvida de volta, obteve %q", got)
	}
	if rr.Header().Get("Vary") != "Origin" {
		t.Errorf("origem restrita precisa de Vary: Origin para nao envenenar cache")
	}
}

func TestOrigemRestrita_BloqueiaNaoListada(t *testing.T) {
	h := withCORS("http://localhost:5173", muxDeTeste())

	rr := requisicao(t, h, http.MethodPost, "/events", map[string]string{
		"Origin": "http://site-invasor.local",
	})

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origem nao listada nao deveria receber header, obteve %q", got)
	}
}

// OPTIONS sem Access-Control-Request-Method nao e preflight: e um OPTIONS
// comum e deve seguir o fluxo normal do mux.
func TestOptionsComum_NaoEhTratadoComoPreflight(t *testing.T) {
	h := withCORS("*", muxDeTeste())

	rr := requisicao(t, h, http.MethodOptions, "/events", nil)

	if rr.Code == http.StatusNoContent {
		t.Errorf("OPTIONS sem Access-Control-Request-Method nao deveria virar 204")
	}
}
