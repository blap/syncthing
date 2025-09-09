// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricDeviceActiveConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "syncthing",
	Subsystem: "connections",
	Name:      "active",
	Help:      "Number of currently active connections, per device. If value is 0, the device is disconnected.",
}, []string{"device"})

var (
	// Connection pool metrics
	metricConnectionPoolSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "pool_size",
		Help:      "Current size of the connection pool for each device.",
	}, []string{"device"})

	metricConnectionPoolCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "pool_created_total",
		Help:      "Total number of connections created in pools.",
	}, []string{"device"})

	metricConnectionPoolReused = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "pool_reused_total",
		Help:      "Total number of connections reused from pools.",
	}, []string{"device"})

	metricConnectionPoolExpired = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "pool_expired_total",
		Help:      "Total number of connections that expired from pools.",
	}, []string{"device"})
	
	metricConnectionMigrationCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "migration_total",
		Help:      "Total number of connection migrations performed.",
	}, []string{"device"})
)

func registerDeviceMetrics(deviceID string) {
	// Register metrics for this device, so that counters & gauges are present even
	// when zero.
	metricDeviceActiveConnections.WithLabelValues(deviceID)
	metricConnectionPoolSize.WithLabelValues(deviceID)
	metricConnectionPoolCreated.WithLabelValues(deviceID)
	metricConnectionPoolReused.WithLabelValues(deviceID)
	metricConnectionPoolExpired.WithLabelValues(deviceID)
	metricConnectionMigrationCount.WithLabelValues(deviceID)
}