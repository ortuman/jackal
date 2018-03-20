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

package localrouter

import (
	"time"

	"github.com/ortuman/jackal/cluster/instance"
	"github.com/prometheus/client_golang/prometheus"
)

const reportTotalConnectionsInterval = time.Second * 30

var (
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
	prometheus.MustRegister(c2sIncomingTotalConnections)
}

func reportTotalIncomingConnections(totalConns int) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
	}
	c2sIncomingTotalConnections.With(metricLabel).Set(float64(totalConns))
}
