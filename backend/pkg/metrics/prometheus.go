package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// LLMTokensTotal tracks total LLM tokens used
	LLMTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "intivai_llm_tokens_total",
			Help: "Total number of LLM tokens used",
		},
		[]string{"model", "type"}, // type can be "prompt" or "completion"
	)

	// WSActiveConnections tracks active WebSockets
	WSActiveConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "intivai_ws_active_connections",
			Help: "Current number of active websocket connections",
		},
		[]string{"type"}, // type can be "chat" or "voice"
	)
)

func init() {
	// Register custom metrics with Prometheus's default registry
	prometheus.MustRegister(LLMTokensTotal)
	prometheus.MustRegister(WSActiveConnections)
}
