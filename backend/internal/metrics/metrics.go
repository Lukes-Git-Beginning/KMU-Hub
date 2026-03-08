package metrics

import (
	"net/http"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
)

// Registry holds all Prometheus metrics and provides HTTP/gRPC instrumentation.
type Registry struct {
	registry *prometheus.Registry

	// HTTP metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight *prometheus.GaugeVec

	// WebSocket metrics
	WSConnectionsActive *prometheus.GaugeVec
	WSMessagesTotal     *prometheus.CounterVec
	WSErrors            *prometheus.CounterVec

	// gRPC metrics
	grpcMetrics *grpcprometheus.ServerMetrics
}

// NewRegistry creates a new metrics registry with HTTP and Go runtime collectors.
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	requestsInFlight := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being served.",
	}, []string{"method"})

	wsConnectionsActive := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ws_connections_active",
		Help: "Number of active WebSocket connections.",
	}, []string{"type"}) // type: "user" or "guest"

	wsMessagesTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_messages_total",
		Help: "Total number of WebSocket messages processed.",
	}, []string{"direction", "message_type"}) // direction: "inbound" or "outbound"

	wsErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_errors_total",
		Help: "Total number of WebSocket errors.",
	}, []string{"error_type"}) // error_type: "auth_failed", "rate_limited", "conn_limit"

	grpcMetrics := grpcprometheus.NewServerMetrics()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		requestsTotal,
		requestDuration,
		requestsInFlight,
		wsConnectionsActive,
		wsMessagesTotal,
		wsErrors,
		grpcMetrics,
	)

	return &Registry{
		registry:            reg,
		RequestsTotal:       requestsTotal,
		RequestDuration:     requestDuration,
		RequestsInFlight:    requestsInFlight,
		WSConnectionsActive: wsConnectionsActive,
		WSMessagesTotal:     wsMessagesTotal,
		WSErrors:            wsErrors,
		grpcMetrics:         grpcMetrics,
	}
}

// Handler returns an HTTP handler for the /metrics endpoint.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

// GRPCUnaryInterceptor returns a gRPC unary server interceptor for metrics.
func (r *Registry) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return r.grpcMetrics.UnaryServerInterceptor()
}

// GRPCStreamInterceptor returns a gRPC stream server interceptor for metrics.
func (r *Registry) GRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return r.grpcMetrics.StreamServerInterceptor()
}

// InitializeGRPCMetrics initializes gRPC metrics after all services are registered.
func (r *Registry) InitializeGRPCMetrics(srv *grpc.Server) {
	r.grpcMetrics.InitializeMetrics(srv)
}
