// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0114

import (
	"time"

	"github.com/ortuman/jackal/cluster/instance"
	"github.com/prometheus/client_golang/prometheus"
)

const reportTotalConnectionsInterval = time.Second * 30

var (
	componentConnectionRegistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "ext_component",
			Name:      "connection_registered",
			Help:      "The total number of register operations.",
		},
		[]string{"instance"},
	)
	componentConnectionUnregistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "ext_component",
			Name:      "connection_unregistered",
			Help:      "The total number of unregister operations.",
		},
		[]string{"instance"},
	)
	componentOutgoingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "ext_component",
			Name:      "outgoing_requests_total",
			Help:      "The total number of outgoing stanza requests.",
		},
		[]string{"instance", "name", "type"},
	)
	componentIncomingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "ext_component",
			Name:      "incoming_requests_total",
			Help:      "The total number of incoming stanza requests.",
		},
		[]string{"instance", "name", "type"},
	)
	componentIncomingRequestDurationBucket = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jackal",
			Subsystem: "ext_component",
			Name:      "incoming_requests_duration_bucket",
			Help:      "Bucketed histogram of incoming stanza requests duration.",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 24),
		},
		[]string{"instance", "name", "type"},
	)
	componentIncomingTotalConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "jackal",
			Subsystem: "ext_component",
			Name:      "incoming_total_connections",
			Help:      "Total incoming component connections.",
		},
		[]string{"instance"},
	)
)

func init() {
	prometheus.MustRegister(componentConnectionRegistered)
	prometheus.MustRegister(componentConnectionUnregistered)
	prometheus.MustRegister(componentOutgoingRequests)
	prometheus.MustRegister(componentIncomingRequests)
	prometheus.MustRegister(componentIncomingRequestDurationBucket)
	prometheus.MustRegister(componentIncomingTotalConnections)
}

func reportOutgoingRequest(name, typ string) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"name":     name,
		"type":     typ,
	}
	componentOutgoingRequests.With(metricLabel).Inc()
}

func reportIncomingRequest(name, typ string, durationInSecs float64) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"name":     name,
		"type":     typ,
	}
	componentIncomingRequests.With(metricLabel).Inc()
	componentIncomingRequestDurationBucket.With(metricLabel).Observe(durationInSecs)
}

func reportConnectionRegistered() {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	componentConnectionRegistered.With(metricLabel).Inc()
}

func reportConnectionUnregistered() {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	componentConnectionUnregistered.With(metricLabel).Inc()
}

func reportTotalIncomingConnections(totalConns int) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	componentIncomingTotalConnections.With(metricLabel).Set(float64(totalConns))
}
