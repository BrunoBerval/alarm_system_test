package main

import (
	"net/http"
	"strings"
)

// allowedMethods lista os métodos que este serviço realmente atende.
const allowedMethods = "GET, PATCH, OPTIONS"

// withCORS envolve o mux inteiro, e isso é obrigatório e não opcional.
//
// Um PATCH nunca é uma requisição simples, então o navegador manda um
// OPTIONS de preflight antes. Como o mux usa os padrões com método do
// Go 1.22 ("PATCH /alarms/{id}/close"), um OPTIONS que chegasse até ele
// cairia em 405, o preflight falharia e a requisição real nunca sairia.
// O middleware intercepta antes do roteamento.
func withCORS(allowedOrigins string, next http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := resolveOrigin(origins, r.Header.Get("Origin")); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if origin != "*" {
				// Sem o Vary, um cache intermediário pode servir a resposta
				// de uma origem para outra.
				w.Header().Add("Vary", "Origin")
			}
		}

		if isPreflight(r) {
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			// Sem o Max-Age o navegador refaz o preflight a cada requisição,
			// dobrando o número de chamadas.
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func parseOrigins(raw string) []string {
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// resolveOrigin devolve o valor que deve ir no header, ou "" quando a
// origem não é permitida. Devolver a origem recebida (e não a lista)
// é o que permite restringir a domínios específicos sem quebrar o cache.
func resolveOrigin(allowed []string, origin string) string {
	for _, a := range allowed {
		if a == "*" {
			return "*"
		}
		if origin != "" && a == origin {
			return origin
		}
	}
	return ""
}
