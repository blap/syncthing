package connections

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Connection stability metrics
	metricConnectionStabilityScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "stability_score",
		Help:      "Connection stability score (0.0 to 1.0) for each device",
	}, []string{"device_id"})

	metricConnectionErrorRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "error_rate",
		Help:      "Connection error rate for each device",
	}, []string{"device_id"})

	metricConnectionLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "latency_seconds",
		Help:      "Connection latency in seconds for each device",
	}, []string{"device_id"})

	// Protocol usage metrics
	metricProtocolUsage = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "protocol_usage_total",
		Help:      "Total number of connections by protocol",
	}, []string{"protocol"})

	// Connection attempt metrics
	metricConnectionAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "attempts_total",
		Help:      "Total number of connection attempts",
	}, []string{"result"}) // success, failure

	metricConnectionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "duration_seconds",
		Help:      "Connection duration in seconds",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 15), // 1s to ~16k seconds
	}, []string{"protocol"})

	// Bandwidth metrics
	metricBandwidthInbound = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "bandwidth_inbound_bytes_total",
		Help:      "Total inbound bandwidth in bytes",
	}, []string{"protocol"})

	metricBandwidthOutbound = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "bandwidth_outbound_bytes_total",
		Help:      "Total outbound bandwidth in bytes",
	}, []string{"protocol"})

	// Connection pool metrics
	metricActiveConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "active_connections",
		Help:      "Number of active connections",
	}, []string{"type"}) // tcp, quic, relay

	// NAT traversal metrics
	metricNATTraversalSuccess = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "nat_traversal_success_total",
		Help:      "Total number of successful NAT traversals",
	}, []string{"method"}) // upnp, stun, relay

	metricNATTraversalFailure = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "nat_traversal_failure_total",
		Help:      "Total number of failed NAT traversals",
	}, []string{"method", "reason"})

	// Connection health metrics
	metricConnectionHealthScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "health_score",
		Help:      "Connection health score (0.0 to 1.0) for each device",
	}, []string{"device_id"})

	// Retry metrics
	metricConnectionRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "retries_total",
		Help:      "Total number of connection retries",
	}, []string{"device_id", "reason"})

	// Timeout metrics
	metricConnectionTimeouts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "timeouts_total",
		Help:      "Total number of connection timeouts",
	}, []string{"type"}) // dial, handshake, idle

	// Error metrics
	metricConnectionErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "errors_total",
		Help:      "Total number of connection errors",
	}, []string{"type", "error"})
)

// ConnectionMetricsTracker tracks connection metrics for enhanced monitoring
type ConnectionMetricsTracker struct {
	// We can add fields here if needed for tracking state
}

// NewConnectionMetricsTracker creates a new connection metrics tracker
func NewConnectionMetricsTracker() *ConnectionMetricsTracker {
	return &ConnectionMetricsTracker{}
}

// RecordConnectionStability records the stability score for a connection
func (cmt *ConnectionMetricsTracker) RecordConnectionStability(deviceID string, score float64) {
	metricConnectionStabilityScore.WithLabelValues(deviceID).Set(score)
}

// RecordConnectionErrorRate records the error rate for a connection
func (cmt *ConnectionMetricsTracker) RecordConnectionErrorRate(deviceID string, rate float64) {
	metricConnectionErrorRate.WithLabelValues(deviceID).Set(rate)
}

// RecordConnectionLatency records the latency for a connection
func (cmt *ConnectionMetricsTracker) RecordConnectionLatency(deviceID string, latency float64) {
	metricConnectionLatency.WithLabelValues(deviceID).Set(latency)
}

// RecordProtocolUsage records usage of a protocol
func (cmt *ConnectionMetricsTracker) RecordProtocolUsage(protocol string) {
	metricProtocolUsage.WithLabelValues(protocol).Inc()
}

// RecordConnectionAttempt records a connection attempt
func (cmt *ConnectionMetricsTracker) RecordConnectionAttempt(result string) {
	metricConnectionAttempts.WithLabelValues(result).Inc()
}

// RecordConnectionDuration records the duration of a connection
func (cmt *ConnectionMetricsTracker) RecordConnectionDuration(protocol string, duration float64) {
	metricConnectionDuration.WithLabelValues(protocol).Observe(duration)
}

// RecordBandwidth records bandwidth usage
func (cmt *ConnectionMetricsTracker) RecordBandwidth(protocol string, inbound, outbound float64) {
	metricBandwidthInbound.WithLabelValues(protocol).Add(inbound)
	metricBandwidthOutbound.WithLabelValues(protocol).Add(outbound)
}

// RecordActiveConnections records the number of active connections
func (cmt *ConnectionMetricsTracker) RecordActiveConnections(connType string, count float64) {
	metricActiveConnections.WithLabelValues(connType).Set(count)
}

// RecordNATTraversal records NAT traversal attempts
func (cmt *ConnectionMetricsTracker) RecordNATTraversal(method, result string) {
	if result == "success" {
		metricNATTraversalSuccess.WithLabelValues(method).Inc()
	} else {
		metricNATTraversalFailure.WithLabelValues(method, result).Inc()
	}
}

// RecordConnectionHealth records the health score for a connection
func (cmt *ConnectionMetricsTracker) RecordConnectionHealth(deviceID string, score float64) {
	metricConnectionHealthScore.WithLabelValues(deviceID).Set(score)
}

// RecordConnectionRetry records a connection retry
func (cmt *ConnectionMetricsTracker) RecordConnectionRetry(deviceID, reason string) {
	metricConnectionRetries.WithLabelValues(deviceID, reason).Inc()
}

// RecordConnectionTimeout records a connection timeout
func (cmt *ConnectionMetricsTracker) RecordConnectionTimeout(timeoutType string) {
	metricConnectionTimeouts.WithLabelValues(timeoutType).Inc()
}

// RecordConnectionError records a connection error
func (cmt *ConnectionMetricsTracker) RecordConnectionError(errorType, error string) {
	metricConnectionErrors.WithLabelValues(errorType, error).Inc()
}