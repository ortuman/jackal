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

package s2s

import (
	"time"

	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/prometheus/client_golang/prometheus"
)

const reportTotalConnectionsInterval = time.Second * 30

var (
	s2sIncomingConnectionRegistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "incoming_connection_registered",
			Help:      "The total number of incomming connection register operations.",
		},
		[]string{"instance"},
	)
	s2sIncomingConnectionUnregistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "incoming_connection_unregistered",
			Help:      "The total number of incomming connection unregister operations.",
		},
		[]string{"instance"},
	)
	s2sOutgoingConnectionRegistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "outgoing_connection_registered",
			Help:      "The total number of outgoing connection register operations.",
		},
		[]string{"instance", "type"},
	)
	s2sOutgoingConnectionUnregistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "outgoing_connection_unregistered",
			Help:      "The total number of outgoing connection unregister operations.",
		},
		[]string{"instance", "type"},
	)
	s2sOutgoingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "outgoing_requests_total",
			Help:      "The total number of outgoing stanza requests.",
		},
		[]string{"instance", "name", "type"},
	)
	s2sIncomingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "incoming_requests_total",
			Help:      "The total number of incoming stanza requests.",
		},
		[]string{"instance", "name", "type"},
	)
	s2sIncomingRequestDurationBucket = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "incoming_requests_duration_bucket",
			Help:      "Bucketed histogram of incoming stanza requests duration.",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 24),
		},
		[]string{"instance", "name", "type"},
	)
	s2sIncomingTotalConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "incoming_total_connections",
			Help:      "Total S2S incoming connections.",
		},
		[]string{"instance"},
	)
	s2sOutgoingTotalConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "jackal",
			Subsystem: "s2s",
			Name:      "outgoing_total_connections",
			Help:      "Total S2S outgoing connections.",
		},
		[]string{"instance"},
	)
)

func init() {
	prometheus.MustRegister(s2sIncomingConnectionRegistered)
	prometheus.MustRegister(s2sIncomingConnectionUnregistered)
	prometheus.MustRegister(s2sOutgoingConnectionRegistered)
	prometheus.MustRegister(s2sOutgoingConnectionUnregistered)
	prometheus.MustRegister(s2sOutgoingRequests)
	prometheus.MustRegister(s2sIncomingRequests)
	prometheus.MustRegister(s2sIncomingRequestDurationBucket)
	prometheus.MustRegister(s2sIncomingTotalConnections)
	prometheus.MustRegister(s2sOutgoingTotalConnections)
}

func reportIncomingConnectionRegistered() {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	s2sIncomingConnectionRegistered.With(metricLabel).Inc()
}

func reportIncomingConnectionUnregistered() {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	s2sIncomingConnectionUnregistered.With(metricLabel).Inc()
}

func reportOutgoingConnectionRegistered(typ outType) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"type":     typ.String(),
	}
	s2sOutgoingConnectionRegistered.With(metricLabel).Inc()
}

func reportOutgoingConnectionUnregistered(typ outType) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"type":     typ.String(),
	}
	s2sOutgoingConnectionUnregistered.With(metricLabel).Inc()
}

func reportOutgoingRequest(name, typ string) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"name":     name,
		"type":     typ,
	}
	s2sOutgoingRequests.With(metricLabel).Inc()
}

func reportIncomingRequest(name, typ string, durationInSecs float64) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"name":     name,
		"type":     typ,
	}
	s2sIncomingRequests.With(metricLabel).Inc()
	s2sIncomingRequestDurationBucket.With(metricLabel).Observe(durationInSecs)
}

func reportTotalIncomingConnections(totalConns int) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	s2sIncomingTotalConnections.With(metricLabel).Set(float64(totalConns))
}

func reportTotalOutgoingConnections(totalConns int) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	s2sOutgoingTotalConnections.With(metricLabel).Set(float64(totalConns))
}
