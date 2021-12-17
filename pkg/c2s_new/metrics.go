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

package c2s_new

import (
	"time"

	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/prometheus/client_golang/prometheus"
)

const reportTotalConnectionsInterval = time.Second * 30

var (
	c2sConnectionRegistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "c2s",
			Name:      "connection_registered",
			Help:      "The total number of register operations.",
		},
		[]string{"instance"},
	)
	c2sConnectionUnregistered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "c2s",
			Name:      "connection_unregistered",
			Help:      "The total number of unregister operations.",
		},
		[]string{"instance"},
	)
	c2sOutgoingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "c2s",
			Name:      "outgoing_requests_total",
			Help:      "The total number of outgoing stanza requests.",
		},
		[]string{"instance", "name", "type"},
	)
	c2sIncomingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "c2s",
			Name:      "incoming_requests_total",
			Help:      "The total number of incoming stanza requests.",
		},
		[]string{"instance", "name", "type"},
	)
	c2sIncomingRequestDurationBucket = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jackal",
			Subsystem: "c2s",
			Name:      "incoming_requests_duration_bucket",
			Help:      "Bucketed histogram of incoming stanza requests duration.",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 24),
		},
		[]string{"instance", "name", "type"},
	)
	c2sIncomingTotalConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "jackal",
			Subsystem: "c2s",
			Name:      "incoming_total_connections",
			Help:      "Total incoming C2S connections.",
		},
		[]string{"instance"},
	)
)

func init() {
	prometheus.MustRegister(c2sConnectionRegistered)
	prometheus.MustRegister(c2sConnectionUnregistered)
	prometheus.MustRegister(c2sOutgoingRequests)
	prometheus.MustRegister(c2sIncomingRequests)
	prometheus.MustRegister(c2sIncomingRequestDurationBucket)
	prometheus.MustRegister(c2sIncomingTotalConnections)
}

func reportOutgoingRequest(name, typ string) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"name":     name,
		"type":     typ,
	}
	c2sOutgoingRequests.With(metricLabel).Inc()
}

func reportIncomingRequest(name, typ string, durationInSecs float64) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"name":     name,
		"type":     typ,
	}
	c2sIncomingRequests.With(metricLabel).Inc()
	c2sIncomingRequestDurationBucket.With(metricLabel).Observe(durationInSecs)
}

func reportConnectionRegistered() {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	c2sConnectionRegistered.With(metricLabel).Inc()
}

func reportConnectionUnregistered() {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	c2sConnectionUnregistered.With(metricLabel).Inc()
}

func reportTotalIncomingConnections(totalConns int) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	c2sIncomingTotalConnections.With(metricLabel).Set(float64(totalConns))
}
