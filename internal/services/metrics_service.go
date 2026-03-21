// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package services

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsService holds all Prometheus metrics
type MetricsService struct {
	// Device metrics
	DevicesTotal       *prometheus.GaugeVec
	DevicesOnline      *prometheus.GaugeVec
	DevicesOffline     *prometheus.GaugeVec

	// MQTT metrics
	MQTTConnections    prometheus.Gauge
	MQTTMessagesSent   *prometheus.CounterVec
	MQTTMessagesRecv   *prometheus.CounterVec

	// API metrics
	HTTPRequestsTotal  *prometheus.CounterVec
	HTTPRequestLatency *prometheus.HistogramVec

	// Command metrics
	CommandsTotal      *prometheus.CounterVec
	CommandsSuccess    *prometheus.CounterVec
	CommandsFailed     *prometheus.CounterVec
	CommandsDuration   *prometheus.HistogramVec

	// Tenant metrics
	TenantCount        prometheus.Gauge

	// Alert metrics
	AlertsTotal        *prometheus.CounterVec
	AlertsActive       prometheus.Gauge
}

// NewMetricsService creates a new metrics service
func NewMetricsService() *MetricsService {
	return &MetricsService{
		// Device metrics
		DevicesTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "freekiosk_devices_total",
				Help: "Total number of devices",
			},
			[]string{"tenant_id"},
		),
		DevicesOnline: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "freekiosk_devices_online",
				Help: "Number of online devices",
			},
			[]string{"tenant_id"},
		),
		DevicesOffline: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "freekiosk_devices_offline",
				Help: "Number of offline devices",
			},
			[]string{"tenant_id"},
		),

		// MQTT metrics
		MQTTConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "freekiosk_mqtt_connections",
				Help: "Number of active MQTT connections",
			},
		),
		MQTTMessagesSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_mqtt_messages_sent_total",
				Help: "Total number of MQTT messages sent",
			},
			[]string{"topic"},
		),
		MQTTMessagesRecv: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_mqtt_messages_received_total",
				Help: "Total number of MQTT messages received",
			},
			[]string{"topic"},
		),

		// API metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "freekiosk_http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		// Command metrics
		CommandsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_commands_total",
				Help: "Total number of commands sent",
			},
			[]string{"type", "tenant_id"},
		),
		CommandsSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_commands_success_total",
				Help: "Total number of successful commands",
			},
			[]string{"type"},
		),
		CommandsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_commands_failed_total",
				Help: "Total number of failed commands",
			},
			[]string{"type"},
		),
		CommandsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "freekiosk_command_duration_seconds",
				Help:    "Command execution duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"type"},
		),

		// Tenant metrics
		TenantCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "freekiosk_tenants_total",
				Help: "Total number of tenants",
			},
		),

		// Alert metrics
		AlertsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "freekiosk_alerts_total",
				Help: "Total number of alerts",
			},
			[]string{"type", "severity"},
		),
		AlertsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "freekiosk_alerts_active",
				Help: "Number of active alerts",
			},
		),
	}
}

// RecordDeviceOnline records a device coming online
func (m *MetricsService) RecordDeviceOnline(tenantID string) {
	m.DevicesOnline.WithLabelValues(tenantID).Inc()
	m.DevicesOffline.WithLabelValues(tenantID).Dec()
}

// RecordDeviceOffline records a device going offline
func (m *MetricsService) RecordDeviceOffline(tenantID string) {
	m.DevicesOnline.WithLabelValues(tenantID).Dec()
	m.DevicesOffline.WithLabelValues(tenantID).Inc()
}

// SetDeviceCounts sets the device counts for a tenant
func (m *MetricsService) SetDeviceCounts(tenantID string, total, online, offline int) {
	m.DevicesTotal.WithLabelValues(tenantID).Set(float64(total))
	m.DevicesOnline.WithLabelValues(tenantID).Set(float64(online))
	m.DevicesOffline.WithLabelValues(tenantID).Set(float64(offline))
}

// IncMQTTSent records an MQTT message sent
func (m *MetricsService) IncMQTTSent(topic string) {
	m.MQTTMessagesSent.WithLabelValues(topic).Inc()
}

// IncMQTTRecv records an MQTT message received
func (m *MetricsService) IncMQTTRecv(topic string) {
	m.MQTTMessagesRecv.WithLabelValues(topic).Inc()
}

// SetMQTTConnections sets the number of MQTT connections
func (m *MetricsService) SetMQTTConnections(count float64) {
	m.MQTTConnections.Set(count)
}

// IncHTTPRequest records an HTTP request
func (m *MetricsService) IncHTTPRequest(method, endpoint, status string) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// ObserveHTTPLatency records HTTP request latency
func (m *MetricsService) ObserveHTTPLatency(method, endpoint string, seconds float64) {
	m.HTTPRequestLatency.WithLabelValues(method, endpoint).Observe(seconds)
}

// IncCommand records a command
func (m *MetricsService) IncCommand(cmdType, tenantID string) {
	m.CommandsTotal.WithLabelValues(cmdType, tenantID).Inc()
}

// IncCommandSuccess records a successful command
func (m *MetricsService) IncCommandSuccess(cmdType string) {
	m.CommandsSuccess.WithLabelValues(cmdType).Inc()
}

// IncCommandFailed records a failed command
func (m *MetricsService) IncCommandFailed(cmdType string) {
	m.CommandsFailed.WithLabelValues(cmdType).Inc()
}

// ObserveCommandDuration records command execution duration
func (m *MetricsService) ObserveCommandDuration(cmdType string, seconds float64) {
	m.CommandsDuration.WithLabelValues(cmdType).Observe(seconds)
}

// SetTenantCount sets the total tenant count
func (m *MetricsService) SetTenantCount(count float64) {
	m.TenantCount.Set(count)
}

// IncAlert records an alert
func (m *MetricsService) IncAlert(alertType, severity string) {
	m.AlertsTotal.WithLabelValues(alertType, severity).Inc()
	m.AlertsActive.Inc()
}

// ResolveAlert records an alert resolution
func (m *MetricsService) ResolveAlert() {
	m.AlertsActive.Dec()
}
