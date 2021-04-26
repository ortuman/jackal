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

package measuredrepository

import "github.com/prometheus/client_golang/prometheus"

var (
	repOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "jackal",
			Subsystem: "repository",
			Name:      "operations_total",
			Help:      "The total number of repository operations.",
		},
		[]string{"instance", "type", "success"},
	)
	repOperationDurationBucket = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "jackal",
			Subsystem: "repository",
			Name:      "operations_duration_bucket",
			Help:      "Bucketed histogram of repository operation duration.",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 24),
		},
		[]string{"instance", "type", "success"},
	)
)

func init() {
	prometheus.MustRegister(repOperations)
	prometheus.MustRegister(repOperationDurationBucket)
}
