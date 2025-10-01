package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Definimos nossas duas métricas como variáveis globais.
// Usamos promauto para registrá-las automaticamente no registro padrão do Prometheus.
var (
	// http_requests_total é um CONTADOR que mede o número de requisições recebidas.
	// As 'labels' nos permitem fatiar os dados (ex: ver requisições só do método GET).
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Número total de requisições HTTP recebidas.",
		},
		[]string{"method", "path", "code"},
	)

	// http_request_duration_seconds é um HISTOGRAMA que mede a latência das requisições.
	// Histogramas são ótimos para calcular percentis (ex: p99, p95).
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP em segundos.",
			Buckets: prometheus.DefBuckets, // Buckets de duração padrão.
		},
		[]string{"method", "path", "code"},
	)
)

// prometheusMiddleware é o nosso middleware que coleta as métricas.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Inicia o timer
		start := time.Now()

		// Usamos um ResponseWriter customizado para capturar o status code da resposta.
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Chama o próximo handler na cadeia (a nossa rota de fato)
		next.ServeHTTP(ww, r)

		// Após a rota ter sido executada, coletamos as métricas.
		duration := time.Since(start).Seconds()
		statusCode := ww.Status()
		
		// Pega o padrão da rota (ex: /usuarios/{id}) para evitar criar métricas para cada ID diferente.
		routePattern := chi.RouteContext(r.Context()).RoutePattern()

		// Incrementa o contador de requisições
		httpRequestsTotal.WithLabelValues(r.Method, routePattern, strconv.Itoa(statusCode)).Inc()

		// Adiciona a observação de duração ao histograma
		httpRequestDuration.WithLabelValues(r.Method, routePattern, strconv.Itoa(statusCode)).Observe(duration)
	})
}