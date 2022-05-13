// Copyright 2022 The jackal Authors
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

package dns

import (
	"context"
	"errors"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const resolveTimeout = time.Second * 5

var (
	errBadSRVFormat = errors.New("bad SRV record format")
)

// SRVUpdate represents an SRV resolution update.
type SRVUpdate struct {
	// NewTargets contains the set of new targets compared to previous resolution.
	NewTargets []string

	// OldTargets contains the set of removed targets compared to previous resolution.
	OldTargets []string
}

// SRVResolver performs SRV resolution and optionally runs those in background sending periodical updates.
type SRVResolver struct {
	srv         string
	proto       string
	name        string
	rsvInterval time.Duration

	targetsMu sync.RWMutex
	targets   []string

	updateCh chan SRVUpdate
	doneCh   chan struct{}

	lookUpFn func(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)

	logger log.Logger
}

var srvDialer = net.Dialer{}

// NewSRVResolver creates and initializes a new SRVResolver instance.
func NewSRVResolver(
	service string,
	proto string,
	name string,
	resolveInterval time.Duration,
	logger log.Logger,
) *SRVResolver {
	r := net.Resolver{
		Dial: func(ctx context.Context, _, address string) (net.Conn, error) {
			return srvDialer.DialContext(ctx, "tcp", address) // force SRV resolution over TCP
		},
	}
	return &SRVResolver{
		srv:         service,
		proto:       proto,
		name:        name,
		rsvInterval: resolveInterval,
		updateCh:    make(chan SRVUpdate, 1),
		doneCh:      make(chan struct{}),
		lookUpFn:    r.LookupSRV,
		logger:      logger,
	}
}

// Resolve performs SRV resolution and starts running background updates.
func (r *SRVResolver) Resolve(ctx context.Context) error {
	// first SRV resolution
	_, _, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	if r.rsvInterval == 0 {
		return nil
	}
	go r.runLoop() // update periodically
	return nil
}

// Targets returns last resolved SRV targets.
func (r *SRVResolver) Targets() []string {
	r.targetsMu.RLock()
	defer r.targetsMu.RUnlock()
	return r.targets
}

// Update returns the SRV record updates channel.
func (r *SRVResolver) Update() <-chan SRVUpdate {
	return r.updateCh
}

// Close stops running SRV resolution.
func (r *SRVResolver) Close() {
	close(r.doneCh)
	close(r.updateCh)
}

func (r *SRVResolver) runLoop() {
	tc := time.NewTicker(r.rsvInterval)
	defer tc.Stop()

	for {
		select {
		case <-tc.C:
			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
			newTargets, oldTargets, err := r.resolve(ctx)
			cancel()

			if err != nil {
				level.Warn(r.logger).Log("msg", "failed to resolve SRV record", "err", err)
				continue
			}
			if len(newTargets) == 0 && len(oldTargets) == 0 {
				continue // no updates
			}
			r.updateCh <- SRVUpdate{
				NewTargets: newTargets,
				OldTargets: oldTargets,
			}

		case <-r.doneCh:
			return
		}
	}
}

func (r *SRVResolver) resolve(ctx context.Context) (newTargets []string, oldTargets []string, err error) {
	_, addrs, err := r.lookUpFn(ctx, r.srv, r.proto, r.name)
	if err != nil {
		return nil, nil, err
	}
	var srvTargets []string
	for _, addr := range addrs {
		if addr.Target == "." {
			continue
		}
		host := strings.TrimSuffix(addr.Target, ".")
		port := strconv.Itoa(int(addr.Port))

		srvTargets = append(srvTargets, net.JoinHostPort(host, port))
	}
	// index SRV targets
	srvTargetM := make(map[string]struct{})
	for _, srvTarget := range srvTargets {
		srvTargetM[srvTarget] = struct{}{}
	}

	// compute new and old targets
	r.targetsMu.RLock()
	for _, target := range r.targets {
		_, ok := srvTargetM[target]
		if !ok {
			oldTargets = append(oldTargets, target)
		} else {
			delete(srvTargetM, target)
		}
	}
	r.targetsMu.RUnlock()

	for srvTarget := range srvTargetM {
		newTargets = append(newTargets, srvTarget)
	}

	// update targets
	sort.Slice(srvTargets, func(i, j int) bool {
		return srvTargets[i] < srvTargets[j]
	})
	r.targetsMu.Lock()
	r.targets = srvTargets
	r.targetsMu.Unlock()

	return newTargets, oldTargets, nil
}

// ParseSRVRecord returns the different elements of SRV records by parsing rec string.
func ParseSRVRecord(rec string) (srv, proto, name string, err error) {
	splits := make([]string, 2)

	remaining := rec

	for i := 0; i < 2; i++ {
		idx := strings.Index(remaining, ".")
		if idx == -1 {
			return "", "", "", errBadSRVFormat
		}
		split := remaining[:idx]
		if !strings.HasPrefix(split, "_") {
			return "", "", "", errBadSRVFormat
		}
		splits[i] = split[1:]
		remaining = remaining[idx+1:]
	}
	if len(remaining) == 0 {
		return "", "", "", errBadSRVFormat
	}
	return splits[0], splits[1], remaining, nil
}
